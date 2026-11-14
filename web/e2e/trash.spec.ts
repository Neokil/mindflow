import { test, expect } from "@playwright/test";
import { activate, create, login } from "./helpers";
test("deleted pages and workspaces can be restored from trash", async ({
  page,
}, info) => {
  await login(page);
  await create(page, `Restore ${info.project.name}`);
  await activate(
    page,
    page.getByRole("button", {
      name: "Page options for Untitled",
      exact: true,
    }),
  );
  page.on("dialog", (dialog) => dialog.accept());
  await activate(
    page,
    page.getByRole("button", { name: "Delete", exact: true }),
  );
  await expect(page.getByRole("tab")).toHaveCount(0);
  await activate(
    page,
    page
      .getByRole("button", { name: "Back to workspaces", exact: true })
      .first(),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Trash", exact: true }),
  );
  await expect(page.getByRole("dialog")).toContainText("Untitled");
  await activate(
    page,
    page.getByRole("button", { name: "Restore Untitled", exact: true }),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Close dialog", exact: true }),
  );
  await activate(
    page,
    page.getByRole("button", {
      name: `Open Restore ${info.project.name}`,
      exact: true,
    }),
  );
  await expect(
    page.getByRole("tab", { name: "Untitled", exact: true }),
  ).toBeVisible();
  await activate(
    page,
    page
      .getByRole("button", { name: "Back to workspaces", exact: true })
      .first(),
  );
  const card = page
    .locator(".workspace-card")
    .filter({ hasText: `Restore ${info.project.name}` });
  await activate(page, card.getByRole("button", { name: /Options for/ }));
  await activate(
    page,
    page.getByRole("button", { name: "Delete", exact: true }),
  );
  await expect(card).toHaveCount(0);
  await activate(
    page,
    page.getByRole("button", { name: "Trash", exact: true }),
  );
  await activate(
    page,
    page.getByRole("button", {
      name: `Restore Restore ${info.project.name}`,
      exact: true,
    }),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Close dialog", exact: true }),
  );
  await expect(card).toBeVisible();
});
