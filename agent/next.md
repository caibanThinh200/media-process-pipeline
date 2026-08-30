---
name: "media-pipeline-nextjs"
description: "Builds the Next.js frontend for the media pipeline. Invoke when creating upload UI, direct S3 upload flows, polling, gallery views, and App Router routes tied to media job status."
---

# Media Pipeline Next.js

Use this skill for the browser and BFF layer of the media processing platform built with Next.js App Router.

## Invoke when

- The user asks for upload UI, media gallery, processing state, or polling flows.
- You need to implement direct browser upload to S3 using presigned URLs.
- You need App Router routes that proxy or normalize job-status reads for the frontend.
- You are reviewing UX for upload progress, retries, completed output display, or media previews.

## Frontend responsibilities

- collect file input and validate supported media types
- request a presigned upload from the backend
- upload the file directly to S3
- poll job status until completion or failure
- render processed output from CloudFront URLs
- present upload history or gallery views

## UI guidance

- Keep upload interaction clear: select file, upload progress, processing status, final result.
- Separate transport progress from processing progress so users understand the two phases.
- Show stable job ids or references only when useful for support or debugging.
- Provide helpful failure states with retry options.
- Prefer server-safe environment variable usage and avoid exposing secrets.

## App Router guidance

- Use `app/page.tsx` for upload-first flow.
- Use `app/gallery/page.tsx` for processed assets or job history display.
- Use `app/api/job-status/route.ts` or equivalent BFF endpoints when the browser should not call backend services directly.
- Keep shared UI in `components/` with small, focused components like `UploadZone`, `ProcessingStatus`, and `MediaCard`.

## Review checklist

- Is direct-to-S3 upload implemented without sending the file through the app server?
- Are polling intervals and stop conditions sensible?
- Are large previews handled efficiently?
- Are success and failure states obvious to the user?

## Expected outputs

- Next.js pages
- reusable UI components
- polling hooks or helpers
- upload and status route handlers

## Example prompts

- `Create the Next.js upload page and polling flow for this media pipeline.`
- `Build reusable components for upload, processing status, and media cards.`
- `Review this App Router upload flow for direct-to-S3 best practices.`
