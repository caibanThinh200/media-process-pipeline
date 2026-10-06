package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"

	"github.com/nguyenquocthinh/media-processing-pipeline/internal/models"
)

// ---------------------------------------------------------------------------
// Fakes (in-process, no AWS)
// ---------------------------------------------------------------------------

// fakeDB satisfies DBClientIface.
// jobs map is keyed by jobId.
// statusHistory records every UpdateJobStatus call in order.
type fakeDB struct {
	jobs          map[string]*models.Job
	statusHistory []string // sequence of statuses written
	forceGetErr   error
	forceUpdateErr error
}

func newFakeDB(jobs ...*models.Job) *fakeDB {
	f := &fakeDB{jobs: make(map[string]*models.Job)}
	for _, j := range jobs {
		jCopy := *j
		f.jobs[j.JobID] = &jCopy
	}
	return f
}

func (f *fakeDB) GetJob(_ context.Context, jobID string) (*models.Job, error) {
	if f.forceGetErr != nil {
		return nil, f.forceGetErr
	}
	j, ok := f.jobs[jobID]
	if !ok {
		return nil, nil
	}
	jCopy := *j
	return &jCopy, nil
}

func (f *fakeDB) UpdateJobStatus(_ context.Context, jobID, status string) error {
	if f.forceUpdateErr != nil {
		return f.forceUpdateErr
	}
	f.statusHistory = append(f.statusHistory, status)
	if j, ok := f.jobs[jobID]; ok {
		j.Status = status
	} else {
		f.jobs[jobID] = &models.Job{JobID: jobID, Status: status}
	}
	return nil
}

func (f *fakeDB) UpdateJobStatusWithOutputKey(_ context.Context, jobID, status, outputKey string) error {
	if f.forceUpdateErr != nil {
		return f.forceUpdateErr
	}
	f.statusHistory = append(f.statusHistory, status)
	if j, ok := f.jobs[jobID]; ok {
		j.Status = status
		j.OutputKey = outputKey
	} else {
		f.jobs[jobID] = &models.Job{JobID: jobID, Status: status, OutputKey: outputKey}
	}
	return nil
}

// fakeStorage satisfies StorageClientIface.
type fakeStorage struct {
	objects      map[string][]byte // objectKey → bytes
	written      map[string][]byte // outputKey → bytes written by PutOutputObject
	contentTypes map[string]string // outputKey → content type
	forceGetErr  error
	forcePutErr  error
}

func newFakeStorage(objects map[string][]byte) *fakeStorage {
	return &fakeStorage{
		objects:      objects,
		written:      make(map[string][]byte),
		contentTypes: make(map[string]string),
	}
}

func (f *fakeStorage) GetObject(_ context.Context, key string) (io.ReadCloser, error) {
	if f.forceGetErr != nil {
		return nil, f.forceGetErr
	}
	data, ok := f.objects[key]
	if !ok {
		return nil, fmt.Errorf("object not found: %s", key)
	}
	return io.NopCloser(strings.NewReader(string(data))), nil
}

func (f *fakeStorage) PutOutputObject(_ context.Context, key string, body []byte, contentType string) error {
	if f.forcePutErr != nil {
		return f.forcePutErr
	}
	f.written[key] = body
	f.contentTypes[key] = contentType
	return nil
}

// ---------------------------------------------------------------------------
// Helper: build a minimal SQS record
// ---------------------------------------------------------------------------

func sqsRecord(body string) events.SQSMessage {
	return events.SQSMessage{
		MessageId: "test-msg-id",
		Body:      body,
	}
}

// ---------------------------------------------------------------------------
// Tests: parseMessage
// ---------------------------------------------------------------------------

