import { expect, test } from "@playwright/test";
import type { Locator } from "@playwright/test";
import { create, login } from "./helpers";

test("collapse navigation into an arrow without losing the active canvas", async ({
  page,
  isMobile,
}, info) => {
  const activate = (target: Locator) =>
    isMobile ? target.tap() : target.click();
  await login(page);
  await create(page, `Canvas space ${info.project.name} ${Date.now()}`);
  const canvas = page.getByTestId("canvas");
  const original = await canvas.elementHandle();
  const expanded = (await canvas.boundingBox())!;
  await activate(page.getByRole("button", { name: "Rectangle", exact: true }));
  if (isMobile)
    await page.touchscreen.tap(
      expanded.x + expanded.width / 2,
      expanded.y + expanded.height / 2,
    );
  else
    await page.mouse.click(
      expanded.x + expanded.width / 2,
      expanded.y + expanded.height / 2,
    );
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  await activate(page.getByRole("button", { name: "Zoom in", exact: true }));
  const zoom = await page.locator(".zoom-value").innerText();
  await activate(
    page.getByRole("button", {
      name: "Hide app header and pages",
      exact: true,
    }),
  );
  const restore = page.getByRole("button", {
    name: "Show app header and pages",
    exact: true,
  });
  await expect(restore).toHaveAttribute("aria-expanded", "false");
  await expect(page.locator(".app-header")).toBeHidden();
  await expect(
    page.getByRole("button", { name: "Back to workspaces", exact: true }),
  ).toBeHidden();
  await expect(
    page.getByRole("tablist", { name: "Workspace pages", exact: true }),
  ).toBeHidden();
  await expect.poll(async () => (await canvas.boundingBox())!.y).toBe(0);
  const collapsed = (await canvas.boundingBox())!;
  expect(collapsed.height).toBeGreaterThan(expanded.height + 150);
  expect(collapsed.height).toBe(await page.evaluate(() => innerHeight));
  expect(
    await canvas.evaluate(
      (element, original) => element === original,
      original,
    ),
  ).toBe(true);
  await expect(page.locator(".zoom-value")).toHaveText(zoom);
  await expect(page.getByText("1 selected", { exact: true })).toBeVisible();
  const arrow = (await restore.boundingBox())!;
  const toolbar = (await page
    .getByLabel("Drawing tools", { exact: true })
    .boundingBox())!;
  expect(arrow.y + arrow.height).toBeLessThanOrEqual(toolbar.y);
  await activate(page.getByRole("button", { name: "Undo", exact: true }));
  await expect(page.getByTestId("element-count")).toHaveText("0 elements");
  await activate(page.getByRole("button", { name: "Redo", exact: true }));
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  // Deployment notices remain visible and do not overlap the restore handle.
  await page.route("**/api/v1/version", (route) =>
    route.fulfill({ json: { version: "focus-new-version" } }),
  );
  await page.evaluate(() => window.dispatchEvent(new Event("focus")));
  const notice = page.locator(".update-banner");
  await expect(notice).toBeVisible();
  const noticeBox = (await notice.boundingBox())!;
  await expect
    .poll(async () => (await canvas.boundingBox())!.y)
    .toBeCloseTo(noticeBox.y + noticeBox.height, 0);
  expect((await restore.boundingBox())!.y).toBeGreaterThanOrEqual(
    noticeBox.y + noticeBox.height,
  );
  await page.screenshot({
    path: `../.cache/e2e/focus-${info.project.name}.png`,
  });
  await activate(restore);
  await expect(page.locator(".app-header")).toBeVisible();
  await expect(
    page.getByRole("tablist", { name: "Workspace pages", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", {
      name: "Hide app header and pages",
      exact: true,
    }),
  ).toHaveAttribute("aria-expanded", "true");
  await expect(page.locator(".zoom-value")).toHaveText(zoom);
  await activate(
    page.getByRole("button", { name: "Back to workspaces", exact: true }),
  );
  await expect(
    page.getByRole("heading", { name: "Workspaces", exact: true }),
  ).toBeVisible();
  await expect(page.locator(".app-header")).toBeVisible();
  await expect(page.locator(".editor-header-toggle")).toHaveCount(0);
});
