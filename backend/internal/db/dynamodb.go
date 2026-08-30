package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/nguyenquocthinh/media-processing-pipeline/internal/models"
)

// Client wraps the DynamoDB SDK client.
type Client struct {
	ddb       *dynamodb.Client
	tableName string
}

// New creates a new db Client using the DYNAMODB_TABLE environment variable.
func New(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("db: load config: %w", err)
	}

	return &Client{
		ddb:       dynamodb.NewFromConfig(cfg),
		tableName: os.Getenv("DYNAMODB_TABLE"),
	}, nil
}

// PutJob writes a new job record to DynamoDB.
// Uses attribute_not_exists(jobId) so duplicate calls are safe.
func (c *Client) PutJob(ctx context.Context, job *models.Job) error {
	item, err := attributevalue.MarshalMap(job)
	if err != nil {
		return fmt.Errorf("db: marshal job: %w", err)
	}

	_, err = c.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(c.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(jobId)"),
	})
	if err != nil {
		return fmt.Errorf("db: put job %s: %w", job.JobID, err)
	}
	return nil
}

// GetJob retrieves a single job by its ID.
// Returns nil, nil if the item does not exist.
func (c *Client) GetJob(ctx context.Context, jobID string) (*models.Job, error) {
	out, err := c.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(c.tableName),
		Key: map[string]types.AttributeValue{
			"jobId": &types.AttributeValueMemberS{Value: jobID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("db: get job %s: %w", jobID, err)
	}
	if out.Item == nil {
		return nil, nil
	}

	var job models.Job
	if err := attributevalue.UnmarshalMap(out.Item, &job); err != nil {
		return nil, fmt.Errorf("db: unmarshal job %s: %w", jobID, err)
	}
	return &job, nil
}

// UpdateJobStatus transitions a job to a new status and updates the updatedAt timestamp.
func (c *Client) UpdateJobStatus(ctx context.Context, jobID, status string) error {
	_, err := c.ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(c.tableName),
		Key: map[string]types.AttributeValue{
			"jobId": &types.AttributeValueMemberS{Value: jobID},
		},
		UpdateExpression: aws.String("SET #s = :s, updatedAt = :u"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":s": &types.AttributeValueMemberS{Value: status},
			":u": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
		},
	})
	if err != nil {
		return fmt.Errorf("db: update job %s status to %s: %w", jobID, status, err)
	}
	return nil
}

// UpdateJobStatusWithOutputKey atomically sets status, outputKey, and updatedAt in one call.
// Used by workers to mark a job COMPLETE and record the processed S3 key.
func (c *Client) UpdateJobStatusWithOutputKey(ctx context.Context, jobID, status, outputKey string) error {
	_, err := c.ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(c.tableName),
		Key: map[string]types.AttributeValue{
			"jobId": &types.AttributeValueMemberS{Value: jobID},
		},
		UpdateExpression: aws.String("SET #s = :s, outputKey = :ok, updatedAt = :u"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":s":  &types.AttributeValueMemberS{Value: status},
			":ok": &types.AttributeValueMemberS{Value: outputKey},
			":u":  &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
		},
	})
	if err != nil {
		return fmt.Errorf("db: update job %s status+outputKey: %w", jobID, err)
	}
	return nil
}

// ListJobs returns all jobs via a full table scan. Suitable for dev / small datasets only.
func (c *Client) ListJobs(ctx context.Context) ([]models.Job, error) {
	out, err := c.ddb.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(c.tableName),
	})
	if err != nil {
		return nil, fmt.Errorf("db: scan jobs: %w", err)
	}

	var jobs []models.Job
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &jobs); err != nil {
		return nil, fmt.Errorf("db: unmarshal jobs: %w", err)
	}
	return jobs, nil
}
