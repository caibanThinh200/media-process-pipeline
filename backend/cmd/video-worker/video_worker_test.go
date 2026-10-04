package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/nguyenquocthinh/media-processing-pipeline/internal/models"
)

// fakeDB implements DBClientIface for unit tests.
type fakeDB struct {
	jobs         map[string]*models.Job
	statusCalls  []string
	outputKey    string
	getJobErr    error
	updateErr    error
}

func newFakeDB(jobs map[string]*models.Job) *fakeDB {
	if jobs == nil {
		jobs = make(map[string]*models.Job)
	}
	return &fakeDB{jobs: jobs}
}

func (f *fakeDB) GetJob(_ context.Context, jobID string) (*models.Job, error) {
	if f.getJobErr != nil {
		return nil, f.getJobErr
	}
	return f.jobs[jobID], nil
}

func (f *fakeDB) UpdateJobStatus(_ context.Context, jobID, status string) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.statusCalls = append(f.statusCalls, fmt.Sprintf("%s:%s", jobID, status))
	if job, ok := f.jobs[jobID]; ok {
		job.Status = status
	} else {
		f.jobs[jobID] = &models.Job{JobID: jobID, Status: status}
	}
	return nil
}

func (f *fakeDB) UpdateJobStatusWithOutputKey(_ context.Context, jobID, status, outputKey string) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.statusCalls = append(f.statusCalls, fmt.Sprintf("%s:%s:%s", jobID, status, outputKey))
	f.outputKey = outputKey
	if job, ok := f.jobs[jobID]; ok {
		job.Status = status
		job.OutputKey = outputKey
	} else {
		f.jobs[jobID] = &models.Job{JobID: jobID, Status: status, OutputKey: outputKey}
	}
	return nil
}

// fakeStorage implements StorageClientIface for unit tests.
type fakeStorage struct {
	objects   map[string][]byte
	output    map[string][]byte
	getErr    error
	putErr    error
}

func newFakeStorage(objects map[string][]byte) *fakeStorage {
	if objects == nil {
		objects = make(map[string][]byte)
	}
	return &fakeStorage{objects: objects, output: make(map[string][]byte)}
}

func (f *fakeStorage) GetObject(_ context.Context, key string) (io.ReadCloser, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	data, ok := f.objects[key]
	if !ok {
		return nil, fmt.Errorf("object %q not found", key)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (f *fakeStorage) PutOutputObject(_ context.Context, key string, body []byte, _ string) error {
	if f.putErr != nil {
		return f.putErr
	}
	f.output[key] = body
	return nil
}

// fakeTranscoder implements TranscoderIface for unit tests.
type fakeTranscoder struct {
	transcodeErr error
	calledWith   []string
}

func (f *fakeTranscoder) Transcode(_ context.Context, in, out string) error {
	f.calledWith = append(f.calledWith, fmt.Sprintf("%s->%s", in, out))
	if f.transcodeErr != nil {
		return f.transcodeErr
	}
	// Simulate writing output file
	return nil
}

func TestParseMessage_DirectJobMessage(t *testing.T) {
	body := `{"jobId":"job-v1","objectKey":"raw/videos/job-v1/video.mp4","fileType":"video"}`
	msg, err := parseMessage(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.JobID != "job-v1" {
		t.Errorf("jobId: got %q, want %q", msg.JobID, "job-v1")
	}
	if msg.ObjectKey != "raw/videos/job-v1/video.mp4" {
		t.Errorf("objectKey: got %q, want %q", msg.ObjectKey, "raw/videos/job-v1/video.mp4")
	}
	if msg.FileType != "video" {
		t.Errorf("fileType: got %q, want %q", msg.FileType, "video")
	}
}

func TestParseMessage_S3EventNotification(t *testing.T) {
	body := `{
		"Records": [
			{
				"s3": {
					"bucket": {"name": "test-raw"},
					"object": {
						"key": "raw/videos/job-v2/sample.mp4",
						"urlDecodedKey": "raw/videos/job-v2/sample.mp4"
					}
				}
			}
		]
	}`

	msg, err := parseMessage(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.JobID != "job-v2" {
		t.Errorf("jobId: got %q, want %q", msg.JobID, "job-v2")
	}
	if msg.ObjectKey != "raw/videos/job-v2/sample.mp4" {
		t.Errorf("objectKey: got %q, want %q", msg.ObjectKey, "raw/videos/job-v2/sample.mp4")
	}
}

func TestProcessRecord_SkipDuplicateWhenComplete(t *testing.T) {
	fdb := newFakeDB(map[string]*models.Job{
		"job-done": {JobID: "job-done", Status: models.StatusComplete},
	})
	fst := newFakeStorage(nil)
	ftc := &fakeTranscoder{}

	record := events.SQSMessage{
		Body: `{"jobId":"job-done","objectKey":"raw/videos/job-done/clip.mp4","fileType":"video"}`,
	}

	err := processRecord(context.Background(), record, fdb, fst, ftc)
	if err != nil {
		t.Fatalf("expected nil error on duplicate, got: %v", err)
	}
	if len(ftc.calledWith) != 0 {
		t.Errorf("transcoder should not have been called for duplicate job")
	}
}
