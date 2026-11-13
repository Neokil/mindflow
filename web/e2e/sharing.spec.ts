import { test, expect, type Page } from "@playwright/test";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { DatabaseSync } from "node:sqlite";
import { login, create } from "./helpers";

async function invite(page: Page, email: string) {
  const status = await page.evaluate(async (address) => {
    const me = await (await fetch("/api/v1/auth/me")).json();
    return (
      await fetch("/api/v1/admin/users", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-CSRF-Token": me.csrf,
        },
        body: JSON.stringify({ email: address, role: "user" }),
      })
    ).status;
  }, email);
  expect(status).toBe(200);
  // Read only the disposable fixture's captured email job; never add a production token endpoint.
  const directory = readFileSync("../.cache/e2e/data-dir", "utf8");
  const database = new DatabaseSync(join(directory, "mindflow.sqlite"), {
    readOnly: true,
  });
  try {
    const row = database
      .prepare(
        "SELECT payload FROM mail_jobs WHERE user_id=(SELECT id FROM users WHERE email=?)",
      )
      .get(email);
    const body = JSON.parse(String(row?.payload)).body as string;
    return body.split("/").at(-1)!.split("\n")[0];
  } finally {
    database.close();
  }
}
async function canvasClick(page: Page, x: number, y: number) {
  const box = await page.getByTestId("canvas").boundingBox();
  if (!box) throw new Error("Canvas missing");
  await page.mouse.click(box.x + x, box.y + y);
}

