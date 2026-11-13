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
  const expandedViewportHeight = await page.evaluate(() => innerHeight);
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
  const hide = page.getByRole("button", {
    name: "Hide app header and pages",
    exact: true,
  });
  if (isMobile) await hide.tap();
  else {
    const handle = (await hide.boundingBox())!;
    await page.mouse.move(
      handle.x + handle.width / 2,
      handle.y + handle.height / 2,
    );
    await page.mouse.down();
    await expect(hide).toHaveCSS("transform", "none");
    expect((await hide.boundingBox())!.x).toBeCloseTo(handle.x);
    await page.mouse.up();
  }
  const restore = page.getByRole("button", {
    name: "Show app header and pages",
    exact: true,
  });
  await expect(restore).toHaveAttribute("aria-expanded", "false");
  if (info.project.name !== "ipad-webkit")
    await expect
      .poll(() =>
        page.evaluate(
          () => document.fullscreenElement === document.documentElement,
        ),
      )
      .toBe(true);
  await expect(page.locator(".app-header")).toBeHidden();
  await expect(
    page.getByRole("button", { name: "Back to workspaces", exact: true }),
  ).toBeHidden();
  await expect(
    page.getByRole("tablist", { name: "Workspace pages", exact: true }),
  ).toBeHidden();
  await expect.poll(async () => (await canvas.boundingBox())!.y).toBe(0);
  const collapsed = (await canvas.boundingBox())!;
  const viewportHeight = await page.evaluate(() => innerHeight);
  // Native fullscreen can resize Firefox to the physical test screen.
  const viewportChange = viewportHeight - expandedViewportHeight;
  expect(collapsed.height).toBeGreaterThan(
    expanded.height + viewportChange + 150,
  );
  expect(collapsed.height).toBe(viewportHeight);
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
  if (info.project.name !== "ipad-webkit")
    await expect
      .poll(() => page.evaluate(() => document.fullscreenElement === null))
      .toBe(true);
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

for (const mode of ["allowed", "denied", "unsupported", "delayed"] as const) {
  test(`fullscreen handles ${mode} requests`, async ({ page, isMobile }) => {
    const activate = (target: Locator) =>
      isMobile ? target.tap() : target.click();
    await login(page);
    await page.evaluate((mode) => {
      let current: Element | null = null;
      Object.defineProperty(document, "fullscreenEnabled", {
        get: () => mode !== "unsupported",
      });
      Object.defineProperty(document, "fullscreenElement", {
        get: () => current,
      });
      document.documentElement.requestFullscreen = async () => {
        document.documentElement.dataset.fullscreenRequests = "1";
        if (mode === "denied") throw new Error("Fullscreen denied");
        if (mode === "delayed")
          await new Promise<void>((resolve) =>
            document.addEventListener("complete-fullscreen", () => resolve(), {
              once: true,
            }),
          );
        current = document.documentElement;
        document.dispatchEvent(new Event("fullscreenchange"));
      };
      document.exitFullscreen = async () => {
        current = null;
        document.documentElement.dataset.fullscreenExits = "1";
        document.dispatchEvent(new Event("fullscreenchange"));
      };
    }, mode);
    await create(page, `Fullscreen ${mode} ${Date.now()}`);
    const canvas = page.getByTestId("canvas");
    const original = await canvas.elementHandle();
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
    await expect(restore).toBeVisible();
    await expect(page.locator(".app-header")).toBeHidden();
    if (mode !== "unsupported")
      await expect(page.locator("html")).toHaveAttribute(
        "data-fullscreen-requests",
        "1",
      );
    if (mode === "allowed")
      await expect
        .poll(() => page.evaluate(() => !!document.fullscreenElement))
        .toBe(true);
    else
      expect(await page.evaluate(() => !!document.fullscreenElement)).toBe(
        false,
      );
    await activate(restore);
    if (mode === "delayed")
      await page.evaluate(() =>
        document.dispatchEvent(new Event("complete-fullscreen")),
      );
    await expect(page.locator(".app-header")).toBeVisible();
    await expect
      .poll(() => page.evaluate(() => !!document.fullscreenElement))
      .toBe(false);
    if (mode === "allowed" || mode === "delayed")
      await expect(page.locator("html")).toHaveAttribute(
        "data-fullscreen-exits",
        "1",
      );
    if (mode === "unsupported")
      await expect(page.locator("html")).not.toHaveAttribute(
        "data-fullscreen-requests",
      );
    expect(
      await canvas.evaluate(
        (element, original) => element === original,
        original,
      ),
    ).toBe(true);
  });
}
