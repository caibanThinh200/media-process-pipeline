package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client wraps the S3 SDK client and presigner.
type Client struct {
	s3        *s3.Client
	presigner *s3.PresignClient
	rawBucket string
	outBucket string
}

// New creates a new storage Client using environment variables for bucket names.
func New(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage: load config: %w", err)
	}

	svc := s3.NewFromConfig(cfg)
	return &Client{
		s3:        svc,
		presigner: s3.NewPresignClient(svc),
		rawBucket: os.Getenv("RAW_BUCKET"),
		outBucket: os.Getenv("OUTPUT_BUCKET"),
	}, nil
}

// PresignPutObject returns a presigned PUT URL for the raw bucket.
func (c *Client) PresignPutObject(ctx context.Context, key string, ttl time.Duration) (string, error) {
	req, err := c.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.rawBucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("storage: presign put: %w", err)
	}
	return req.URL, nil
}

// GetObject downloads an object from the raw bucket and returns its body reader.
// Caller is responsible for closing the returned ReadCloser.
func (c *Client) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.rawBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("storage: get object %s: %w", key, err)
	}
	return out.Body, nil
}

// PutOutputObject uploads processed media bytes to the output bucket.
func (c *Client) PutOutputObject(ctx context.Context, key string, body []byte, contentType string) error {
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.outBucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(body),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(int64(len(body))),
	})
	if err != nil {
		return fmt.Errorf("storage: put output object %s: %w", key, err)
	}
	return nil
}
