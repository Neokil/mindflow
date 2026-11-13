import { test, expect, type Page } from "@playwright/test";
import { invite } from "./accounts";
import { login, create } from "./helpers";

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
  const self = await (await page.request.get("/api/v1/auth/me")).json();
  const name = `Shared ideas ${info.project.name} ${Date.now()}`;
  await create(page, name);
  await page.getByRole("button", { name: "Rectangle", exact: true }).click();
  await canvasClick(page, 450, 300);
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const email = `guest-${info.project.name}-${Date.now()}@test.invalid`;
  const password = "a shared test passphrase";
  const displayName = "Sam Tester";
  const token = await invite(page, email);
  const guestContext = await browser.newContext({
    viewport: { width: 1440, height: 1000 },
  });
  const guest = await guestContext.newPage();
  guest.on("pageerror", (error) => errors.push(error.message));
  guest.on("dialog", (dialog) => dialog.accept());
  try {
    await guest.goto(`http://127.0.0.1:8087/#accept-invite/${token}`);
    await guest.getByLabel("Display name", { exact: true }).fill(displayName);
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
    await page.getByLabel("Search users", { exact: true }).fill("sam");
    await page.getByRole("radio", { name: displayName, exact: true }).check();
    await page
      .getByLabel("Sharing access", { exact: true })
      .selectOption("read");
    await page.getByRole("button", { name: "Add user", exact: true }).click();
    await expect(
      page.getByLabel(`Access for ${displayName}`, { exact: true }),
    ).toHaveValue("read");
    await page
      .getByRole("button", { name: "Close dialog", exact: true })
      .click();
    await expect(
      page.getByRole("button", { name: "Rectangle", exact: true }),
    ).toBeEnabled();
    await guest.goto(page.url());
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
        .getByText(self.user.displayName, { exact: true }),
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
      .getByLabel(`Access for ${displayName}`, { exact: true })
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
      name: `Comment by ${displayName}`,
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
      .getByRole("button", { name: `Comment by ${displayName}`, exact: true })
      .hover();
    await expect(
      guest.getByText("Let's keep it.", { exact: true }),
    ).toBeVisible();
    await guest.getByRole("button", { name: "Resolve", exact: true }).click();
    await expect(
      page.getByRole("button", {
        name: `Comment by ${displayName}`,
        exact: true,
      }),
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
      .getByLabel(`Access for ${displayName}`, { exact: true })
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
        name: `${displayName} is requesting session control`,
        exact: true,
      }),
    ).toBeVisible();
    const acceptBox = await page
      .getByRole("button", { name: /^Accept \(\d+\)$/ })
      .boundingBox();
    const denyBox = await page
      .getByRole("button", { name: "Deny", exact: true })
      .boundingBox();
    expect(acceptBox!.x - (denyBox!.x + denyBox!.width)).toBeGreaterThanOrEqual(
      10,
    );
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
      .getByRole("button", { name: `Remove ${displayName}`, exact: true })
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
