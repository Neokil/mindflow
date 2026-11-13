import { expect, test } from "@playwright/test";
import { invite } from "./accounts";
import { activate, create, login } from "./helpers";

test("display names persist and sharing searches names without exposing emails", async ({
  page,
}, info) => {
  await login(page);
  const initial = await (await page.request.get("/api/v1/auth/me")).json();
  try {
    await activate(
      page,
      page.getByRole("button", { name: "Your profile", exact: true }),
    );
    await page.getByLabel("Display name", { exact: true }).fill("River Stone");
    await activate(
      page,
      page.getByRole("button", { name: "Save name", exact: true }),
    );
    await expect(
      page.getByText("Display name saved.", { exact: true }),
    ).toBeVisible();
    await activate(
      page,
      page.getByRole("button", { name: "Close dialog", exact: true }),
    );
    await page.reload();
    await expect(
      page.getByRole("button", { name: "Your profile", exact: true }),
    ).toContainText("River Stone");
    for (const [index, displayName] of [
      "Élodie Chen",
      "Jamie Lee",
      "Jamie Lee",
    ].entries()) {
      const email = `directory-${info.project.name}-${Date.now()}-${index}@test.invalid`;
      const token = await invite(page, email);
      const response = await page.request.post("/api/v1/auth/accept-invite", {
        data: { token, password: "a profile test passphrase", displayName },
      });
      expect(response.ok()).toBe(true);
    }
    await invite(
      page,
      `pending-${info.project.name}-${Date.now()}@test.invalid`,
    );
    await create(page, `Named sharing ${Date.now()}`);
    await activate(
      page,
      page.getByRole("button", { name: "Workspace sharing", exact: true }),
    );
    const dialog = page.getByRole("dialog", {
      name: "Workspace sharing",
      exact: true,
    });
    await expect(dialog).toContainText("River Stone");
    await expect(dialog).not.toContainText("@test.invalid");
    const search = dialog.getByLabel("Search users", { exact: true });
    await expect(dialog.getByRole("radio")).toHaveCount(3);
    const firstOption = dialog.locator(".sharing-user-option").first();
    const firstName = firstOption.locator(":scope > span").last();
    await expect(firstName).toBeInViewport();
    const row = (await firstOption.boundingBox())!;
    const label = (await firstName.boundingBox())!;
    expect(label.x + label.width).toBeLessThanOrEqual(row.x + row.width);
    expect(row.height).toBeLessThanOrEqual(64);

    await expect(
      dialog.getByRole("radio", { name: "Jamie Lee", exact: true }),
    ).toHaveCount(2);
    await search.fill("eLoDiE");
    await expect(dialog.getByRole("radio")).toHaveCount(1);
    await activate(
      page,
      dialog.getByRole("radio", { name: "Élodie Chen", exact: true }),
    );
    const add = dialog.getByRole("button", { name: "Add user", exact: true });
    await expect(add).toBeEnabled();
    await search.fill("Jamie");
    await expect(add).toBeDisabled();
    const recipient = dialog
      .getByRole("radio", { name: "Jamie Lee", exact: true })
      .nth(1);
    const selectedID = await recipient.getAttribute("value");
    await activate(page, recipient);
    const grant = page.waitForRequest(
      (request) =>
        request.method() === "PUT" && request.url().endsWith("/members"),
    );
    await activate(page, add);
    expect((await grant).postDataJSON()).toEqual({
      userId: selectedID,
      role: "read",
    });
    await expect(
      dialog.getByLabel("Access for Jamie Lee", { exact: true }),
    ).toHaveValue("read");
    await expect(
      dialog.getByRole("radio", { name: "Jamie Lee", exact: true }),
    ).toHaveCount(1);
    await expect(dialog).not.toContainText("@test.invalid");
    await dialog
      .getByLabel("Access for Jamie Lee", { exact: true })
      .selectOption("comment");
    await expect(
      dialog.getByLabel("Access for Jamie Lee", { exact: true }),
    ).toHaveValue("comment");
    await search.fill("does-not-exist");
    await expect(
      dialog.getByText("No users match your search.", { exact: true }),
    ).toBeVisible();
    await expect(add).toBeDisabled();
    await search.fill("");
    await expect(dialog.getByRole("radio")).toHaveCount(2);
    await page.screenshot({
      path: `../.cache/e2e/display-names-${info.project.name}.png`,
    });
    await activate(
      page,
      dialog.getByRole("button", { name: "Remove Jamie Lee", exact: true }),
    );
    await expect(
      dialog.getByLabel("Access for Jamie Lee", { exact: true }),
    ).toHaveCount(0);
  } finally {
    const restored = await page.request.patch("/api/v1/auth/profile", {
      headers: { "X-CSRF-Token": initial.csrf },
      data: { displayName: initial.user.displayName },
    });
    expect(restored.ok()).toBe(true);
  }
});
