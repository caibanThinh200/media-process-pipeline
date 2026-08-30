ACCOUNT_ID   := 954212211710
ADMIN_PROFILE := root-admin
POLICY_DIR   := terraform/iam-policies

.PHONY: policy-update-core policy-update-platform policy-list-versions \
        build-upload-api build-image-worker test-unit deploy-phase3 \
        frontend-dev frontend-build

## policy-update-core: push a new default version of the Core policy (run from project root)
policy-update-core:
	@SIZE=$$(python3 -c "import json; f=open('$(POLICY_DIR)/terraform-policy-core.json'); print(len(json.dumps(json.load(f),separators=(',',':'))))"); \
	echo "Policy size: $$SIZE chars"; \
	if [ $$SIZE -gt 6144 ]; then echo "ERROR: exceeds 6144 char limit"; exit 1; fi
	AWS_PROFILE=$(ADMIN_PROFILE) aws iam create-policy-version \
	  --policy-arn arn:aws:iam::$(ACCOUNT_ID):policy/TerraformMediaPipelineCorePolicy \
	  --policy-document file://$(POLICY_DIR)/terraform-policy-core.json \
	  --set-as-default \
	  --output text
	@echo "Core policy updated."

## policy-update-platform: push a new default version of the Platform policy
policy-update-platform:
	@SIZE=$$(python3 -c "import json; f=open('$(POLICY_DIR)/terraform-policy-platform.json'); print(len(json.dumps(json.load(f),separators=(',',':'))))"); \
	echo "Policy size: $$SIZE chars"; \
	if [ $$SIZE -gt 6144 ]; then echo "ERROR: exceeds 6144 char limit"; exit 1; fi
	AWS_PROFILE=$(ADMIN_PROFILE) aws iam create-policy-version \
	  --policy-arn arn:aws:iam::$(ACCOUNT_ID):policy/TerraformMediaPipelinePlatformPolicy \
	  --policy-document file://$(POLICY_DIR)/terraform-policy-platform.json \
	  --set-as-default \
	  --output text
	@echo "Platform policy updated."

## policy-list-versions: list all versions of both policies
policy-list-versions:
	@echo "=== Core Policy ==="
	AWS_PROFILE=$(ADMIN_PROFILE) aws iam list-policy-versions \
	  --policy-arn arn:aws:iam::$(ACCOUNT_ID):policy/TerraformMediaPipelineCorePolicy \
	  --output table
	@echo "=== Platform Policy ==="
	AWS_PROFILE=$(ADMIN_PROFILE) aws iam list-policy-versions \
	  --policy-arn arn:aws:iam::$(ACCOUNT_ID):policy/TerraformMediaPipelinePlatformPolicy \
	  --output table

## build-upload-api: compile the upload-api Lambda binary for Linux arm64
build-upload-api:
	$(MAKE) -C backend build-upload-api

## build-image-worker: compile the image-worker Lambda binary for Linux arm64
build-image-worker:
	$(MAKE) -C backend build-image-worker

## test-unit: run all Go unit tests (no AWS credentials required)
test-unit:
	cd backend && /usr/local/go/bin/go test ./... -v -count=1

## deploy-phase3: build image-worker binary then apply Terraform (Phase 3)
deploy-phase3: build-image-worker
	AWS_PROFILE=media-pipeline-dev terraform -chdir=terraform apply -auto-approve

## tf-plan: run terraform plan with the infra profile
tf-plan:
	AWS_PROFILE=media-pipeline-dev terraform -chdir=terraform plan

## tf-apply: run terraform apply with the infra profile
tf-apply:
	AWS_PROFILE=media-pipeline-dev terraform -chdir=terraform apply

## help: show available targets
help:
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/## //'

## frontend-dev: start the Next.js dev server (requires Node >=20 via nvm)
frontend-dev:
	. ~/.nvm/nvm.sh && nvm use 22 && cd frontend && pnpm dev

## frontend-build: build the Next.js production bundle
frontend-build:
	. ~/.nvm/nvm.sh && nvm use 22 && cd frontend && pnpm build
