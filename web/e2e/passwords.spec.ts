import { expect, test } from "@playwright/test";
import { login } from "./helpers";
test("all account password-setting screens offer local word suggestions and accept eight characters", async ({
  page,
}) => {
  await login(page);
  await page.getByRole("button", { name: "Sign out", exact: true }).click();
  for (const mode of ["accept-invite", "reset-password"]) {
    await page.goto(`/?password-form=${mode}#${mode}/test-token`);
    await page
      .getByRole("button", { name: "Suggest good password", exact: true })
      .click();
    const password = page.getByLabel("Password", { exact: true });
    await expect(password).toHaveAttribute("type", "text");
    const suggestion = await password.inputValue();
    expect(suggestion.length).toBeGreaterThanOrEqual(10);
    expect(suggestion.length).toBeLessThanOrEqual(15);
    expect(suggestion).toMatch(/^[A-Z][a-z]+(?:[.!#\-+_?=][A-Z][a-z]+){1,2}$/);
    await expect(password).toHaveAttribute("minlength", "8");
    await password.fill("12345678");
    await page.route(`**/api/v1/auth/${mode}`, async (route) => {
      expect(route.request().postDataJSON().password).toBe("12345678");
      await route.fulfill({ json: { ok: true } });
    });
    await page
      .getByRole("button", { name: "Set password", exact: true })
      .click();
    await expect(
      page.getByText("Your password is set. You can now sign in."),
    ).toBeVisible();
  }
  await page.route("**/api/v1/setup/status", (route) =>
    route.fulfill({ json: { initialized: false } }),
  );
  await page.goto("/?password-form=setup");
  await page
    .getByRole("button", { name: "Suggest good password", exact: true })
    .click();
  const setupSuggestion = await page
    .getByLabel("Password", { exact: true })
    .inputValue();
  expect(setupSuggestion.length).toBeGreaterThanOrEqual(10);
  expect(setupSuggestion.length).toBeLessThanOrEqual(15);
  await expect(page.getByLabel("Password", { exact: true })).toHaveAttribute(
    "minlength",
    "8",
  );
});
