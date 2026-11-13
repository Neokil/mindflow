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
    expect((await password.inputValue()).split(/[.!#\-+_?=]/)).toHaveLength(6);
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
  await expect(page.getByLabel("Password", { exact: true })).toHaveAttribute(
    "minlength",
    "8",
  );
});
