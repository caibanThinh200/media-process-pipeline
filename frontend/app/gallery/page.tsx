"use client";

import { useEffect, useMemo, useState } from "react";
import { cn } from "@/lib/utils";
import { MediaCard } from "@/components/MediaCard";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";

type JobStatus = "PENDING" | "UPLOADED" | "PROCESSING" | "COMPLETE" | "FAILED";
type FilterValue = "ALL" | JobStatus;

interface StoredJob {
  jobId: string;
  status: JobStatus;
  fileType: "image" | "video";
  outputKey?: string;
  createdAt: string;
}

const CLOUDFRONT = process.env.NEXT_PUBLIC_CLOUDFRONT_URL ?? "";
const FILTERS: { label: string; value: FilterValue }[] = [
  { label: "All", value: "ALL" },
  { label: "Complete", value: "COMPLETE" },
  { label: "Processing", value: "PROCESSING" },
  { label: "Failed", value: "FAILED" },
];

export default function GalleryPage() {
  const [jobs, setJobs] = useState<StoredJob[]>([]);
  const [filter, setFilter] = useState<FilterValue>("ALL");
  const [loading, setLoading] = useState(true);

  // Load job IDs from localStorage and fetch their current status
  useEffect(() => {
    async function loadJobs() {
      setLoading(true);
      try {
        const storedIds = JSON.parse(
          localStorage.getItem("mpp_job_ids") ?? "[]"
        ) as string[];

        if (storedIds.length === 0) {
          setJobs([]);
          return;
        }

        // Fetch all jobs in parallel
        const settled = await Promise.allSettled(
          storedIds.map(async (jobId) => {
            const res = await fetch(`/api/job-status/${jobId}`);
            if (!res.ok) throw new Error(`Failed to fetch ${jobId}`);
            return res.json() as Promise<StoredJob>;
          })
        );

        const loaded: StoredJob[] = settled
          .filter((r): r is PromiseFulfilledResult<StoredJob> => r.status === "fulfilled")
          .map((r) => r.value);

        // Sort newest-first using jobId timestamp if createdAt is missing
        loaded.sort(
          (a, b) =>
            new Date(b.createdAt || 0).getTime() -
            new Date(a.createdAt || 0).getTime()
        );

        setJobs(loaded);
      } catch {
        setJobs([]);
      } finally {
        setLoading(false);
      }
    }

    loadJobs();
  }, []);

  const filtered = useMemo(
    () => (filter === "ALL" ? jobs : jobs.filter((j) => j.status === filter)),
    [jobs, filter]
  );

  const clearGallery = () => {
    localStorage.removeItem("mpp_job_ids");
    setJobs([]);
  };

  return (
    <div className="space-y-8">
      {/* ── Page header ── */}
      <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground sm:text-3xl">
            Gallery
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {jobs.length > 0
              ? `${jobs.length} job${jobs.length !== 1 ? "s" : ""} in your session`
              : "Your processed media will appear here"}
          </p>
        </div>
        {jobs.length > 0 && (
          <Button
            variant="ghost"
            size="sm"
            className="text-muted-foreground hover:text-destructive"
            onClick={clearGallery}
          >
            Clear history
          </Button>
        )}
      </div>

      <Separator className="opacity-30" />

      {/* ── Filters ── */}
      {jobs.length > 0 && (
        <div
          className="flex flex-wrap gap-2"
          role="tablist"
          aria-label="Filter by status"
        >
          {FILTERS.map(({ label, value }) => {
            const count =
              value === "ALL" ? jobs.length : jobs.filter((j) => j.status === value).length;
            if (count === 0 && value !== "ALL") return null;
            return (
              <button
                key={value}
                id={`filter-${value.toLowerCase()}`}
                role="tab"
                aria-selected={filter === value}
                onClick={() => setFilter(value)}
                className={cn(
                  "rounded-full border px-3 py-1 text-xs font-medium transition-colors",
                  filter === value
                    ? "border-primary bg-primary/20 text-primary"
                    : "border-border/60 bg-muted/30 text-muted-foreground hover:border-primary/40 hover:text-foreground"
                )}
              >
                {label}
                <span className="ml-1.5 text-[10px] opacity-60">{count}</span>
              </button>
            );
          })}
        </div>
      )}

      {/* ── Loading skeleton ── */}
      {loading && (
        <div
          className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4"
          aria-label="Loading jobs"
        >
          {Array.from({ length: 4 }).map((_, i) => (
            <div
              key={i}
              className="aspect-video animate-pulse rounded-xl bg-muted/40"
            />
          ))}
        </div>
      )}

      {/* ── Empty state ── */}
      {!loading && filtered.length === 0 && (
        <div className="flex flex-col items-center gap-4 py-20 text-center">
          <div className="flex h-16 w-16 items-center justify-center rounded-full bg-muted/40 text-muted-foreground/40">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="h-8 w-8"
              aria-hidden="true"
            >
              <rect width="18" height="18" x="3" y="3" rx="2" ry="2" />
              <circle cx="9" cy="9" r="2" />
              <path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21" />
            </svg>
          </div>
          <div>
            <p className="font-medium text-muted-foreground">
              {jobs.length === 0 ? "No processed media yet" : "No jobs match this filter"}
            </p>
            <p className="mt-1 text-sm text-muted-foreground/60">
              {jobs.length === 0 ? (
                <>
                  Head to the{" "}
                  <a href="/" className="text-primary underline-offset-2 hover:underline">
                    upload page
                  </a>{" "}
                  to get started.
                </>
              ) : (
                "Try a different filter."
              )}
            </p>
          </div>
        </div>
      )}

      {/* ── Media grid ── */}
      {!loading && filtered.length > 0 && (
        <div
          className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4"
          aria-label="Processed media gallery"
        >
          {filtered.map((job) => (
            <MediaCard
              key={job.jobId}
              jobId={job.jobId}
              status={job.status}
              fileType={job.fileType ?? "image"}
              outputUrl={
                job.outputKey && CLOUDFRONT
                  ? `https://${CLOUDFRONT}/${job.outputKey}`
                  : undefined
              }
              createdAt={job.createdAt}
            />
          ))}
        </div>
      )}
    </div>
  );
}
