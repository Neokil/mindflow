import { invite } from "./accounts";
import { test, expect } from "@playwright/test";
import { activate, create, login } from "./helpers";
test("transfer ownership leaves the previous owner with full access", async ({
  page,
}, info) => {
  await login(page);
  const token = await invite(
    page,
    `transfer-${info.project.name}-${Date.now()}@test.invalid`,
  );
  const accepted = await page.request.post("/api/v1/auth/accept-invite", {
    data: {
      token,
      password: "a transfer test passphrase",
      displayName: "New Workspace Owner",
    },
  });
  expect(accepted.ok()).toBe(true);
  await create(page, `Ownership ${info.project.name}`);
  await activate(
    page,
    page.getByRole("button", { name: "Workspace sharing", exact: true }),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Transfer ownership", exact: true }),
  );
  const dialog = page.getByRole("dialog", {
    name: "Transfer ownership",
    exact: true,
  });
  await dialog
    .getByLabel("Search users", { exact: true })
    .fill("New Workspace Owner");
  await activate(
    page,
    dialog.getByRole("radio", { name: "New Workspace Owner", exact: true }),
  );
  const transfer = dialog.getByRole("button", {
    name: "Transfer ownership",
    exact: true,
  });
  await expect(transfer).toBeDisabled();
  await activate(
    page,
    dialog.getByRole("checkbox", {
      name: "Confirm ownership transfer",
      exact: true,
    }),
  );
  await activate(page, transfer);
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await activate(
    page,
    page.getByRole("button", { name: "Workspace sharing", exact: true }),
  );
  const sharing = page.getByRole("dialog", {
    name: "Workspace sharing",
    exact: true,
  });
  await expect(sharing).toContainText("New Workspace Owner");
  await expect(
    sharing.getByRole("button", { name: "Transfer ownership", exact: true }),
  ).toHaveCount(0);
});
