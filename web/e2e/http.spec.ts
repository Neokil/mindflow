import { selectTool } from "./helpers";
import { test, expect } from "@playwright/test";
import { login, create } from "./helpers";

test("opens, edits and reopens a workspace over insecure HTTP", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await login(page);
  // Check real browser behavior; do not mock secure-context restrictions.
  expect(
    await page.evaluate(() => ({
      secure: window.isSecureContext,
      randomUUID: typeof crypto.randomUUID,
      getRandomValues: typeof crypto.getRandomValues,
    })),
  ).toEqual({
    secure: false,
    randomUUID: "undefined",
    getRandomValues: "function",
  });
  const name = `HTTP workspace ${Date.now()}`;
  await create(page, name);
  await selectTool(page, "Rectangle");
  const canvas = await page.getByTestId("canvas").boundingBox();
  if (!canvas) throw new Error("Canvas missing");
  await page.mouse.click(canvas.x + 450, canvas.y + 300);
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  // A thumbnail may finish loading between pressing and releasing its button.
  // Keep that timing deterministic: the control must still receive the click.
  let releasePreview!: () => void;
  const pendingPreview = new Promise<void>((resolve) => {
    releasePreview = resolve;
  });
  await page.route("**/api/v1/pages/*/document", async (route) => {
    await pendingPreview;
    await route.continue();
  });
  try {
    await page
      .getByRole("button", { name: "Back to workspaces", exact: true })
      .click();
    const preview = page.getByRole("button", {
      name: `Open ${name}`,
      exact: true,
    });
    const loading = preview.getByText("Loading preview…", { exact: true });
    await expect(loading).toBeVisible();
    await preview.hover();
    await page.mouse.down();
    releasePreview();
    await expect(preview.getByRole("img")).toBeVisible();
    await page.mouse.up();
    await expect(page.getByTestId("element-count")).toHaveText("1 element");
  } finally {
    releasePreview();
    await page.mouse.up();
  }
  expect(errors).toEqual([]);
});
