package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/nguyenquocthinh/media-processing-pipeline/internal/models"
)

// Client wraps the SQS SDK client.
type Client struct {
	sqs      *sqs.Client
	queueURL string
}

// New creates a new queue Client using the QUEUE_URL environment variable.
func New(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("queue: load config: %w", err)
	}

	return &Client{
		sqs:      sqs.NewFromConfig(cfg),
		queueURL: os.Getenv("QUEUE_URL"),
	}, nil
}

// SendJob serialises a JobMessage and sends it to SQS.
func (c *Client) SendJob(ctx context.Context, job *models.Job) error {
	msg := models.JobMessage{
		JobID:     job.JobID,
		ObjectKey: job.ObjectKey,
		FileType:  job.FileType,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("queue: marshal message: %w", err)
	}

	_, err = c.sqs.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(c.queueURL),
		MessageBody: aws.String(string(body)),
	})
	if err != nil {
		return fmt.Errorf("queue: send message: %w", err)
	}

	return nil
}
