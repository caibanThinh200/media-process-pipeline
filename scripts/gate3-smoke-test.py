#!/usr/bin/env python3
import sys
import os
import json
import time
import subprocess
import urllib.request
import urllib.error

# ANSI coloring
def info(msg):
    print(f"\033[0;32m[INFO] {msg}\033[0m")

def warn(msg):
    print(f"\033[0;33m[WARN] {msg}\033[0m")

def error(msg):
    print(f"\033[0;31m[ERROR] {msg}\033[0m")
    sys.exit(1)

# Helper to run shell commands and return stdout
def run_cmd(cmd, env=None):
    try:
        res = subprocess.run(cmd, shell=True, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=env)
        return res.stdout.decode('utf-8').strip()
    except subprocess.CalledProcessError as e:
        error(f"Command failed: {cmd}\nStdout: {e.stdout.decode('utf-8')}\nStderr: {e.stderr.decode('utf-8')}")

# Resolve script paths
DIR = os.path.dirname(os.path.abspath(__file__))
TF_DIR = os.path.join(DIR, "..", "terraform")

# Load environment variables
env = os.environ.copy()

info("Fetching Terraform outputs...")
# Verify terraform works
api_endpoint = run_cmd("terraform -chdir=" + TF_DIR + " output -raw api_endpoint", env)
raw_bucket = run_cmd("terraform -chdir=" + TF_DIR + " output -raw raw_bucket_name", env)
output_bucket = run_cmd("terraform -chdir=" + TF_DIR + " output -raw output_bucket_name", env)
jobs_table = run_cmd("terraform -chdir=" + TF_DIR + " output -raw jobs_table_name", env)
queue_url = run_cmd("terraform -chdir=" + TF_DIR + " output -raw sqs_queue_url", env)
dlq_name = run_cmd("terraform -chdir=" + TF_DIR + " output -raw sqs_dlq_name", env)
cloudfront_domain_name = run_cmd("terraform -chdir=" + TF_DIR + " output -raw cloudfront_domain_name", env)

dlq_url = run_cmd(f"aws sqs get-queue-url --queue-name {dlq_name} --query 'QueueUrl' --output text", env)


info("Configuration:")
print(f"  API Endpoint:  {api_endpoint}")
print(f"  Raw Bucket:    {raw_bucket}")
print(f"  Output Bucket: {output_bucket}")
print(f"  Jobs Table:    {jobs_table}")
print(f"  Queue URL:     {queue_url}")
print(f"  DLQ URL:       {dlq_url}")

# ==============================================================================
# CRITERION 1 & 2 & 3: Upload → SQS → Lambda processing & output object creation
# ==============================================================================
info("Testing Criterion 1, 2 & 3: Upload flow and SQS-triggered worker processing...")

filename = "smoke-test-image.jpg"
payload = json.dumps({
    "fileName": filename,
    "fileType": "image",
    "mediaType": "image/jpeg",
    "sizeBytes": 1024
}).encode('utf-8')


# Request presigned upload URL
info("Requesting presigned upload URL...")
req = urllib.request.Request(
    f"{api_endpoint.rstrip('/')}/upload/presign",
    data=payload,
    headers={'Content-Type': 'application/json'}
)

try:
    with urllib.request.urlopen(req) as response:
        presign_response = json.loads(response.read().decode('utf-8'))
except urllib.error.URLError as e:
    error(f"Failed to post to upload api: {e}")

job_id = presign_response.get('jobId')
upload_url = presign_response.get('uploadUrl')
object_key = presign_response.get('objectKey')

if not job_id:
    error(f"Failed to get jobId from presign response: {presign_response}")

info(f"Presign success. Job ID: {job_id}")
info("Uploading fake image content to raw bucket...")
fake_content = f"fake-image-content-{int(time.time())}".encode('utf-8')

# Put to S3 presigned URL
req_put = urllib.request.Request(
    upload_url,
    data=fake_content,
    method='PUT',
    headers={'Content-Type': 'image/jpeg'}
)
try:
    with urllib.request.urlopen(req_put) as resp:
        pass
except urllib.error.URLError as e:
    error(f"Failed to upload to S3 presigned URL: {e}")

info(f"File uploaded. Polling API status for {job_id}...")
max_attempts = 20
status = ""

for attempt in range(1, max_attempts + 1):
    req_status = urllib.request.Request(
        f"{api_endpoint.rstrip('/')}/jobs/{job_id}",
        method='GET'
    )
    try:
        with urllib.request.urlopen(req_status) as response:
            job_record = json.loads(response.read().decode('utf-8'))
            status = job_record.get('status', '')
            info(f"Attempt {attempt}/{max_attempts}: Status is '{status}'")
            if status == "COMPLETE":
                break
            if status == "FAILED":
                error(f"Job {job_id} failed processing!")
    except urllib.error.HTTPError as e:
        if e.code == 404:
            warn(f"Attempt {attempt}/{max_attempts}: Job record not found in DynamoDB yet...")
        else:
            warn(f"Attempt {attempt}/{max_attempts}: HTTP Error {e.code}")
    except urllib.error.URLError as e:
        warn(f"Attempt {attempt}/{max_attempts}: Connection error: {e}")
    
    time.sleep(3)

