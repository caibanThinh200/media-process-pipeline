package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/nguyenquocthinh/media-processing-pipeline/internal/db"
	"github.com/nguyenquocthinh/media-processing-pipeline/internal/models"
	"github.com/nguyenquocthinh/media-processing-pipeline/internal/storage"
)

var (
	dbClient      *db.Client
	storageClient *storage.Client
)

func init() {
	ctx := context.Background()
	var err error

	dbClient, err = db.New(ctx)
	if err != nil {
		log.Fatalf("failed to init db client: %v", err)
	}

	storageClient, err = storage.New(ctx)
	if err != nil {
		log.Fatalf("failed to init storage client: %v", err)
	}
}

// SQSJobMessage is the payload sent by the upload-api onto the queue.
type SQSJobMessage struct {
	JobID     string `json:"jobId"`
	ObjectKey string `json:"objectKey"`
	FileType  string `json:"fileType"`
}

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	for _, record := range sqsEvent.Records {
		if err := processRecord(ctx, record); err != nil {
			log.Printf("error processing record %s: %v", record.MessageId, err)
			return err
		}
	}
	return nil
}

func processRecord(ctx context.Context, record events.SQSMessage) error {
	var msg SQSJobMessage
	if err := json.Unmarshal([]byte(record.Body), &msg); err != nil {
		log.Printf("unmarshal error: %v", err)
		return err
	}

	if err := dbClient.UpdateJobStatus(ctx, msg.JobID, models.StatusProcessing); err != nil {
		return err
	}

	// TODO: download from S3, run FFmpeg transcode, generate thumbnails, upload results
	log.Printf("video-worker: processing job %s (key=%s)", msg.JobID, msg.ObjectKey)

	if err := dbClient.UpdateJobStatus(ctx, msg.JobID, models.StatusComplete); err != nil {
		return err
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
