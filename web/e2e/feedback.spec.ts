import { test, expect } from "@playwright/test";
import { activate, login, create } from "./helpers";

test("private report with screenshot, discussion and release-linked admin review", async ({
  page,
}) => {
  await login(page);
  await create(page, "Feedback example");
  await activate(
    page,
    page.getByRole("button", { name: "Feedback & Bugreport", exact: true }),
  );
  const dialog = page.getByRole("dialog");
  await activate(
    page,
    dialog.getByRole("button", { name: "New report", exact: true }),
  );
  await activate(
    page,
    dialog.getByRole("button", { name: "Report a bug", exact: true }),
  );
  await dialog
    .getByLabel("Title", { exact: true })
    .fill("Canvas feedback example");
  await dialog
    .getByLabel("Details", { exact: true })
    .fill("A visual issue while testing the canvas.");
  await activate(
    page,
    dialog.getByRole("button", { name: "Capture current view", exact: true }),
  );
  await expect(dialog.getByAltText("Screenshot preview")).toBeVisible();
  await activate(
    page,
    dialog.getByRole("button", { name: "Send report", exact: true }),
  );
  await expect(
    dialog.getByRole("heading", {
      name: "Canvas feedback example",
      exact: true,
    }),
  ).toBeVisible();
  await expect(dialog.getByAltText("Report screenshot")).toBeVisible();
  await dialog
    .getByLabel("Add a comment", { exact: true })
    .fill("Additional steps for reproduction.");
  await activate(
    page,
    dialog.getByRole("button", { name: "Send comment", exact: true }),
  );
  await expect(
    dialog.getByText("Additional steps for reproduction.", { exact: true }),
  ).toBeVisible();
  await activate(
    page,
    dialog.getByRole("button", { name: "Close dialog", exact: true }),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Administration", exact: true }),
  );
  await activate(
    page,
    dialog.getByRole("button", { name: "Feedback & bugs", exact: true }),
  );
  await activate(
    page,
    dialog.getByRole("button", { name: /Canvas feedback example/ }),
  );
  await activate(
    page,
    dialog.getByRole("button", { name: "Accept", exact: true }),
  );
  await expect(dialog.getByText(/Bug · accepted/)).toBeVisible();
  await dialog.getByLabel("Fixed in release", { exact: true }).fill("v1.2.0");
  await activate(
    page,
    dialog.getByRole("button", { name: "Close with release", exact: true }),
  );
  await expect(
    dialog.getByRole("link", { name: "Fixed in v1.2.0", exact: true }),
  ).toHaveAttribute(
    "href",
    "https://github.com/Neokil/mindflow/releases/tag/v1.2.0",
  );
});

test("product-update consent is off by default and persists in profile", async ({
  page,
}) => {
  await login(page);
  await activate(
    page,
    page.getByRole("button", { name: "Your profile", exact: true }),
  );
  const checkbox = page.getByRole("checkbox", {
    name: "Email me when a new mindflow version is available.",
    exact: true,
  });
  await expect(checkbox).not.toBeChecked();
  await activate(page, checkbox);
  await expect(checkbox).toBeChecked();
  await expect(checkbox).toBeEnabled();
  await activate(
    page,
    page.getByRole("button", { name: "Close dialog", exact: true }),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Your profile", exact: true }),
  );
  await expect(checkbox).toBeChecked();
  await activate(page, checkbox);
  await expect(checkbox).not.toBeChecked();
});
