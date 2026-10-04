export PATH := /usr/local/go/bin:$(PATH)

GOOS    ?= linux
GOARCH  ?= arm64
OUTDIR  := build

.PHONY: all build-upload-api clean

all: build-upload-api

## build-upload-api: compile the upload-api Lambda binary for Linux arm64
build-upload-api:
	@echo "==> Building upload-api ($(GOOS)/$(GOARCH))..."
	@mkdir -p $(OUTDIR)/upload-api
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-tags lambda.norpc \
		-ldflags="-s -w" \
		-o $(OUTDIR)/upload-api/bootstrap \
		./cmd/upload-api
	@echo "==> Binary written to $(OUTDIR)/upload-api/bootstrap"

## build-image-worker: compile the image-worker Lambda binary (Phase 3)
build-image-worker:
	@echo "==> Building image-worker ($(GOOS)/$(GOARCH))..."
	@mkdir -p $(OUTDIR)/image-worker
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-tags lambda.norpc \
		-ldflags="-s -w" \
		-o $(OUTDIR)/image-worker/bootstrap \
		./cmd/image-worker
	@echo "==> Binary written to $(OUTDIR)/image-worker/bootstrap"

## build-video-worker: compile the video-worker Lambda binary (Phase 6)
build-video-worker:
	@echo "==> Building video-worker ($(GOOS)/$(GOARCH))..."
	@mkdir -p $(OUTDIR)/video-worker
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-tags lambda.norpc \
		-ldflags="-s -w" \
		-o $(OUTDIR)/video-worker/bootstrap \
		./cmd/video-worker
	@echo "==> Binary written to $(OUTDIR)/video-worker/bootstrap"

## tidy: run go mod tidy
tidy:
	go mod tidy

## vet: run go vet across all packages
vet:
	go vet ./...

## clean: remove build artifacts
clean:
	rm -rf $(OUTDIR)

## help: show this help
help:
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/## //'
