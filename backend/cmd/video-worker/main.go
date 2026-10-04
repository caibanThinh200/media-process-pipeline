package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/nguyenquocthinh/media-processing-pipeline/internal/db"
	"github.com/nguyenquocthinh/media-processing-pipeline/internal/models"
	"github.com/nguyenquocthinh/media-processing-pipeline/internal/storage"
)

// ---------------------------------------------------------------------------
// Interfaces — allow unit tests to inject fakes without AWS credentials.
// ---------------------------------------------------------------------------

type DBClientIface interface {
	GetJob(ctx context.Context, jobID string) (*models.Job, error)
	UpdateJobStatus(ctx context.Context, jobID, status string) error
	UpdateJobStatusWithOutputKey(ctx context.Context, jobID, status, outputKey string) error
}

type StorageClientIface interface {
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)
	PutOutputObject(ctx context.Context, key string, body []byte, contentType string) error
}

type TranscoderIface interface {
	Transcode(ctx context.Context, inputPath, outputPath string) error
}

// ---------------------------------------------------------------------------
// Package-level singletons (initialised once in Lambda cold start).
// ---------------------------------------------------------------------------

var (
	dbClient      DBClientIface
	storageClient StorageClientIface
	transcoder    TranscoderIface
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

	transcoder = NewFFmpegTranscoder()
}

// ---------------------------------------------------------------------------
// Lambda handler
// ---------------------------------------------------------------------------

func handler(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	var batchItemFailures []events.SQSBatchItemFailure
	for _, record := range sqsEvent.Records {
		if err := processRecord(ctx, record, dbClient, storageClient, transcoder); err != nil {
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

func parseMessage(body string) (*models.JobMessage, error) {
	// Try direct JobMessage JSON
	var msg models.JobMessage
	if err := json.Unmarshal([]byte(body), &msg); err == nil && msg.JobID != "" {
		return &msg, nil
	}

	// Try S3 event notification format
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

	// objectKey is raw/videos/<jobId>/<filename>
	// path.Dir("raw/videos/abc-123/clip.mp4") -> "raw/videos/abc-123"
	// path.Base("raw/videos/abc-123") -> "abc-123"
	jobID := path.Base(path.Dir(objectKey))
	if jobID == "" || jobID == "." || jobID == "videos" || jobID == "raw" {
		return nil, fmt.Errorf("could not extract jobId from object key %q", objectKey)
	}

	return &models.JobMessage{
		JobID:     jobID,
		ObjectKey: objectKey,
		FileType:  "video",
	}, nil
}

// ---------------------------------------------------------------------------
// Core processing logic
// ---------------------------------------------------------------------------

func processRecord(
	ctx context.Context,
	record events.SQSMessage,
	dbc DBClientIface,
	sc StorageClientIface,
	tc TranscoderIface,
) error {
	msg, err := parseMessage(record.Body)
	if err != nil {
		return fmt.Errorf("parse message: %w", err)
	}

	// --- Idempotency check ---
	existing, err := dbc.GetJob(ctx, msg.JobID)
	if err != nil {
		return fmt.Errorf("idempotency check getJob %s: %w", msg.JobID, err)
	}
	if existing != nil && (existing.Status == models.StatusComplete || existing.Status == models.StatusFailed) {
		log.Printf("jobId=%s status=%s workerType=video action=skip_duplicate", msg.JobID, existing.Status)
		return nil
	}

	// --- PROCESSING ---
	if err := dbc.UpdateJobStatus(ctx, msg.JobID, models.StatusProcessing); err != nil {
		return fmt.Errorf("set PROCESSING %s: %w", msg.JobID, err)
	}
	log.Printf("jobId=%s objectKey=%s workerType=video status=PROCESSING", msg.JobID, msg.ObjectKey)

	// --- Download raw video from S3 to /tmp ---
	body, err := sc.GetObject(ctx, msg.ObjectKey)
	if err != nil {
		_ = dbc.UpdateJobStatus(ctx, msg.JobID, models.StatusFailed)
		return fmt.Errorf("getObject %s: %w", msg.ObjectKey, err)
	}
	defer body.Close()

	rawFilename := path.Base(msg.ObjectKey)
	ext := filepath.Ext(rawFilename)
	if ext == "" {
		ext = ".mp4"
	}

	tmpInput := filepath.Join("/tmp", fmt.Sprintf("in_%s%s", msg.JobID, ext))
	tmpOutput := filepath.Join("/tmp", fmt.Sprintf("out_%s.mp4", msg.JobID))
	defer os.Remove(tmpInput)
	defer os.Remove(tmpOutput)

	inFile, err := os.Create(tmpInput)
	if err != nil {
		_ = dbc.UpdateJobStatus(ctx, msg.JobID, models.StatusFailed)
		return fmt.Errorf("create temp input file: %w", err)
	}

	if _, err := io.Copy(inFile, body); err != nil {
		inFile.Close()
		_ = dbc.UpdateJobStatus(ctx, msg.JobID, models.StatusFailed)
		return fmt.Errorf("write temp input file: %w", err)
	}
	inFile.Close()

	// --- Run FFmpeg Transcoding ---
	log.Printf("jobId=%s starting transcode input=%s output=%s", msg.JobID, tmpInput, tmpOutput)
	if err := tc.Transcode(ctx, tmpInput, tmpOutput); err != nil {
		_ = dbc.UpdateJobStatus(ctx, msg.JobID, models.StatusFailed)
		return fmt.Errorf("transcode %s: %w", msg.JobID, err)
	}

	// --- Read and Upload Transcoded Video ---
	outData, err := os.ReadFile(tmpOutput)
	if err != nil {
		_ = dbc.UpdateJobStatus(ctx, msg.JobID, models.StatusFailed)
		return fmt.Errorf("read transcode output: %w", err)
	}

	baseName := strings.TrimSuffix(rawFilename, filepath.Ext(rawFilename))
	outputKey := fmt.Sprintf("output/%s/%s.mp4", msg.JobID, baseName)

	if err := sc.PutOutputObject(ctx, outputKey, outData, "video/mp4"); err != nil {
		_ = dbc.UpdateJobStatus(ctx, msg.JobID, models.StatusFailed)
		return fmt.Errorf("putOutputObject %s: %w", outputKey, err)
	}

	// --- COMPLETE ---
	if err := dbc.UpdateJobStatusWithOutputKey(ctx, msg.JobID, models.StatusComplete, outputKey); err != nil {
		return fmt.Errorf("set COMPLETE %s: %w", msg.JobID, err)
	}

	log.Printf("jobId=%s objectKey=%s outputKey=%s workerType=video status=COMPLETE bytes=%d",
		msg.JobID, msg.ObjectKey, outputKey, len(outData))

	return nil
}

func main() {
	lambda.Start(handler)
}
