"use client";

import { useState, useCallback } from "react";
import { UploadZone, type CompletedJob } from "@/components/UploadZone";
import { MediaCard } from "@/components/MediaCard";
import { Separator } from "@/components/ui/separator";

export default function HomePage() {
  const [latestJob, setLatestJob] = useState<CompletedJob | null>(null);

  const handleComplete = useCallback((job: CompletedJob) => {
    setLatestJob(job);
  }, []);

  return (
    <div className="space-y-12">
      {/* ── Hero ── */}
      <section className="space-y-2 text-center" aria-labelledby="hero-heading">
        <h1
          id="hero-heading"
          className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl"
        >
          Upload &amp; Process Media
        </h1>
        <p className="mx-auto max-w-lg text-muted-foreground">
          Drop an image or video. It gets uploaded directly to S3, processed in the cloud,
          and delivered via CloudFront — all in seconds.
        </p>
      </section>

      {/* ── Steps ── */}
      <section aria-label="How it works">
        <ol className="mx-auto grid max-w-2xl grid-cols-3 gap-4 text-center text-sm">
          {[
            { step: "1", label: "Pick a file", desc: "Drag & drop or click to browse" },
            { step: "2", label: "Upload to S3", desc: "Direct browser-to-S3, no middleman" },
            { step: "3", label: "Get result", desc: "Processed and served via CloudFront" },
          ].map(({ step, label, desc }) => (
            <li key={step} className="flex flex-col items-center gap-2">
              <span className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/20 text-xs font-bold text-primary">
                {step}
              </span>
              <span className="font-medium text-foreground">{label}</span>
              <span className="text-xs text-muted-foreground">{desc}</span>
            </li>
          ))}
        </ol>
      </section>

      <Separator className="opacity-30" />

      {/* ── Upload zone ── */}
      <section aria-label="Upload media">
        <UploadZone onComplete={handleComplete} />
      </section>

      {/* ── Inline result (shown after COMPLETE) ── */}
      {latestJob && latestJob.outputUrl && (
        <>
          <Separator className="opacity-30" />
          <section aria-label="Processing result" className="space-y-4 animate-fade-up">
            <h2 className="text-lg font-semibold text-foreground">Result</h2>
            <div className="mx-auto max-w-sm">
              <MediaCard
                jobId={latestJob.jobId}
                status="COMPLETE"
                fileType={latestJob.fileType}
                outputUrl={latestJob.outputUrl}
                createdAt={new Date().toISOString()}
              />
            </div>
            <p className="text-center text-sm text-muted-foreground">
              Browse all processed files in the{" "}
              <a href="/gallery" className="text-primary underline-offset-2 hover:underline">
                Gallery
              </a>
              .
            </p>
          </section>
        </>
      )}
    </div>
  );
}
