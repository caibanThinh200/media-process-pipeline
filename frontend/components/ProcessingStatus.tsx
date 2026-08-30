"use client";

import { useEffect, useRef, useState } from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";

type JobStatus = "PENDING" | "UPLOADED" | "PROCESSING" | "COMPLETE" | "FAILED";

interface JobRecord {
  jobId: string;
  status: JobStatus;
  outputKey?: string;
  fileType?: "image" | "video";
}

interface ProcessingStatusProps {
  jobId: string;
  pollIntervalMs?: number;
  onComplete?: (jobId: string, outputKey: string, fileType: "image" | "video") => void;
  onFailed?: () => void;
}

const STATUS_STEPS: JobStatus[] = ["PENDING", "UPLOADED", "PROCESSING", "COMPLETE"];

const STATUS_LABEL: Record<JobStatus, string> = {
  PENDING: "Queued",
  UPLOADED: "Received",
  PROCESSING: "Processing",
  COMPLETE: "Complete",
  FAILED: "Failed",
};

const STATUS_COLOR: Record<JobStatus, string> = {
  PENDING: "bg-muted text-muted-foreground",
  UPLOADED: "bg-blue-500/20 text-blue-400 border-blue-500/30",
  PROCESSING: "bg-amber-500/20 text-amber-400 border-amber-500/30",
  COMPLETE: "bg-green-500/20 text-green-400 border-green-500/30",
  FAILED: "bg-destructive/20 text-destructive border-destructive/30",
};

export function ProcessingStatus({
  jobId,
  pollIntervalMs = 2500,
  onComplete,
  onFailed,
}: ProcessingStatusProps) {
  const [status, setStatus] = useState<JobStatus>("PENDING");
  const [pollError, setPollError] = useState<string | null>(null);
  const onCompleteRef = useRef(onComplete);
  const onFailedRef = useRef(onFailed);

  // Keep refs stable so the effect doesn't re-run when callbacks change
  useEffect(() => { onCompleteRef.current = onComplete; }, [onComplete]);
  useEffect(() => { onFailedRef.current = onFailed; }, [onFailed]);

  useEffect(() => {
    if (status === "COMPLETE" || status === "FAILED") return;

    const interval = setInterval(async () => {
      try {
        const res = await fetch(`/api/job-status/${jobId}`);
        if (!res.ok) throw new Error(`Status ${res.status}`);
        const data: JobRecord = await res.json();

        setStatus(data.status);
        setPollError(null);

        if (data.status === "COMPLETE") {
          onCompleteRef.current?.(
            jobId,
            data.outputKey ?? "",
            data.fileType ?? "image"
          );
        } else if (data.status === "FAILED") {
          onFailedRef.current?.();
        }
      } catch (err) {
        setPollError(err instanceof Error ? err.message : "Polling error");
      }
    }, pollIntervalMs);

    return () => clearInterval(interval);
  }, [jobId, status, pollIntervalMs]);

  const activeStep = STATUS_STEPS.indexOf(status === "FAILED" ? "COMPLETE" : status);

  return (
    <div
      role="status"
      aria-live="polite"
      aria-label={`Job status: ${STATUS_LABEL[status]}`}
      className="flex w-full flex-col items-center gap-5"
    >
      {/* ── Step indicator ── */}
      <div className="flex w-full max-w-xs items-center justify-between" aria-hidden="true">
        {STATUS_STEPS.map((step, i) => {
          const isCurrent = i === activeStep && status !== "FAILED" && status !== "COMPLETE";
          const isDone = i < activeStep || status === "COMPLETE";
          const isFailed = status === "FAILED" && i === STATUS_STEPS.length - 1;

          return (
            <div key={step} className="flex flex-1 flex-col items-center gap-1.5">
              {/* Connector line (between circles) */}
              {i > 0 && (
                <div
                  className={cn(
                    "absolute mt-3 h-0.5 w-full -translate-x-1/2 transition-colors duration-500",
                    isDone ? "bg-primary/60" : "bg-border/50"
                  )}
                  style={{ left: 0, position: "relative", flex: 1 }}
                />
              )}
              {/* Circle */}
              <div
                className={cn(
                  "flex h-7 w-7 items-center justify-center rounded-full border-2 text-xs font-semibold transition-all duration-500",
                  isDone && !isFailed && "border-primary bg-primary text-primary-foreground",
                  isCurrent && "border-primary bg-primary/20 text-primary",
                  !isDone && !isCurrent && "border-border bg-muted text-muted-foreground",
                  isFailed && "border-destructive bg-destructive/20 text-destructive"
                )}
              >
                {isDone && !isFailed ? (
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"
                    strokeLinecap="round" strokeLinejoin="round" className="h-3.5 w-3.5">
                    <polyline points="20 6 9 17 4 12" />
                  </svg>
                ) : isFailed ? (
                  "✕"
                ) : (
                  i + 1
                )}
              </div>
              {/* Label */}
              <span
                className={cn(
                  "text-[10px] transition-colors",
                  isCurrent ? "font-medium text-primary" : "text-muted-foreground"
                )}
              >
                {STATUS_LABEL[step]}
              </span>
            </div>
          );
        })}
      </div>

      {/* ── Status badge + spinner ── */}
      <div className="flex items-center gap-2">
        {status !== "COMPLETE" && status !== "FAILED" && (
          <svg
            className="h-4 w-4 animate-spin text-primary"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8z" />
          </svg>
        )}
        <Badge
          variant="outline"
          className={cn("border text-xs font-medium", STATUS_COLOR[status])}
        >
          {STATUS_LABEL[status]}
        </Badge>
      </div>

      {/* ── Job ID (debug) ── */}
      <p className="text-[10px] text-muted-foreground/50">
        Job: <span className="font-mono">{jobId.slice(0, 8)}…</span>
      </p>

      {/* ── Poll error ── */}
      {pollError && (
        <p className="text-xs text-destructive" role="alert">
          {pollError}
        </p>
      )}
    </div>
  );
}
