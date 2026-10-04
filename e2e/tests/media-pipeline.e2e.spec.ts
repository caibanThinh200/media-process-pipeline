import { test, expect } from "@playwright/test";
import { makePng, uploadViaUi } from "./helpers";

const PIPELINE_TIMEOUT = 120_000;

test.describe("Media pipeline — full user journey (real cloud, no mocks)", () => {
  test("image: upload → processing → complete → appears in gallery with live CDN image", async ({
    page,
  }) => {
    await uploadViaUi(page, {
      name: `e2e-${Date.now()}.png`,
      mimeType: "image/png",
      buffer: makePng(),
    });

    // Upload phase + status tracker visible
    await expect(page.getByRole("status")).toBeVisible({ timeout: 30_000 });

    // Async pipeline (SQS → image-worker → DynamoDB) finishes
    await expect(page.getByText("Processing complete!")).toBeVisible({
      timeout: PIPELINE_TIMEOUT,
    });

    // The job id must have been persisted for the gallery
    const ids = await page.evaluate(() =>
      JSON.parse(localStorage.getItem("mpp_job_ids") ?? "[]")
    );
    expect(ids).toHaveLength(1);

    // Gallery shows it as Complete and the CloudFront image really loads
    await page.locator("#nav-gallery").click();
    await expect(page).toHaveURL(/\/gallery/);

    const card = page.getByLabel(`Media job ${ids[0].slice(0, 8)}`);
    await expect(card).toBeVisible();
    await expect(card.getByText("Complete", { exact: true })).toBeVisible();

    const img = card.locator("img");
    await expect(img).toBeVisible();
    await expect
      .poll(() => img.evaluate((el: HTMLImageElement) => el.complete && el.naturalWidth), {
        timeout: 30_000,
      })
      .toBeGreaterThan(0);
  });

  test("gallery persists completed jobs across reload and can be cleared", async ({
    page,
  }) => {
    await uploadViaUi(page, {
      name: `e2e-persist-${Date.now()}.png`,
      mimeType: "image/png",
      buffer: makePng(64, 64),
    });
    await expect(page.getByText("Processing complete!")).toBeVisible({
      timeout: PIPELINE_TIMEOUT,
    });

    await page.goto("/gallery");
    await expect(page.getByText("1 job in your session")).toBeVisible();
    await page.reload();
    await expect(page.getByText("1 job in your session")).toBeVisible();

    await page.getByRole("button", { name: "Clear history" }).click();
    await expect(page.getByText("Your processed media will appear here")).toBeVisible();
  });

  test("empty gallery shows the empty state for a fresh visitor", async ({ page }) => {
    await page.goto("/gallery");
    await expect(page.getByRole("heading", { name: "Gallery" })).toBeVisible();
    await expect(page.getByText("Your processed media will appear here")).toBeVisible();
  });

  // Needs a real video file; supply one with E2E_VIDEO_FILE=/path/to/clip.mp4
  test("video: upload → FFmpeg transcode → complete → playable in gallery", async ({
    page,
  }) => {
    const videoPath = process.env.E2E_VIDEO_FILE;
    test.skip(!videoPath, "Set E2E_VIDEO_FILE to a small .mp4 to run the video flow");
    test.setTimeout(8 * 60_000);

    await uploadViaUi(page, videoPath!);
    await expect(page.getByText("Processing complete!")).toBeVisible({
      timeout: 6 * 60_000,
    });

    const ids = await page.evaluate(() =>
      JSON.parse(localStorage.getItem("mpp_job_ids") ?? "[]")
    );
    await page.goto("/gallery");
    const card = page.getByLabel(`Media job ${ids[0].slice(0, 8)}`);
    await expect(card.getByText("Complete", { exact: true })).toBeVisible();

    const video = card.locator("video");
    await expect(video).toBeVisible();
    await expect
      .poll(() => video.evaluate((el: HTMLVideoElement) => el.readyState), {
        timeout: 30_000,
      })
      .toBeGreaterThanOrEqual(1); // metadata loaded from CloudFront
    expect(await video.evaluate((el: HTMLVideoElement) => el.duration)).toBeGreaterThan(0);
  });
});
