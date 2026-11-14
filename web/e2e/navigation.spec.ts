import { selectTool } from "./helpers";
import { test, expect } from "@playwright/test";
import { login, create } from "./helpers";

test("page links survive sign-in and browser back/forward", async ({
  page,
  browser,
}, info) => {
  await login(page);
  await create(page, `Linked pages ${info.project.name} ${Date.now()}`);
  await expect(page).toHaveURL(/#\/workspaces\/[a-f0-9]+\/pages\/[a-f0-9]+$/);
  const first = page.url();
  await page.getByRole("button", { name: "Add page", exact: true }).click();
  await page.getByLabel("Name", { exact: true }).fill("Project notes");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await expect(
    page.getByRole("tab", { name: "Project notes", exact: true }),
  ).toHaveAttribute("aria-selected", "true");
  const linked = page.url();
  expect(linked).not.toBe(first);
  await page.evaluate(() => history.back());
  await expect(page).toHaveURL(first);
  await expect(
    page.getByRole("tab", { name: "Untitled", exact: true }),
  ).toHaveAttribute("aria-selected", "true");
  await page.evaluate(() => history.forward());
  await expect(page).toHaveURL(linked);
  // A workspace-only link also chooses a real page when this editor is already open.
  await page.evaluate((hash) => {
    location.hash = hash;
  }, new URL(first).hash.split("/pages/")[0]);
  await expect(page).toHaveURL(first);
  await page.evaluate((hash) => {
    location.hash = hash;
  }, new URL(linked).hash);
  await expect(
    page.getByRole("tab", { name: "Project notes", exact: true }),
  ).toHaveAttribute("aria-selected", "true");
  const context = await browser.newContext();
  try {
    const guest = await context.newPage();
    await guest.goto(linked);
    await guest
      .getByLabel("Email address", { exact: true })
      .fill("owner@test.invalid");
    await guest
      .getByLabel("Password", { exact: true })
      .fill("a private test passphrase");
    await guest.getByRole("button", { name: "Sign in", exact: true }).click();
    await expect(
      guest.getByRole("tab", { name: "Project notes", exact: true }),
    ).toHaveAttribute("aria-selected", "true");
    await expect(guest).toHaveURL(linked);
    await expect(
      guest.getByRole("button", { name: "Force control", exact: true }),
    ).toBeVisible();
    guest.on("dialog", (dialog) => dialog.accept());
    await guest
      .getByRole("button", { name: "Force control", exact: true })
      .click();
    await expect(
      guest.getByRole("button", { name: "Shapes", exact: true }),
    ).toBeEnabled();
    await expect(
      page.getByRole("button", { name: "Shapes", exact: true }),
    ).toBeDisabled();
    await expect(guest.getByRole("dialog")).toHaveCount(0);
  } finally {
    await context.close();
  }
});

test("new-build banner preserves the unsaved-changes reload guard", async ({
  page,
}, info) => {
  await login(page);
  await create(page, `Update guard ${info.project.name} ${Date.now()}`);
  await expect(
    page.getByRole("link", { name: "mindflow on GitHub", exact: true }),
  ).toHaveAttribute("href", "https://github.com/Neokil/mindflow");
  await page.route("**/api/v1/version", (route) =>
    route.fulfill({ json: { version: "a-new-deployment" } }),
  );
  await page.evaluate(() => window.dispatchEvent(new Event("focus")));
  await expect(
    page.getByText(
      "A new version of mindflow is available. Reload to update.",
      { exact: true },
    ),
  ).toBeVisible();
  await page.context().setOffline(true);
  await selectTool(page, "Rectangle");
  const box = (await page.getByTestId("canvas").boundingBox())!;
  await page.mouse.click(box.x + 400, box.y + 300);
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  const warning = page.waitForEvent("dialog");
  page.on("dialog", (dialog) => dialog.dismiss());
  await page.getByRole("button", { name: "Reload", exact: true }).click();
  expect((await warning).type()).toBe("beforeunload");
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  await page.context().setOffline(false);
});