if status != "COMPLETE":
    error(f"Timeout waiting for job status to reach COMPLETE. Current status: {status}")

info("Criterion 1 & 2 Passed: Upload triggered SQS/Lambda. Status is COMPLETE.")

# Verify output object exists and is served via CloudFront CDN
info(f"Checking output object via CloudFront CDN: {cloudfront_domain_name}...")
cdn_url = f"https://{cloudfront_domain_name}/output/{job_id}/{filename}"
info(f"Fetching from CDN URL: {cdn_url}")

# CloudFront might take a second to serve or cache. Let's try up to 3 times.
cdn_ok = False
for cdn_attempt in range(1, 4):
    try:
        with urllib.request.urlopen(cdn_url) as resp:
            content = resp.read()
            if len(content) > 0:
                cdn_ok = True
                break
    except urllib.error.URLError as e:
        warn(f"CDN fetch attempt {cdn_attempt}/3 failed: {e}")
        time.sleep(2)

if not cdn_ok:
    error("Output object not served correctly via CloudFront CDN!")

info("Criterion 3 Passed: Output object served via CloudFront CDN successfully.")


# ==============================================================================
# CRITERION 5: Duplicate message delivery / Idempotency
# ==============================================================================
info("Testing Criterion 5: Idempotency (Duplicate message processing)...")

# Construct a valid JobMessage for SQS
sqs_message_body = json.dumps({
    "jobId": job_id,
    "objectKey": object_key,
    "fileType": "image"
})

info("Sending duplicate SQS message directly to queue...")
run_cmd(f"aws sqs send-message --queue-url '{queue_url}' --message-body '{sqs_message_body}'", env)

info("Waiting for any potential processing (5s)...")
time.sleep(5)

# Get the job record again and ensure status is still COMPLETE using API
req_status = urllib.request.Request(
    f"{api_endpoint.rstrip('/')}/jobs/{job_id}",
    method='GET'
)
try:
    with urllib.request.urlopen(req_status) as response:
        job_record = json.loads(response.read().decode('utf-8'))
        updated_status = job_record.get('status', '')
except Exception as e:
    error(f"Failed to fetch job status for idempotency check: {e}")

if updated_status != "COMPLETE":
    error(f"Duplicate message changed job status to {updated_status}!")
info("Criterion 5 Passed: Duplicate message did not affect job status.")

# ==============================================================================
# CRITERION 4: DLQ receives message after 3 failed processing attempts
# ==============================================================================
info("Testing Criterion 4: Dead-Letter Queue (DLQ) processing...")

# Temporarily lower VisibilityTimeout of the main queue to 5s to speed up retries
info("Temporarily setting media-queue VisibilityTimeout to 5 seconds...")
run_cmd(f"aws sqs set-queue-attributes --queue-url '{queue_url}' --attributes VisibilityTimeout=5", env)

try:
    # Send a completely malformed JSON message (should fail parsing and retry)
    bad_msg_body = "this-is-not-valid-json-and-will-fail-parsing"
    info("Sending malformed message to queue...")
    bad_msg_response = json.loads(run_cmd(
        f"aws sqs send-message --queue-url '{queue_url}' --message-body '{bad_msg_body}' --output json",
        env
    ))
    bad_msg_id = bad_msg_response.get('MessageId')

    # Since VisibilityTimeout is 5s, 3 attempts will take ~15 seconds.
    info(f"Message sent (ID: {bad_msg_id}). Waiting 25s for 3x Lambda retries...")
    time.sleep(25)


    info(f"Polling Dead-Letter Queue ({dlq_url}) for the malformed message...")
    dlq_check_attempts = 5
    found = False

    for dlq_check in range(1, dlq_check_attempts + 1):
        receive_response_json = run_cmd(
            f"aws sqs receive-message --queue-url '{dlq_url}' --max-number-of-messages 10 --wait-time-seconds 5 --output json",
            env
        )
        if receive_response_json and receive_response_json != "null":
            receive_response = json.loads(receive_response_json)
            messages = receive_response.get('Messages', [])
            
            target_msg = None
            for msg in messages:
                if msg.get('Body') == bad_msg_body:
                    target_msg = msg
                    break
            
            if target_msg:
                info("Found malformed message in Dead-Letter Queue!")
                found = True
                receipt_handle = target_msg.get('ReceiptHandle')
                # Clean up: delete message from DLQ
                run_cmd(f"aws sqs delete-message --queue-url '{dlq_url}' --receipt-handle '{receipt_handle}'", env)
                info("Cleaned up malformed message from DLQ.")
                break
                
        warn(f"DLQ poll attempt {dlq_check}/{dlq_check_attempts}: Message not found in DLQ yet...")
        time.sleep(2)

    if not found:
        error("Malformed message did not land in DLQ after retries!")

    info("Criterion 4 Passed: Malformed message successfully routed to DLQ.")

finally:
    # ALWAYS restore original VisibilityTimeout of 300s
    info("Restoring media-queue VisibilityTimeout to 300 seconds...")
    run_cmd(f"aws sqs set-queue-attributes --queue-url '{queue_url}' --attributes VisibilityTimeout=300", env)

info("======================================================================")
info("ALL GATE 3 SMOKE TEST CRITERIA PASSED SUCCESSFULLY!")
info("======================================================================")

