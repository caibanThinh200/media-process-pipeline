---
name: "ui"
description: "Enhances and updates the Next.js frontend UI in response to a new frontend defect (docs/DEFECTS.md) or a new vision/change item (docs/VISION.md). Invoke with /ui, optionally followed by an ID (e.g. /ui DEF-001, /ui UPD-003) or a free-text request."
---

# UI Agent (`/ui`)

Implements frontend UI defects and vision changes for the media pipeline, then verifies them visually and updates the trackers.

## Invocation

- `/ui` – scan trackers and pick up all open frontend items.
- `/ui DEF-001` / `/ui FEAT-002` / `/ui UPD-003` – work on one specific item.
- `/ui <free text>` – ad-hoc UI change; create a tracker entry first, then implement.

## Sources of truth

- `docs/DEFECTS.md` – bugs/regressions. Pick rows with **Area = frontend** and Status `Open` / `Investigating` / `In Progress`.
- `docs/VISION.md` – features (`FEAT-NNN`) and updates (`UPD-NNN`). Pick rows with **Area = frontend** and Status `Planned` / `In Progress` (ask before taking `Idea`).
- Defects go in DEFECTS.md, new features/changes go in VISION.md. Never mix them.

## Frontend stack (respect it)

- Next.js 14 App Router in `frontend/` (`app/page.tsx` upload flow, `app/gallery/page.tsx` gallery, `app/api/*` BFF routes).
- Tailwind CSS v4 + shadcn/ui primitives in `components/ui/` (badge, button, card, progress, separator), `lucide-react` icons, `cn()` from `lib/utils.ts`.
- Feature components: `UploadZone`, `ProcessingStatus`, `MediaCard`.
- Global styles/tokens/fonts in `app/globals.css` and `app/layout.tsx`.
- Do not add new UI libraries unless the item requires it; prefer existing shadcn primitives.

## Workflow

1. **Intake** – Read the target item (details section included). Restate the expected UI outcome and acceptance criteria in 2–3 lines. If ambiguous, ask before coding.
2. **Mark in progress** – Set the item's Status to `In Progress` in its tracker.
3. **Inspect** – Read the affected pages/components and `globals.css`. Reuse existing tokens, components, and patterns.
4. **Implement** – Make the smallest cohesive change:
   - Prefer design tokens (CSS variables / Tailwind theme) over hard-coded values.
   - Keep components small; extract when a block is reused.
   - Preserve behavior: upload → progress → processing status → result; polling and retry logic must not regress.
   - Include loading, empty, error, and success states.
5. **Quality bar**
   - Responsive (mobile → desktop), keyboard accessible, visible focus, sufficient color contrast, alt text on media, `aria-live` for status changes.
   - Respect `prefers-reduced-motion`; animations subtle and purposeful.
   - Optimized media (`next/image` or sized previews where applicable).
   - No secrets or backend-only env vars in client code.
6. **Verify**
   - The dev server (`npm run dev` in `frontend/`) is usually already running; do not start a second one.
   - `cd frontend && npx tsc --noEmit` and `npm run lint` must pass.
   - Visually check affected pages in the browser (desktop + mobile width) and take a screenshot.
   - If selectors/text used by e2e tests changed, update `e2e/tests` and run Playwright for the affected specs.
7. **Update trackers**
   - Defect: add what was changed and how to verify under its detail section; set Status `Fixed` once visually verified, move the row to **Resolved Defects** with the commit/PR.
   - Vision item: set Status `Done`.
   - Leave unrelated rows and docs untouched.
8. **Report** – Brief summary: files changed, before/after notes (screenshots if available), verification results, open questions.

## Guardrails

- Frontend only. If the root cause is backend/terraform, stop, tell the user, and log/adjust the defect's Area instead of patching around it in the UI.
- Do not change API contracts or `app/api/*` response shapes without being asked.
- Do not delete or reword existing comments/docs unrelated to the change.
- Do not commit or push unless asked.
- Ask before large redesigns that go beyond the scope of the item.

## Example prompts

- `/ui` – "Pick up all open frontend items and implement them."
- `/ui DEF-001` – "Finish the Barlow font + gradient background and verify visually."
- `/ui UPD-003` – "Implement this vision update on the gallery page."
- `/ui Make the upload zone show a drag-over highlight` – ad-hoc change, tracked first.
