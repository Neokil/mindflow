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
  await page.getByRole("button", { name: "Rectangle", exact: true }).click();
  const canvas = await page.getByTestId("canvas").boundingBox();
  if (!canvas) throw new Error("Canvas missing");
  await page.mouse.click(canvas.x + 450, canvas.y + 300);
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  await page
    .getByRole("button", { name: "Back to workspaces", exact: true })
    .click();
  await page.getByRole("button", { name: `Open ${name}`, exact: true }).click();
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  expect(errors).toEqual([]);
});
