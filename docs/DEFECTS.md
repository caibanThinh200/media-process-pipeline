# Defect Tracker

Track bugs and regressions here. New features and roadmap ideas go in [VISION.md](./VISION.md), not here.

## How to use

1. Add a new row to the **Open Defects** table with the next ID (`DEF-NNN`).
2. Fill out a detail section below the table, using the template.
3. When fixed, move the row to **Resolved Defects** and record the fix commit or PR.

**Severity:** `Critical` (data loss, outage, security) · `High` (core flow broken) · `Medium` (workaround exists) · `Low` (cosmetic)
**Status:** `Open` · `Investigating` · `In Progress` · `Fixed` · `Won't Fix`
**Area:** `frontend` · `backend` · `video-worker` · `terraform` · `e2e` · `observability` · `other`

## Open Defects

| ID | Title | Area | Severity | Status | Reported |
|----|-------|------|----------|--------|----------|
| DEF-001 | Enhance frontend UI (Barlow font + gradient background) | frontend | Low | In Progress | 2026-10-05 |
| DEF-002 | Uploaded image is not optimized (output identical to original) | backend | High | Investigating | 2026-10-05 |

## Resolved Defects

| ID | Title | Area | Severity | Fixed in | Resolved |
|----|-------|------|----------|----------|----------|
| – | – | – | – | – | – |

---

## Defect Details

### DEF-001: Enhance frontend UI (Barlow font + gradient background)

- **Area:** frontend
- **Severity:** Low (visual polish)
- **Status:** In Progress (implemented, pending visual verification)
- **Reported:** 2026-10-05

**Steps to reproduce**
1. Run `npm run dev` in `frontend/`.
2. Open the app at first launch (`/`).
3. Observe the default look.

**Expected behavior**
- Text uses the **Barlow** font family.
- The page background has a gradient that gives the layout more style.

**Actual behavior**
- The UI looks plain on first launch.
- A flat `bg-background` color is used, with no gradient.
- No font is loaded. `--font-sans` in `globals.css` references itself, so the browser default applies.

**Environment / logs**
- Next.js app router, Tailwind v4, shadcn, dark theme forced via `<html className="dark">`.

**Suspected cause / notes**
- [layout.tsx](../frontend/app/layout.tsx) loads no font and sets `bg-background` on `<body>`.
- [globals.css](../frontend/app/globals.css) has a circular `--font-sans: var(--font-sans)` in `@theme inline`.
- Scope is deliberately small. Further UI improvements will be separate defects or `UPD` items in VISION.md.

**Fix plan**
1. In `layout.tsx`, load Barlow with `next/font/google` (weights 400/500/600/700, `variable: "--font-barlow"`) and add the variable class to `<html>`.
2. In `globals.css`, set `--font-sans: var(--font-barlow), ui-sans-serif, system-ui, sans-serif` to fix the circular reference.
3. Add a gradient to the body background in `globals.css`, for example a dark indigo-to-violet diagonal gradient built from the existing `--primary` and `--accent` hues. Use `background-attachment: fixed` so it doesn't tile or cut off on long pages.
4. Keep the header and card translucency readable over the gradient (check contrast).
5. Verify `/` and `/gallery` in the dev server.

**Acceptance criteria**
- [ ] Barlow renders on all pages (check computed font-family in devtools).
- [ ] A gradient background is visible on `/` and `/gallery` and doesn't tile or cut off when scrolling.
- [ ] Text contrast stays readable.
- [ ] No console or build errors.

---

### DEF-002: Uploaded image is not optimized (output identical to original)

- **Area:** backend (image-worker)
- **Severity:** High (core flow produces no useful result)
- **Status:** Fix implemented (awaiting deploy verification)
- **Reported:** 2026-10-05

**Steps to reproduce**
1. Upload an image through the frontend.
2. Wait for the job to reach COMPLETE.
3. Compare the generated output with the original.

**Expected behavior**
- The output image is optimized (resized and/or compressed, smaller file size).

**Actual behavior**
- A new object is created under `output/<jobId>/`, but its bytes are identical to the original.

**Environment / logs**
- Check the `bytes=` value in the image-worker COMPLETE log line. It should equal the raw object size.

**Suspected cause / notes**
- Confirmed in code: [main.go](../backend/cmd/image-worker/main.go#L174-L182) is a passthrough copy. The comment reads "Phase 3: copy raw bytes as-is; Phase 6 will add resize/compress/watermark". No image processing is implemented.
- Content type is also hardcoded to `application/octet-stream`.

**Fix plan**
1. Add `backend/internal/imageproc` with `Optimize(data, Options) ([]byte, contentType, error)`. Pipeline: `DecodeConfig` guard (reject > ~50 MP), decode, apply EXIF orientation, resize so the longest side is at most 1920px (never upscale), draw the watermark, encode as lossy WebP (quality 80). Use `github.com/gen2brain/webp` (WASM, no cgo) for encoding.
2. Convert every input format to WebP. Preserve PNG alpha. Animated GIFs are flattened to the first frame (known limitation).
3. Watermark is on by default. It is semi-transparent text "tAI" in the bottom-right corner, about 4% of image width, drawn with an embedded TTF. Config via env vars `WATERMARK_ENABLED` (default `true`), `WATERMARK_TEXT` (default `tAI`), and `WATERMARK_OPACITY` (default ~0.5). `Options.Watermark` allows a future per-upload toggle (via `JobMessage`) with no change to `imageproc`.
4. Wire into `processRecord` after `io.ReadAll`. Output key becomes `output/<jobId>/<name>.webp` with content type `image/webp`. If decoding fails, fall back to passthrough and log a warning. Log `origBytes`, `outBytes` and `ratio` in the COMPLETE line.
5. Check that the frontend gallery and `job-status` route don't assume the original file extension.
6. Terraform: raise `image_worker_memory_mb` (256 to 512-1024) and the timeout (to 60s if needed), add the env vars, and update the function description.
7. Tests: `imageproc` unit tests (output is WebP, longest side is at most 1920, no upscaling, watermark changes pixels versus disabled, corrupt input falls back). Update `image_worker_test.go` fakes to capture the key, content type and bytes.

**Acceptance criteria**
- [ ] Uploaded JPEG/PNG produces `output/<jobId>/<name>.webp` with `Content-Type: image/webp`.
- [ ] Output is smaller than the original and its longest side is at most 1920px.
- [ ] A visible "tAI" watermark appears bottom-right.
- [ ] Gallery displays the output correctly.
- [ ] Corrupt or unsupported input does not crash the worker (passthrough with warning).
- [ ] `go test ./...` passes.

---

<!-- Copy this template for each new defect

### DEF-NNN: Title

- **Area:**
- **Severity:**
- **Status:**
- **Reported:**

**Steps to reproduce**
1.

**Expected behavior**

**Actual behavior**

**Environment / logs**

**Suspected cause / notes**

**Fix plan**

-->
