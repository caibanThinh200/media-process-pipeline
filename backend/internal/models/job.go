package models

// Job status lifecycle constants.
const (
	StatusPending    = "PENDING"
	StatusUploaded   = "UPLOADED"
	StatusProcessing = "PROCESSING"
	StatusComplete   = "COMPLETE"
	StatusFailed     = "FAILED"
)

// Job represents a media processing job record stored in DynamoDB.
type Job struct {
	JobID     string `dynamodbav:"jobId"     json:"jobId"`
	Status    string `dynamodbav:"status"    json:"status"`
	FileType  string `dynamodbav:"fileType"  json:"fileType"`  // "image" | "video"
	ObjectKey string `dynamodbav:"objectKey" json:"objectKey"` // raw S3 key
	OutputKey string `dynamodbav:"outputKey" json:"outputKey"` // processed S3 key
	CreatedAt string `dynamodbav:"createdAt" json:"createdAt"` // RFC3339
	UpdatedAt string `dynamodbav:"updatedAt" json:"updatedAt"` // RFC3339
	ExpiresAt int64  `dynamodbav:"expiresAt" json:"expiresAt"` // Unix epoch for TTL
}

// JobMessage is the payload published to SQS.
type JobMessage struct {
	JobID     string `json:"jobId"`
	ObjectKey string `json:"objectKey"`
	FileType  string `json:"fileType"`
}
