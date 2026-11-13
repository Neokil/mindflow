import { expect, type Page } from "@playwright/test";
import { readFileSync } from "node:fs";
const email = "owner@test.invalid",
  password = "a private test passphrase";
export async function login(page: Page) {
  await page.goto("/");
  await page.bringToFront();
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
    await page
      .getByRole("button", { name: "Create mindflow", exact: true })
      .click();
  }
  await page.getByLabel("Email address", { exact: true }).fill(email);
  await page.getByLabel("Password", { exact: true }).fill(password);
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Workspaces", exact: true }),
  ).toBeVisible();
}
export async function create(page: Page, name: string) {
  await page
    .getByRole("button", { name: "New workspace", exact: true })
    .click();
  await page.getByLabel("Name", { exact: true }).fill(name);
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByTestId("canvas")).toBeVisible();
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
}
