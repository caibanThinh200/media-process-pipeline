package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/nguyenquocthinh/media-processing-pipeline/internal/db"
	"github.com/nguyenquocthinh/media-processing-pipeline/internal/imageproc"
	"github.com/nguyenquocthinh/media-processing-pipeline/internal/models"
	"github.com/nguyenquocthinh/media-processing-pipeline/internal/storage"
)

// optimizeFunc is swappable in tests.
var optimizeFunc = imageproc.Optimize

// loadOptions builds imageproc options from environment variables.
//
//	WATERMARK_ENABLED (default true), WATERMARK_TEXT (default "tAI"),
//	WATERMARK_OPACITY (default 0.5), MAX_DIMENSION (default 1920),
//	WEBP_QUALITY (default 80).
func loadOptions() imageproc.Options {
	opts := imageproc.Options{
		Watermark:        true,
		WatermarkText:    "tAI",
		WatermarkOpacity: imageproc.DefaultOpacity,
	}
	if v := os.Getenv("WATERMARK_ENABLED"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			opts.Watermark = b
		}
	}
	if v := os.Getenv("WATERMARK_TEXT"); v != "" {
		opts.WatermarkText = v
	}
	if v := os.Getenv("WATERMARK_OPACITY"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			opts.WatermarkOpacity = f
		}
	}
	if v := os.Getenv("MAX_DIMENSION"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			opts.MaxDimension = n
		}
	}
	if v := os.Getenv("WEBP_QUALITY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			opts.Quality = n
		}
	}
	return opts
}

// ---------------------------------------------------------------------------
// Interfaces — allow unit tests to inject fakes without AWS credentials.
// ---------------------------------------------------------------------------

// DBClientIface is the subset of db.Client used by the image worker.
type DBClientIface interface {
	GetJob(ctx context.Context, jobID string) (*models.Job, error)
	UpdateJobStatus(ctx context.Context, jobID, status string) error
	UpdateJobStatusWithOutputKey(ctx context.Context, jobID, status, outputKey string) error
}

// StorageClientIface is the subset of storage.Client used by the image worker.
type StorageClientIface interface {
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)
	PutOutputObject(ctx context.Context, key string, body []byte, contentType string) error
}

// ---------------------------------------------------------------------------
// Package-level singletons (initialised once in Lambda cold start).
// ---------------------------------------------------------------------------

var (
	dbClient      DBClientIface
	storageClient StorageClientIface
)

func init() {
	ctx := context.Background()
	var err error

	concreteDB, err := db.New(ctx)
	if err != nil {
		log.Fatalf("failed to init db client: %v", err)
	}
	dbClient = concreteDB

	concreteStorage, err := storage.New(ctx)
	if err != nil {
		log.Fatalf("failed to init storage client: %v", err)
	}
	storageClient = concreteStorage
}

// ---------------------------------------------------------------------------
// Lambda handler
// ---------------------------------------------------------------------------

// handler is the Lambda entry point for SQS batch events.
// Reports partial batch item failures back to SQS.
func handler(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	var batchItemFailures []events.SQSBatchItemFailure
	for _, record := range sqsEvent.Records {
		if err := processRecord(ctx, record, dbClient, storageClient); err != nil {
			log.Printf("messageId=%s error=%v", record.MessageId, err)
			batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
		}
	}
	return events.SQSEventResponse{
		BatchItemFailures: batchItemFailures,
	}, nil
}

// ---------------------------------------------------------------------------
// Message parsing
// ---------------------------------------------------------------------------

