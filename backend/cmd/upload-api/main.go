package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"

	"github.com/nguyenquocthinh/media-processing-pipeline/internal/db"
	"github.com/nguyenquocthinh/media-processing-pipeline/internal/models"
	"github.com/nguyenquocthinh/media-processing-pipeline/internal/queue"
	"github.com/nguyenquocthinh/media-processing-pipeline/internal/storage"
)

// validFileTypes are the only accepted fileType values.
var validFileTypes = map[string]bool{
	"image": true,
	"video": true,
}

// maxUploadBytes is the default max upload size (500 MB). Override via MAX_UPLOAD_BYTES env var.
var maxUploadBytes int64 = 524_288_000

// jobTTLDays is how long job records live in DynamoDB before TTL removes them.
const jobTTLDays = 30

// presignTTL is how long the presigned PUT URL remains valid.
const presignTTL = 15 * time.Minute

type PresignRequest struct {
	FileName  string `json:"fileName"`
	FileType  string `json:"fileType"`  // "image" | "video"
	MediaType string `json:"mediaType"` // MIME type
	SizeBytes int64  `json:"sizeBytes"`
}

type PresignResponse struct {
	JobID        string `json:"jobId"`
	UploadURL    string `json:"uploadUrl"`
	ObjectKey    string `json:"objectKey"`
	ExpiresInSec int    `json:"expiresInSec"`
}

var (
	storageClient *storage.Client
	dbClient      *db.Client
	queueClient   *queue.Client
)

func init() {
	if v := os.Getenv("MAX_UPLOAD_BYTES"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			maxUploadBytes = n
		}
	}

	ctx := context.Background()
	var err error

	storageClient, err = storage.New(ctx)
	if err != nil {
		log.Fatalf("failed to init storage client: %v", err)
	}

	dbClient, err = db.New(ctx)
	if err != nil {
		log.Fatalf("failed to init db client: %v", err)
	}

	queueClient, err = queue.New(ctx)
	if err != nil {
		log.Fatalf("failed to init queue client: %v", err)
	}
}

// normalizedRequest is a source-agnostic view of an incoming Lambda invocation,
// populated from either an API Gateway v2 or ALB event.
type normalizedRequest struct {
	method string
	path   string
	body   string
}