// TestParseMessage_DirectJobMessage — direct JSON format sent by upload-api.
func TestParseMessage_DirectJobMessage(t *testing.T) {
	body := `{"jobId":"job-1","objectKey":"raw/job-1/photo.jpg","fileType":"image"}`
	msg, err := parseMessage(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.JobID != "job-1" {
		t.Errorf("jobId: got %q, want %q", msg.JobID, "job-1")
	}
	if msg.ObjectKey != "raw/job-1/photo.jpg" {
		t.Errorf("objectKey: got %q, want %q", msg.ObjectKey, "raw/job-1/photo.jpg")
	}
	if msg.FileType != "image" {
		t.Errorf("fileType: got %q, want %q", msg.FileType, "image")
	}
}

// TestParseMessage_S3EventFormat — S3 event notification format from bucket notification.
func TestParseMessage_S3EventFormat(t *testing.T) {
	// Minimal S3 event notification body (URLDecodedKey populated).
	body := `{
		"Records": [{
			"s3": {
				"object": {
					"key": "raw/job-abc/image.png",
					"urlDecodedKey": "raw/job-abc/image.png"
				}
			}
		}]
	}`
	msg, err := parseMessage(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.JobID != "job-abc" {
		t.Errorf("jobId: got %q, want %q", msg.JobID, "job-abc")
	}
	if msg.ObjectKey != "raw/job-abc/image.png" {
		t.Errorf("objectKey: got %q, want %q", msg.ObjectKey, "raw/job-abc/image.png")
	}
}

// TestParseMessage_InvalidBody — completely unrecognisable body must return error
// so the message retries and eventually lands in the DLQ.
func TestParseMessage_InvalidBody(t *testing.T) {
	_, err := parseMessage("not-json-at-all!!!")
	if err == nil {
		t.Fatal("expected an error for invalid body, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: processRecord
// ---------------------------------------------------------------------------

// TestProcessRecord_IdempotencySkipComplete — duplicate delivery on a COMPLETE job
// must return nil without writing anything to storage.
func TestProcessRecord_IdempotencySkipComplete(t *testing.T) {
	existing := &models.Job{JobID: "job-done", Status: models.StatusComplete}
	fdb := newFakeDB(existing)
	fst := newFakeStorage(nil)

	body := `{"jobId":"job-done","objectKey":"raw/job-done/file.jpg","fileType":"image"}`
	err := processRecord(context.Background(), sqsRecord(body), fdb, fst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fdb.statusHistory) != 0 {
		t.Errorf("expected no DB writes, got: %v", fdb.statusHistory)
	}
	if len(fst.written) != 0 {
		t.Errorf("expected no S3 writes, got: %v", fst.written)
	}
}

// TestProcessRecord_IdempotencySkipFailed — duplicate delivery on a FAILED job must also skip.
func TestProcessRecord_IdempotencySkipFailed(t *testing.T) {
	existing := &models.Job{JobID: "job-fail", Status: models.StatusFailed}
	fdb := newFakeDB(existing)
	fst := newFakeStorage(nil)

	body := `{"jobId":"job-fail","objectKey":"raw/job-fail/file.jpg","fileType":"image"}`
	err := processRecord(context.Background(), sqsRecord(body), fdb, fst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fdb.statusHistory) != 0 {
		t.Errorf("expected no DB writes, got: %v", fdb.statusHistory)
	}
}

// TestProcessRecord_FullFlow_PENDING — happy path for a PENDING job:
// status must transition PROCESSING → COMPLETE, output object must be written.
func TestProcessRecord_FullFlow_PENDING(t *testing.T) {
	jobID := "job-123"
	objectKey := "raw/job-123/photo.jpg"
	rawBytes := []byte("fake image data")

	fdb := newFakeDB(&models.Job{JobID: jobID, Status: models.StatusPending})
	fst := newFakeStorage(map[string][]byte{objectKey: rawBytes})

	body := fmt.Sprintf(`{"jobId":"%s","objectKey":"%s","fileType":"image"}`, jobID, objectKey)
	err := processRecord(context.Background(), sqsRecord(body), fdb, fst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Status transitions must be PROCESSING then COMPLETE.
	wantHistory := []string{models.StatusProcessing, models.StatusComplete}
	if len(fdb.statusHistory) != len(wantHistory) {
		t.Fatalf("status history: got %v, want %v", fdb.statusHistory, wantHistory)
	}
	for i, want := range wantHistory {
		if fdb.statusHistory[i] != want {
			t.Errorf("status[%d]: got %q, want %q", i, fdb.statusHistory[i], want)
		}
	}

	// Output object must exist at output/<jobId>/photo.jpg.
	expectedOutputKey := fmt.Sprintf("output/%s/photo.jpg", jobID)
	written, ok := fst.written[expectedOutputKey]
	if !ok {
		t.Fatalf("expected output object at %q, not found; written keys: %v", expectedOutputKey, keys(fst.written))
	}
	if string(written) != string(rawBytes) {
		t.Errorf("output bytes mismatch: got %q, want %q", written, rawBytes)
	}

	// DynamoDB record must carry the outputKey.
	job := fdb.jobs[jobID]
	if job.OutputKey != expectedOutputKey {
		t.Errorf("outputKey in DB: got %q, want %q", job.OutputKey, expectedOutputKey)
	}
}

// TestProcessRecord_FailedGetObject — when S3 GetObject fails the worker must
// mark the job FAILED and propagate the error (so SQS retries → DLQ).
func TestProcessRecord_FailedGetObject(t *testing.T) {
	jobID := "job-get-err"
	fdb := newFakeDB(&models.Job{JobID: jobID, Status: models.StatusPending})
	fst := newFakeStorage(nil)
	fst.forceGetErr = errors.New("s3: connection refused")

	body := fmt.Sprintf(`{"jobId":"%s","objectKey":"raw/%s/file.jpg","fileType":"image"}`, jobID, jobID)
	err := processRecord(context.Background(), sqsRecord(body), fdb, fst)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Status must be PROCESSING (set before the download attempt) then FAILED.
	wantHistory := []string{models.StatusProcessing, models.StatusFailed}
	if len(fdb.statusHistory) != len(wantHistory) {
		t.Fatalf("status history: got %v, want %v", fdb.statusHistory, wantHistory)
	}
	for i, want := range wantHistory {
		if fdb.statusHistory[i] != want {
			t.Errorf("status[%d]: got %q, want %q", i, fdb.statusHistory[i], want)
		}
	}
}

// TestProcessRecord_FailedPutOutput — when S3 PutObject fails the worker must
// mark the job FAILED and propagate the error.
func TestProcessRecord_FailedPutOutput(t *testing.T) {
	jobID := "job-put-err"
	objectKey := fmt.Sprintf("raw/%s/file.jpg", jobID)
	fdb := newFakeDB(&models.Job{JobID: jobID, Status: models.StatusPending})
	fst := newFakeStorage(map[string][]byte{objectKey: []byte("bytes")})
	fst.forcePutErr = errors.New("s3: permission denied")

	body := fmt.Sprintf(`{"jobId":"%s","objectKey":"%s","fileType":"image"}`, jobID, objectKey)
	err := processRecord(context.Background(), sqsRecord(body), fdb, fst)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Status must be PROCESSING then FAILED.
	wantHistory := []string{models.StatusProcessing, models.StatusFailed}
	if len(fdb.statusHistory) != len(wantHistory) {
		t.Fatalf("status history: got %v, want %v", fdb.statusHistory, wantHistory)
	}
	for i, want := range wantHistory {
		if fdb.statusHistory[i] != want {
			t.Errorf("status[%d]: got %q, want %q", i, fdb.statusHistory[i], want)
		}
	}
}

// TestProcessRecord_OptimizesToWebP — a real image is converted to WebP with
// the .webp key and image/webp content type.
func TestProcessRecord_OptimizesToWebP(t *testing.T) {
	jobID := "job-webp"
	objectKey := "raw/job-webp/photo.jpg"

	img := image.NewNRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(y), 128, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}

	fdb := newFakeDB(&models.Job{JobID: jobID, Status: models.StatusPending})
	fst := newFakeStorage(map[string][]byte{objectKey: buf.Bytes()})

	body := fmt.Sprintf(`{"jobId":"%s","objectKey":"%s","fileType":"image"}`, jobID, objectKey)
	if err := processRecord(context.Background(), sqsRecord(body), fdb, fst); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantKey := "output/job-webp/photo.webp"
	if _, ok := fst.written[wantKey]; !ok {
		t.Fatalf("expected output at %q; written keys: %v", wantKey, keys(fst.written))
	}
	if ct := fst.contentTypes[wantKey]; ct != "image/webp" {
		t.Errorf("content type = %q, want image/webp", ct)
	}
	if fdb.jobs[jobID].OutputKey != wantKey {
		t.Errorf("outputKey in DB = %q, want %q", fdb.jobs[jobID].OutputKey, wantKey)
	}
}

// TestProcessRecord_FallbackPassthrough — undecodable input is copied as-is
// and the job still completes.
func TestProcessRecord_FallbackPassthrough(t *testing.T) {
	jobID := "job-fallback"
	objectKey := "raw/job-fallback/file.bin"
	raw := []byte("definitely not an image")

	fdb := newFakeDB(&models.Job{JobID: jobID, Status: models.StatusPending})
	fst := newFakeStorage(map[string][]byte{objectKey: raw})

	body := fmt.Sprintf(`{"jobId":"%s","objectKey":"%s","fileType":"image"}`, jobID, objectKey)
	if err := processRecord(context.Background(), sqsRecord(body), fdb, fst); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantKey := "output/job-fallback/file.bin"
	if string(fst.written[wantKey]) != string(raw) {
		t.Errorf("expected passthrough bytes at %q", wantKey)
	}
	if fdb.jobs[jobID].Status != models.StatusComplete {
		t.Errorf("status = %q, want COMPLETE", fdb.jobs[jobID].Status)
	}
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
