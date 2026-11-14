import { expect, type Page, type Locator } from "@playwright/test";
import { readFileSync } from "node:fs";
const email = "owner@test.invalid",
  password = "a private test passphrase";
export async function activate(page: Page, target: Locator) {
  if (await page.evaluate(() => "ontouchstart" in window)) await target.tap();
  else await target.click();
}
export async function login(page: Page) {
  await page.goto("/");
  if (page.context().browser()?.browserType().name() === "firefox")
    await page.bringToFront();
  await expect(
    page
      .getByLabel("Setup credential")
      .or(page.getByRole("button", { name: "Sign in", exact: true })),
  ).toBeVisible();
  if (await page.getByLabel("Setup credential").isVisible()) {
    await page
      .getByLabel("Setup credential")
      .fill(readFileSync("../.cache/e2e/bootstrap", "utf8"));
    await page.getByLabel("Email address", { exact: true }).fill(email);
    await page.getByLabel("Password", { exact: true }).fill(password);
    await page
      .getByLabel("Public application URL")
      .fill("http://127.0.0.1:8087");
    await page.getByLabel("SMTP host").fill("127.0.0.1");
    await page.getByLabel("Port", { exact: true }).fill("9");
    await page.getByLabel("Sender email").fill("mindflow@test.invalid");
    await activate(
      page,
      page.getByRole("button", { name: "Create mindflow", exact: true }),
    );
  }
  await expect(
    page.getByRole("button", { name: "Sign in", exact: true }),
  ).toBeVisible();
  await page.getByLabel("Email address", { exact: true }).fill(email);
  await page.getByLabel("Password", { exact: true }).fill(password);
  await activate(
    page,
    page.getByRole("button", { name: "Sign in", exact: true }),
  );
  await expect(
    page.getByRole("heading", { name: "Workspaces", exact: true }),
  ).toBeVisible();
}
export async function create(page: Page, name: string) {
  await activate(
    page,
    page.getByRole("button", { name: "New workspace", exact: true }),
  );
  await page.getByLabel("Name", { exact: true }).fill(name);
  await activate(page, page.getByRole("button", { name: "Save", exact: true }));
  await expect(page.getByTestId("canvas")).toBeVisible();
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
}

export async function selectTool(page: Page, name: string) {
  if (["Rectangle", "Circle", "Star"].includes(name)) {
    await activate(
      page,
      page.getByRole("button", { name: "Shapes", exact: true }),
    );
    await activate(
      page,
      page.getByRole("menuitemradio", { name, exact: true }),
    );
  } else await activate(page, page.getByRole("button", { name, exact: true }));
}