// normalizeEvent detects the event source and extracts method, path, and body.
// API Gateway v2 events have a "requestContext.http" field.
// ALB events have a top-level "httpMethod" field.
func normalizeEvent(raw json.RawMessage) (normalizedRequest, error) {
	// Probe for ALB event shape (has top-level "httpMethod")
	var probe struct {
		HTTPMethod string `json:"httpMethod"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return normalizedRequest{}, fmt.Errorf("unmarshal probe: %w", err)
	}

	if probe.HTTPMethod != "" {
		// ALB target group event
		var albReq events.ALBTargetGroupRequest
		if err := json.Unmarshal(raw, &albReq); err != nil {
			return normalizedRequest{}, fmt.Errorf("unmarshal ALB event: %w", err)
		}
		return normalizedRequest{
			method: albReq.HTTPMethod,
			path:   albReq.Path,
			body:   albReq.Body,
		}, nil
	}

	// API Gateway HTTP API v2 event
	var apigwReq events.APIGatewayV2HTTPRequest
	if err := json.Unmarshal(raw, &apigwReq); err != nil {
		return normalizedRequest{}, fmt.Errorf("unmarshal APIGW v2 event: %w", err)
	}
	return normalizedRequest{
		method: apigwReq.RequestContext.HTTP.Method,
		path:   apigwReq.RawPath,
		body:   apigwReq.Body,
	}, nil
}

// handler is the Lambda entry point. It accepts a raw JSON event, detects the source
// (API Gateway v2 or ALB), normalizes it, and routes to the appropriate handler.
// Returns events.ALBTargetGroupResponse — its JSON shape (statusCode, headers, body,
// isBase64Encoded) is read correctly by both ALB and API Gateway v2.
func handler(ctx context.Context, raw json.RawMessage) (events.ALBTargetGroupResponse, error) {
	nr, err := normalizeEvent(raw)
	if err != nil {
		log.Printf("normalizeEvent error: %v", err)
		return jsonResp(http.StatusBadRequest, map[string]string{"error": "bad request"}), nil
	}

	log.Printf("method=%s path=%s", nr.method, nr.path)

	switch {
	case nr.method == http.MethodPost && nr.path == "/upload/presign":
		return handlePresign(ctx, nr)

	case nr.method == http.MethodGet && strings.HasPrefix(nr.path, "/jobs/"):
		jobID := strings.TrimPrefix(nr.path, "/jobs/")
		return handleGetJob(ctx, jobID)

	case nr.method == http.MethodGet && nr.path == "/jobs":
		return handleListJobs(ctx)

	case nr.method == http.MethodGet && nr.path == "/health":
		return jsonResp(http.StatusOK, map[string]string{"status": "ok"}), nil

	default:
		return jsonResp(http.StatusNotFound, map[string]string{"error": "not found"}), nil
	}
}

// handlePresign validates the request, generates a presigned PUT URL, and writes a PENDING job record.
func handlePresign(ctx context.Context, nr normalizedRequest) (events.ALBTargetGroupResponse, error) {
	var body PresignRequest
	if err := json.Unmarshal([]byte(nr.body), &body); err != nil {
		return jsonResp(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"}), nil
	}

	if strings.TrimSpace(body.FileName) == "" {
		return jsonResp(http.StatusBadRequest, map[string]string{"error": "fileName is required"}), nil
	}
	if !validFileTypes[body.FileType] {
		return jsonResp(http.StatusBadRequest, map[string]string{"error": "fileType must be 'image' or 'video'"}), nil
	}
	if body.SizeBytes <= 0 {
		return jsonResp(http.StatusBadRequest, map[string]string{"error": "sizeBytes must be a positive integer"}), nil
	}
	if body.SizeBytes > maxUploadBytes {
		return jsonResp(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("file size %d exceeds maximum allowed %d bytes", body.SizeBytes, maxUploadBytes),
		}), nil
	}

	jobID := uuid.NewString()
	var objectKey string
	if body.FileType == "video" {
		objectKey = fmt.Sprintf("raw/videos/%s/%s", jobID, body.FileName)
	} else {
		objectKey = fmt.Sprintf("raw/images/%s/%s", jobID, body.FileName)
	}
	now := time.Now().UTC()

	presignURL, err := storageClient.PresignPutObject(ctx, objectKey, presignTTL)
	if err != nil {
		log.Printf("jobId=%s presign_error=%v", jobID, err)
		return jsonResp(http.StatusInternalServerError, map[string]string{"error": "failed to generate upload URL"}), nil
	}

	job := &models.Job{
		JobID:     jobID,
		Status:    models.StatusPending,
		FileType:  body.FileType,
		ObjectKey: objectKey,
		CreatedAt: now.Format(time.RFC3339),
		UpdatedAt: now.Format(time.RFC3339),
		ExpiresAt: now.AddDate(0, 0, jobTTLDays).Unix(),
	}

	if err := dbClient.PutJob(ctx, job); err != nil {
		log.Printf("jobId=%s db_put_error=%v", jobID, err)
		return jsonResp(http.StatusInternalServerError, map[string]string{"error": "failed to create job"}), nil
	}

	// NOTE: Do NOT enqueue to SQS here. The image-worker is triggered by the
	// S3 event notification on the raw bucket (raw/ prefix PUT), which fires
	// only after the client completes the presigned PUT upload.

	log.Printf("jobId=%s objectKey=%s fileType=%s sizeBytes=%d status=PENDING presign_ok=true",
		jobID, objectKey, body.FileType, body.SizeBytes)

	return jsonResp(http.StatusOK, PresignResponse{
		JobID:        jobID,
		UploadURL:    presignURL,
		ObjectKey:    objectKey,
		ExpiresInSec: int(presignTTL.Seconds()),
	}), nil
}

// handleGetJob looks up a single job by ID — used by the frontend polling loop.
func handleGetJob(ctx context.Context, jobID string) (events.ALBTargetGroupResponse, error) {
	if strings.TrimSpace(jobID) == "" {
		return jsonResp(http.StatusBadRequest, map[string]string{"error": "jobId is required"}), nil
	}

	job, err := dbClient.GetJob(ctx, jobID)
	if err != nil {
		log.Printf("jobId=%s get_job_error=%v", jobID, err)
		return jsonResp(http.StatusInternalServerError, map[string]string{"error": "failed to fetch job"}), nil
	}
	if job == nil {
		return jsonResp(http.StatusNotFound, map[string]string{"error": "job not found"}), nil
	}

	return jsonResp(http.StatusOK, job), nil
}

// handleListJobs returns all jobs — for dev/debug use only (full table scan).
func handleListJobs(ctx context.Context) (events.ALBTargetGroupResponse, error) {
	jobs, err := dbClient.ListJobs(ctx)
	if err != nil {
		return jsonResp(http.StatusInternalServerError, map[string]string{"error": "failed to list jobs"}), nil
	}
	return jsonResp(http.StatusOK, jobs), nil
}

// jsonResp builds a response compatible with both ALB and API Gateway v2.
// ALBTargetGroupResponse is accepted by both invocation sources.
func jsonResp(statusCode int, body any) events.ALBTargetGroupResponse {
	b, _ := json.Marshal(body)
	return events.ALBTargetGroupResponse{
		StatusCode:        statusCode,
		StatusDescription: http.StatusText(statusCode),
		Headers:           map[string]string{"Content-Type": "application/json"},
		Body:              string(b),
		IsBase64Encoded:   false,
	}
}

func main() {
	lambda.Start(handler)
}