// parseMessage extracts a JobMessage from an SQS record body.
// The body can be either:
//  1. A direct JobMessage JSON (sent by the upload-api)
//  2. An S3 event notification JSON (sent by the S3→SQS bucket notification)
//
// For S3 events, the jobId is embedded in the object key: raw/<jobId>/<filename>
func parseMessage(body string) (*models.JobMessage, error) {
	// Try direct JobMessage first.
	var msg models.JobMessage
	if err := json.Unmarshal([]byte(body), &msg); err == nil && msg.JobID != "" {
		return &msg, nil
	}

	// Try S3 event notification format.
	var s3Event events.S3Event
	if err := json.Unmarshal([]byte(body), &s3Event); err != nil {
		return nil, fmt.Errorf("unrecognised message format: %w", err)
	}

	if len(s3Event.Records) == 0 {
		return nil, fmt.Errorf("s3 event has no records")
	}

	rec := s3Event.Records[0]
	objectKey := rec.S3.Object.URLDecodedKey
	if objectKey == "" {
		objectKey = rec.S3.Object.Key
	}

	// objectKey is raw/<jobId>/<filename> — extract jobId from the second segment.
	// path.Dir("raw/abc-123/file.jpg") → "raw/abc-123"
	// path.Base("raw/abc-123") → "abc-123"
	jobID := path.Base(path.Dir(objectKey))
	if jobID == "" || jobID == "." || jobID == "raw" {
		return nil, fmt.Errorf("could not extract jobId from object key %q", objectKey)
	}

	return &models.JobMessage{
		JobID:     jobID,
		ObjectKey: objectKey,
		FileType:  "image", // S3 events don't carry fileType; default to image for now
	}, nil
}

// ---------------------------------------------------------------------------
// Core processing logic (accepts interfaces for testability)
// ---------------------------------------------------------------------------

func processRecord(ctx context.Context, record events.SQSMessage, dbc DBClientIface, sc StorageClientIface) error {
	msg, err := parseMessage(record.Body)
	if err != nil {
		// Malformed message — return error so it retries and hits DLQ.
		return fmt.Errorf("parse message: %w", err)
	}

	// --- Idempotency check ---
	// If the job is already COMPLETE or FAILED, skip silently.
	// This handles duplicate SQS delivery (at-least-once semantics).
	existing, err := dbc.GetJob(ctx, msg.JobID)
	if err != nil {
		return fmt.Errorf("idempotency check getJob %s: %w", msg.JobID, err)
	}
	if existing != nil && (existing.Status == models.StatusComplete || existing.Status == models.StatusFailed) {
		log.Printf("jobId=%s status=%s workerType=image action=skip_duplicate", msg.JobID, existing.Status)
		return nil
	}

	// --- PROCESSING ---
	if err := dbc.UpdateJobStatus(ctx, msg.JobID, models.StatusProcessing); err != nil {
		return fmt.Errorf("set PROCESSING %s: %w", msg.JobID, err)
	}
	log.Printf("jobId=%s objectKey=%s workerType=image status=PROCESSING", msg.JobID, msg.ObjectKey)

	// --- Download raw object ---
	body, err := sc.GetObject(ctx, msg.ObjectKey)
	if err != nil {
		_ = dbc.UpdateJobStatus(ctx, msg.JobID, models.StatusFailed)
		return fmt.Errorf("getObject %s: %w", msg.ObjectKey, err)
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		_ = dbc.UpdateJobStatus(ctx, msg.JobID, models.StatusFailed)
		return fmt.Errorf("read body %s: %w", msg.ObjectKey, err)
	}

	// --- Optimize: orient, resize, watermark, encode as WebP ---
	filename := path.Base(msg.ObjectKey)
	outData, contentType, optErr := optimizeFunc(data, loadOptions())
	if optErr != nil {
		// Unsupported/corrupt image: fall back to a passthrough copy.
		log.Printf("jobId=%s workerType=image action=passthrough_fallback error=%v", msg.JobID, optErr)
		outData = data
		contentType = "application/octet-stream"
	} else {
		filename = strings.TrimSuffix(filename, path.Ext(filename)) + ".webp"
	}
	outputKey := fmt.Sprintf("output/%s/%s", msg.JobID, filename)

	if err := sc.PutOutputObject(ctx, outputKey, outData, contentType); err != nil {
		_ = dbc.UpdateJobStatus(ctx, msg.JobID, models.StatusFailed)
		return fmt.Errorf("putOutputObject %s: %w", outputKey, err)
	}

	// --- COMPLETE ---
	if err := dbc.UpdateJobStatusWithOutputKey(ctx, msg.JobID, models.StatusComplete, outputKey); err != nil {
		return fmt.Errorf("set COMPLETE %s: %w", msg.JobID, err)
	}

	log.Printf("jobId=%s objectKey=%s outputKey=%s workerType=image status=COMPLETE origBytes=%d outBytes=%d ratio=%.2f contentType=%s",
		msg.JobID, msg.ObjectKey, outputKey, len(data), len(outData), float64(len(outData))/float64(len(data)), contentType)

	return nil
}

func main() {
	lambda.Start(handler)
}