test("shares roles, discusses points and fences writers across users", async ({
  page,
  browser,
}, info) => {
  test.setTimeout(90000);
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("dialog", (dialog) => dialog.accept());
  await login(page);
  const name = `Shared ideas ${info.project.name} ${Date.now()}`;
  await create(page, name);
  await page.getByRole("button", { name: "Rectangle", exact: true }).click();
  await canvasClick(page, 450, 300);
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const email = `guest-${info.project.name}-${Date.now()}@test.invalid`;
  const password = "a shared test passphrase";
  const token = await invite(page, email);
  const guestContext = await browser.newContext({
    viewport: { width: 1440, height: 1000 },
  });
  const guest = await guestContext.newPage();
  guest.on("pageerror", (error) => errors.push(error.message));
  guest.on("dialog", (dialog) => dialog.accept());
  try {
    await guest.goto(`http://127.0.0.1:8087/#accept-invite/${token}`);
    await guest.getByLabel("Password", { exact: true }).fill(password);
    await guest
      .getByRole("button", { name: "Set password", exact: true })
      .click();
    await guest.getByLabel("Email address", { exact: true }).fill(email);
    await guest.getByLabel("Password", { exact: true }).fill(password);
    await guest.getByRole("button", { name: "Sign in", exact: true }).click();
    await expect(
      guest.getByRole("heading", { name: "Workspaces", exact: true }),
    ).toBeVisible();
    await expect(
      guest.getByRole("button", { name: `Open ${name}`, exact: true }),
    ).toHaveCount(0);
    await page
      .getByRole("button", { name: "Workspace sharing", exact: true })
      .click();
    await page.getByLabel("Share with email", { exact: true }).fill(email);
    await page
      .getByLabel("Sharing access", { exact: true })
      .selectOption("read");
    await page.getByRole("button", { name: "Add user", exact: true }).click();
    await expect(
      page.getByLabel(`Access for ${email}`, { exact: true }),
    ).toHaveValue("read");
    await page
      .getByRole("button", { name: "Close dialog", exact: true })
      .click();
    await expect(
      page.getByRole("button", { name: "Rectangle", exact: true }),
    ).toBeEnabled();
    await guest
      .getByRole("button", { name: `Open ${name}`, exact: true })
      .click();
    await expect(guest.getByTestId("element-count")).toHaveText("1 element");
    await guest
      .getByRole("button", { name: "Workspace sharing", exact: true })
      .click();
    await expect(
      guest.getByText("Workspace owner", { exact: true }),
    ).toBeVisible();
    await expect(
      guest
        .getByRole("dialog")
        .getByText("owner@test.invalid", { exact: true }),
    ).toBeVisible();
    await guest
      .getByRole("button", { name: "Close dialog", exact: true })
      .click();
    await expect(
      guest.getByRole("button", { name: "Rectangle", exact: true }),
    ).toBeDisabled();
    await expect(
      guest.getByRole("button", { name: "Add comment", exact: true }),
    ).toBeDisabled();
    await expect(
      guest.getByRole("button", { name: "Take control", exact: true }),
    ).toBeHidden();
    await page
      .getByRole("button", { name: "Workspace sharing", exact: true })
      .click();
    await page
      .getByLabel(`Access for ${email}`, { exact: true })
      .selectOption("comment");
    await page
      .getByRole("button", { name: "Close dialog", exact: true })
      .click();
    await expect(
      guest.getByRole("button", { name: "Add comment", exact: true }),
    ).toBeEnabled();
    await guest
      .getByRole("button", { name: "Add comment", exact: true })
      .click();
    await canvasClick(guest, 600, 350);
    await guest
      .getByLabel("Comment text", { exact: true })
      .fill("What do you think of this point?");
    await guest
      .getByRole("button", { name: "Post comment", exact: true })
      .click();
    const pin = page.getByRole("button", {
      name: `Comment by ${email}`,
      exact: true,
    });
    await pin.hover();
    await expect(
      page.getByText("What do you think of this point?", { exact: true }),
    ).toBeVisible();
    await page
      .getByLabel("Reply to comment", { exact: true })
      .fill("Let's keep it.");
    await page.getByRole("button", { name: "Reply", exact: true }).click();
    await guest
      .getByRole("button", { name: `Comment by ${email}`, exact: true })
      .hover();
    await expect(
      guest.getByText("Let's keep it.", { exact: true }),
    ).toBeVisible();
    await guest.getByRole("button", { name: "Resolve", exact: true }).click();
    await expect(
      page.getByRole("button", { name: `Comment by ${email}`, exact: true }),
    ).toBeHidden();
    await page
      .getByRole("button", { name: "Show resolved comments", exact: true })
      .click();
    await pin.hover();
    await page.getByRole("button", { name: "Reopen", exact: true }).click();
    await page
      .getByRole("button", { name: "Workspace sharing", exact: true })
      .click();
    await page
      .getByLabel(`Access for ${email}`, { exact: true })
      .selectOption("full");
    await page
      .getByRole("button", { name: "Close dialog", exact: true })
      .click();
    await expect(
      guest.getByRole("button", { name: "Take control", exact: true }),
    ).toBeVisible();
    await guest
      .getByRole("button", { name: "Take control", exact: true })
      .click();
    await expect(
      page.getByRole("dialog", {
        name: `${email} is requesting session control`,
        exact: true,
      }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Deny", exact: true }).click();
    await expect(
      guest.getByRole("dialog", {
        name: "Control request denied",
        exact: true,
      }),
    ).toBeVisible();
    await guest.getByRole("button", { name: "OK", exact: true }).click();
    await guest
      .getByRole("button", { name: "Take control", exact: true })
      .click();
    await page.getByRole("button", { name: /^Accept \(\d+\)$/ }).click();
    await expect(
      guest.getByRole("button", { name: "Circle", exact: true }),
    ).toBeEnabled();
    await expect(
      page.getByRole("button", { name: "Rectangle", exact: true }),
    ).toBeDisabled();
    await guest.getByRole("button", { name: "Circle", exact: true }).click();
    await canvasClick(guest, 800, 500);
    await expect(guest.getByTestId("save-status")).toHaveText("Saved");
    await expect(page.getByTestId("element-count")).toHaveText("2 elements");
    await page.screenshot({
      path: `../.cache/e2e/shared-${info.project.name}.png`,
    });
    await page
      .getByRole("button", { name: "Workspace sharing", exact: true })
      .click();
    await page
      .getByRole("button", { name: `Remove ${email}`, exact: true })
      .click();
    await page
      .getByRole("button", { name: "Close dialog", exact: true })
      .click();
    await expect(guest.getByTestId("canvas")).toHaveCount(0);
    await page
      .getByRole("button", { name: "Take control", exact: true })
      .click();
    await expect(
      page.getByRole("button", { name: "Rectangle", exact: true }),
    ).toBeEnabled();
    expect(errors).toEqual([]);
  } finally {
    await guestContext.close();
  }
});
