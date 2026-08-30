"use client";

import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardFooter,
} from "@/components/ui/card";

type JobStatus = "PENDING" | "UPLOADED" | "PROCESSING" | "COMPLETE" | "FAILED";

export interface MediaCardProps {
  jobId: string;
  status: JobStatus;
  fileType: "image" | "video";
  outputUrl?: string;
  createdAt: string;
}

const STATUS_BADGE: Record<JobStatus, string> = {
  PENDING: "bg-muted text-muted-foreground border-border",
  UPLOADED: "bg-blue-500/15 text-blue-400 border-blue-500/30",
  PROCESSING: "bg-amber-500/15 text-amber-400 border-amber-500/30",
  COMPLETE: "bg-green-500/15 text-green-400 border-green-500/30",
  FAILED: "bg-destructive/15 text-destructive border-destructive/30",
};

const STATUS_LABEL: Record<JobStatus, string> = {
  PENDING: "Queued",
  UPLOADED: "Received",
  PROCESSING: "Processing",
  COMPLETE: "Complete",
  FAILED: "Failed",
};

export function MediaCard({ jobId, status, fileType, outputUrl, createdAt }: MediaCardProps) {
  const hasOutput = status === "COMPLETE" && outputUrl;

  return (
    <Card
      className={cn(
        "group overflow-hidden border-border/60 bg-card/60 backdrop-blur-sm transition-all duration-300",
        "hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-lg hover:shadow-primary/10",
        "animate-fade-up"
      )}
      aria-label={`Media job ${jobId.slice(0, 8)}`}
    >
      {/* ── Media preview ── */}
      <div className="relative aspect-video overflow-hidden bg-muted/40">
        {hasOutput ? (
          <>
            {fileType === "video" ? (
              <video
                src={outputUrl}
                controls
                preload="metadata"
                className="h-full w-full object-cover"
              />
            ) : (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={outputUrl}
                alt={`Processed output for job ${jobId.slice(0, 8)}`}
                loading="lazy"
                className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-105"
              />
            )}
            {/* Hover overlay with link */}
            {fileType === "image" && (
              <a
                href={outputUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="absolute inset-0 flex items-center justify-center bg-black/0 opacity-0 transition-all duration-300 group-hover:bg-black/40 group-hover:opacity-100"
                aria-label="Open processed image in new tab"
              >
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
                  stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
                  className="h-8 w-8 text-white drop-shadow-lg">
                  <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
                  <polyline points="15 3 21 3 21 9" />
                  <line x1="10" y1="14" x2="21" y2="3" />
                </svg>
              </a>
            )}
          </>
        ) : (
          /* Placeholder with animated gradient for processing */
          <div
            className={cn(
              "flex h-full w-full items-center justify-center text-muted-foreground/40",
              status === "PROCESSING" || status === "UPLOADED"
                ? "animate-pulse bg-gradient-to-br from-muted/60 to-muted/20"
                : "bg-muted/30"
            )}
            aria-label={status === "FAILED" ? "Processing failed" : "Processing…"}
          >
            {status === "FAILED" ? (
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"
                className="h-10 w-10 text-destructive/40">
                <circle cx="12" cy="12" r="10" />
                <line x1="12" y1="8" x2="12" y2="12" />
                <line x1="12" y1="16" x2="12.01" y2="16" />
              </svg>
            ) : (
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"
                className="h-10 w-10">
                <rect width="18" height="18" x="3" y="3" rx="2" ry="2" />
                <circle cx="9" cy="9" r="2" />
                <path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21" />
              </svg>
            )}
          </div>
        )}
      </div>

      {/* ── Card info ── */}
      <CardContent className="px-3 pt-3 pb-1">
        <div className="flex items-center justify-between gap-2">
          <Badge
            variant="outline"
            className={cn("border text-[10px] font-medium", STATUS_BADGE[status])}
          >
            {STATUS_LABEL[status]}
          </Badge>
          <span className="rounded bg-muted/50 px-1.5 py-0.5 text-[10px] uppercase tracking-wider text-muted-foreground">
            {fileType}
          </span>
        </div>
      </CardContent>

      <CardFooter className="flex flex-col items-start gap-0.5 px-3 pb-3 pt-0">
        <p className="font-mono text-[10px] text-muted-foreground/60">
          {jobId.slice(0, 8)}…
        </p>
        <time
          className="text-[10px] text-muted-foreground/50"
          dateTime={createdAt}
          title={new Date(createdAt).toLocaleString()}
        >
          {new Date(createdAt).toLocaleDateString(undefined, {
            month: "short",
            day: "numeric",
            hour: "2-digit",
            minute: "2-digit",
          })}
        </time>
      </CardFooter>
    </Card>
  );
}
