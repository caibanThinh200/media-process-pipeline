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
	// Read configurable max upload size from environment.
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

// handler is the Lambda entry point for API Gateway HTTP API (v2) events.
func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	method := req.RequestContext.HTTP.Method
	path := req.RawPath

	switch {
	case method == http.MethodPost && path == "/upload/presign":
		return handlePresign(ctx, req)

	case method == http.MethodGet && strings.HasPrefix(path, "/jobs/"):
		jobID := strings.TrimPrefix(path, "/jobs/")
		return handleGetJob(ctx, jobID)

	case method == http.MethodGet && path == "/jobs":
		return handleListJobs(ctx)

	default:
		return jsonResp(http.StatusNotFound, map[string]string{"error": "not found"}), nil
	}
}

// handlePresign validates the request, generates a presigned PUT URL, writes a PENDING job
// record to DynamoDB, and enqueues the job to SQS.
func handlePresign(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var body PresignRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonResp(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"}), nil
	}

	// --- Input validation ---
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
	objectKey := fmt.Sprintf("raw/%s/%s", jobID, body.FileName)
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
	// only after the client completes the presigned PUT upload. Enqueueing here
	// causes a race: the worker would try to GetObject before the file exists.

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
func handleGetJob(ctx context.Context, jobID string) (events.APIGatewayV2HTTPResponse, error) {
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
func handleListJobs(ctx context.Context) (events.APIGatewayV2HTTPResponse, error) {
	jobs, err := dbClient.ListJobs(ctx)
	if err != nil {
		return jsonResp(http.StatusInternalServerError, map[string]string{"error": "failed to list jobs"}), nil
	}
	return jsonResp(http.StatusOK, jobs), nil
}

// jsonResp serialises body to JSON and returns an API Gateway v2 HTTP response.
func jsonResp(statusCode int, body any) events.APIGatewayV2HTTPResponse {
	b, _ := json.Marshal(body)
	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(b),
	}
}

func main() {
	lambda.Start(handler)
}
