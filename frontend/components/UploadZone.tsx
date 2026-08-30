"use client";

import { useRef, useState, useCallback } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { ProcessingStatus } from "@/components/ProcessingStatus";

type UploadPhase = "idle" | "dragging" | "uploading" | "processing" | "complete" | "error";

export interface CompletedJob {
  jobId: string;
  outputUrl: string;
  fileType: "image" | "video";
}

interface UploadZoneProps {
  onComplete?: (job: CompletedJob) => void;
}

const ACCEPTED = "image/*,video/*";
const CLOUDFRONT = process.env.NEXT_PUBLIC_CLOUDFRONT_URL ?? "";

export function UploadZone({ onComplete }: UploadZoneProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [phase, setPhase] = useState<UploadPhase>("idle");
  const [uploadProgress, setUploadProgress] = useState(0);
  const [activeJobId, setActiveJobId] = useState<string | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const reset = useCallback(() => {
    setPhase("idle");
    setUploadProgress(0);
    setActiveJobId(null);
    setErrorMessage(null);
    if (inputRef.current) inputRef.current.value = "";
  }, []);

  const handleFile = useCallback(async (file: File) => {
    setPhase("uploading");
    setUploadProgress(0);
    setErrorMessage(null);

    try {
      const fileType = file.type.startsWith("video/") ? "video" : "image";

      // ── 1. Request presigned URL from BFF ──────────────────────────────────
      const presignRes = await fetch("/api/upload/presign", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          fileName: file.name,
          fileType,
          mediaType: file.type,
          sizeBytes: file.size,
        }),
      });

      if (!presignRes.ok) {
        const err = await presignRes.json().catch(() => ({}));
        throw new Error(err?.error ?? "Failed to get upload URL");
      }

      const { jobId, uploadUrl } = (await presignRes.json()) as {
        jobId: string;
        uploadUrl: string;
        objectKey: string;
      };

      // ── 2. Upload directly to S3 with XHR (gives us progress events) ───────
      await new Promise<void>((resolve, reject) => {
        const xhr = new XMLHttpRequest();

        xhr.upload.addEventListener("progress", (e) => {
          if (e.lengthComputable) {
            setUploadProgress(Math.round((e.loaded / e.total) * 100));
          }
        });

        xhr.addEventListener("load", () => {
          if (xhr.status >= 200 && xhr.status < 300) {
            resolve();
          } else {
            reject(new Error(`S3 upload failed (HTTP ${xhr.status})`));
          }
        });

        xhr.addEventListener("error", () => reject(new Error("Network error during upload")));
        xhr.addEventListener("abort", () => reject(new Error("Upload cancelled")));

        xhr.open("PUT", uploadUrl);
        xhr.setRequestHeader("Content-Type", file.type);
        xhr.send(file);
      });

      // ── 3. Hand off to polling phase ────────────────────────────────────────
      setUploadProgress(100);
      setActiveJobId(jobId);
      setPhase("processing");

      // Persist job id to localStorage for gallery
      try {
        const stored = JSON.parse(localStorage.getItem("mpp_job_ids") ?? "[]") as string[];
        localStorage.setItem("mpp_job_ids", JSON.stringify([jobId, ...stored].slice(0, 50)));
      } catch {
        // localStorage unavailable — non-fatal
      }
    } catch (err) {
      setErrorMessage(err instanceof Error ? err.message : "Upload failed");
      setPhase("error");
    }
  }, []);

  const onJobComplete = useCallback(
    (jobId: string, outputKey: string, fileType: "image" | "video") => {
      setPhase("complete");
      const outputUrl = outputKey
        ? `https://${CLOUDFRONT}/${outputKey}`
        : "";
      onComplete?.({ jobId, outputUrl, fileType });
    },
    [onComplete]
  );

  const onJobFailed = useCallback(() => {
    setPhase("error");
    setErrorMessage("Processing failed. You can try uploading again.");
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      setPhase("idle");
      const file = e.dataTransfer.files[0];
      if (file) handleFile(file);
    },
    [handleFile]
  );

  return (
    <div className="space-y-4">
      {/* ── Drop zone ── */}
      <div
        id="upload-zone"
        role="button"
        tabIndex={0}
        aria-label="Upload media file — drag and drop or click to browse"
        onClick={() => phase === "idle" && inputRef.current?.click()}
        onKeyDown={(e) => {
          if ((e.key === "Enter" || e.key === " ") && phase === "idle") {
            inputRef.current?.click();
          }
        }}
        onDragOver={(e) => { e.preventDefault(); if (phase === "idle") setPhase("dragging"); }}
        onDragLeave={() => setPhase((p) => (p === "dragging" ? "idle" : p))}
        onDrop={handleDrop}
        className={cn(
          "relative flex min-h-56 cursor-pointer flex-col items-center justify-center gap-3 rounded-2xl border-2 border-dashed px-6 py-10 text-center transition-all duration-300",
          // Base state
          phase === "idle" &&
            "border-border/50 bg-muted/30 hover:border-primary/50 hover:bg-muted/50",
          // Dragging
          phase === "dragging" &&
            "border-primary/70 bg-primary/5 animate-drag-glow cursor-copy",
          // Uploading / Processing
          (phase === "uploading" || phase === "processing") &&
            "cursor-default border-primary/40 bg-muted/20",
          // Complete
          phase === "complete" &&
            "cursor-default border-green-500/50 bg-green-500/5",
          // Error
          phase === "error" &&
            "cursor-default border-destructive/50 bg-destructive/5"
        )}
      >
        <input
          ref={inputRef}
          type="file"
          accept={ACCEPTED}
          className="sr-only"
          id="file-input"
          onChange={(e) => {
            const file = e.target.files?.[0];
            if (file) handleFile(file);
          }}
        />

        {/* Idle & dragging */}
        {(phase === "idle" || phase === "dragging") && (
          <div className="flex flex-col items-center gap-3 animate-fade-up">
            <div className={cn(
              "flex h-14 w-14 items-center justify-center rounded-full transition-colors",
              phase === "dragging" ? "bg-primary/20 text-primary" : "bg-muted text-muted-foreground"
            )}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round"
                className="h-7 w-7" aria-hidden="true">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                <polyline points="17 8 12 3 7 8" />
                <line x1="12" y1="3" x2="12" y2="15" />
              </svg>
            </div>
            <div>
              <p className="font-medium text-foreground">
                {phase === "dragging" ? "Drop it here!" : "Drag & drop your media"}
              </p>
              <p className="mt-1 text-sm text-muted-foreground">
                or{" "}
                <span className="text-primary underline-offset-2 hover:underline">
                  click to browse
                </span>
              </p>
              <p className="mt-2 text-xs text-muted-foreground/70">
                Images and videos supported
              </p>
            </div>
          </div>
        )}

        {/* Uploading */}
        {phase === "uploading" && (
          <div className="flex w-full max-w-xs flex-col items-center gap-4 animate-fade-up">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/20 text-primary">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"
                className="h-6 w-6" aria-hidden="true">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                <polyline points="17 8 12 3 7 8" />
                <line x1="12" y1="3" x2="12" y2="15" />
              </svg>
            </div>
            <div className="w-full space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Uploading to S3…</span>
                <span className="font-medium text-foreground">{uploadProgress}%</span>
              </div>
              <Progress value={uploadProgress} className="h-1.5" />
            </div>
          </div>
        )}

        {/* Processing */}
        {phase === "processing" && activeJobId && (
          <div className="flex w-full flex-col items-center gap-4 animate-fade-up">
            <ProcessingStatus
              jobId={activeJobId}
              onComplete={onJobComplete}
              onFailed={onJobFailed}
            />
          </div>
        )}

        {/* Complete */}
        {phase === "complete" && (
          <div className="flex flex-col items-center gap-3 animate-fade-up">
            <div className="flex h-14 w-14 items-center justify-center rounded-full bg-green-500/20 text-green-400">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
                className="h-7 w-7" aria-hidden="true">
                <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                <polyline points="22 4 12 14.01 9 11.01" />
              </svg>
            </div>
            <p className="font-medium text-green-400">Processing complete!</p>
            <Button variant="outline" size="sm" onClick={reset}>
              Upload another
            </Button>
          </div>
        )}

        {/* Error */}
        {phase === "error" && (
          <div className="flex flex-col items-center gap-3 animate-fade-up">
            <div className="flex h-14 w-14 items-center justify-center rounded-full bg-destructive/20 text-destructive">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"
                className="h-7 w-7" aria-hidden="true">
                <circle cx="12" cy="12" r="10" />
                <line x1="12" y1="8" x2="12" y2="12" />
                <line x1="12" y1="16" x2="12.01" y2="16" />
              </svg>
            </div>
            <p className="max-w-xs text-sm text-destructive">{errorMessage}</p>
            <Button variant="outline" size="sm" onClick={reset}>
              Try again
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
