import { test, expect } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { create, login, selectTool } from "./helpers";

test("exports a selected canvas region without changing its content", async ({
  page,
}) => {
  await login(page);
  await create(page, `Screenshot ${Date.now()}`);
  await selectTool(page, "Rectangle");
  const canvas = (await page.getByTestId("canvas").boundingBox())!;
  await page.mouse.move(canvas.x + 300, canvas.y + 250);
  await page.mouse.down();
  await page.mouse.move(canvas.x + 500, canvas.y + 400);
  await page.mouse.up();
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  await page
    .getByRole("button", { name: "Export screenshot", exact: true })
    .click();
  const overlay = page.getByRole("region", { name: "Screenshot selection" });
  await expect(overlay).toBeVisible();
  await page.mouse.move(canvas.x + 280, canvas.y + 230);
  await page.mouse.down();
  await page.mouse.move(canvas.x + 520, canvas.y + 420);
  await page.mouse.up();
  const download = page.waitForEvent("download");
  await page.getByRole("button", { name: "Download PNG", exact: true }).click();
  const file = await download;
  expect(file.suggestedFilename()).toBe("mindflow-screenshot.png");
  const bytes = await readFile((await file.path())!);
  expect([...bytes.subarray(0, 8)]).toEqual([137, 80, 78, 71, 13, 10, 26, 10]);
  const scale = await page.evaluate(() => devicePixelRatio);
  expect(bytes.readUInt32BE(16)).toBe(Math.ceil(240 * scale));
  expect(bytes.readUInt32BE(20)).toBe(Math.ceil(190 * scale));
  await expect(overlay.getByRole("status")).toHaveText(
    "Screenshot downloaded.",
  );
  if (page.context().browser()?.browserType().name() === "chromium") {
    await page
      .context()
      .grantPermissions(["clipboard-read", "clipboard-write"]);
    await overlay
      .getByRole("button", { name: "Copy to clipboard", exact: true })
      .click();
    await expect(overlay.getByRole("status")).toHaveText(
      "Screenshot copied to clipboard.",
    );
    expect(
      await page.evaluate(async () =>
        (await navigator.clipboard.read()).flatMap((item) => [...item.types]),
      ),
    ).toContain("image/png");
  }
  await overlay.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(overlay).toBeHidden();
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
});
