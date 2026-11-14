import { test, expect } from "@playwright/test";
import { activate, login, create } from "./helpers";
import { invite } from "./accounts";

test("reports use nested dialogs and enforce the review lifecycle", async ({
  page,
}, info) => {
  await login(page);
  await create(page, "Feedback example");
  await activate(
    page,
    page.getByRole("button", { name: "Feedback & Bugreport", exact: true }),
  );
  const list = page.getByRole("dialog", {
    name: "Feedback & Bugreport",
    exact: true,
  });
  await expect(list.getByRole("combobox")).toHaveCount(0);
  await activate(
    page,
    list.getByRole("button", { name: "New report", exact: true }),
  );
  const form = page.getByRole("dialog", { name: "New report", exact: true });
  await activate(
    page,
    form.getByRole("button", { name: "Report a bug", exact: true }),
  );
  await form
    .getByLabel("Title", { exact: true })
    .fill("Canvas feedback example");
  await form
    .getByLabel("Details", { exact: true })
    .fill("A visual issue while testing the canvas.");
  await activate(
    page,
    form.getByRole("button", { name: "Capture current view", exact: true }),
  );
  await expect(form.getByAltText("Screenshot preview")).toBeVisible();
  await activate(
    page,
    form.getByRole("button", { name: "Send report", exact: true }),
  );
  const ticket = page.getByRole("dialog", {
    name: "Canvas feedback example",
    exact: true,
  });
  await expect(
    ticket.getByRole("heading", {
      name: "Canvas feedback example",
      exact: true,
    }),
  ).toBeVisible();
  await expect(ticket.getByAltText("Report screenshot")).toBeVisible();
  await ticket
    .getByLabel("Add a comment", { exact: true })
    .fill("Additional steps for reproduction.");
  await activate(
    page,
    ticket.getByRole("button", { name: "Send comment", exact: true }),
  );
  await expect(
    ticket.getByText("Additional steps for reproduction.", { exact: true }),
  ).toBeVisible();
  await activate(
    page,
    ticket.getByRole("button", { name: "Back to reports", exact: true }),
  );
  await expect(list).toBeVisible();
  await expect(
    list.getByRole("button", { name: /Canvas feedback example/ }),
  ).toBeVisible();
  await activate(
    page,
    list.getByRole("button", { name: "Close dialog", exact: true }),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Administration", exact: true }),
  );
  const admin = page.getByRole("dialog", {
    name: "Administration",
    exact: true,
  });
  await activate(page, admin.getByRole("button", { name: /^Feedback & bugs/ }));
  await activate(
    page,
    admin.getByRole("button", { name: /Canvas feedback example/ }),
  );
  await expect(
    ticket.getByRole("button", { name: "Accept", exact: true }),
  ).toBeVisible();
  await expect(
    ticket.getByRole("button", { name: "Deny", exact: true }),
  ).toBeDisabled();
  await expect(
    ticket.getByRole("button", { name: "Close with release", exact: true }),
  ).toHaveCount(0);
  await activate(
    page,
    ticket.getByRole("button", { name: "Accept", exact: true }),
  );
  await expect(ticket.getByText(/Bug · Accepted, to do/)).toBeVisible();
  await expect(
    ticket.getByRole("button", { name: "Accept", exact: true }),
  ).toHaveCount(0);
  await expect(
    ticket.getByRole("button", { name: "Deny", exact: true }),
  ).toHaveCount(0);
  await expect(
    ticket.getByLabel("Fixed in release", { exact: true }),
  ).toHaveCount(0);
  await ticket
    .getByLabel("Add a comment", { exact: true })
    .fill("Fixed in v1.2.0");
  await page.screenshot({
    path: `../.cache/e2e/feedback-review-${info.project.name}.png`,
  });
  await activate(
    page,
    ticket.getByRole("button", { name: "Close with release", exact: true }),
  );
  await expect(
    ticket.getByRole("link", { name: "Fixed in v1.2.0", exact: true }),
  ).toHaveAttribute(
    "href",
    "https://github.com/Neokil/mindflow/releases/tag/v1.2.0",
  );
  await expect(
    ticket.getByRole("button", { name: "Close with release", exact: true }),
  ).toHaveCount(0);
  await expect(
    ticket.getByRole("button", { name: "Reopen", exact: true }),
  ).toBeVisible();
  await activate(
    page,
    ticket.getByRole("button", { name: "Reopen", exact: true }),
  );
  await expect(ticket.getByText(/Bug · Created/)).toBeVisible();
  await ticket
    .getByLabel("Add a comment", { exact: true })
    .fill("Closing because the issue is no longer reproducible.");
  await activate(
    page,
    ticket.getByRole("button", { name: "Deny", exact: true }),
  );
  await expect(ticket.getByText(/Bug · Closed/)).toBeVisible();
  await expect(
    ticket.getByText("Closing because the issue is no longer reproducible.", {
      exact: true,
    }),
  ).toBeVisible();
  await activate(
    page,
    ticket.getByRole("button", { name: "Back to reports", exact: true }),
  );
  await expect(admin).toBeVisible();
});

test("profile avoids an unsolicited close tooltip and keeps update consent inline", async ({
  page,
}) => {
  await login(page);
  const controls = page.locator(".header-actions > button");
  await expect(controls.nth(-2)).toHaveAttribute(
    "aria-label",
    "Feedback & Bugreport",
  );
  await expect(controls.last()).toHaveAttribute("aria-label", "Sign out");
  await activate(
    page,
    page.getByRole("button", { name: "Your profile", exact: true }),
  );
  const dialog = page.getByRole("dialog", {
    name: "Your profile",
    exact: true,
  });
  await expect(
    dialog.getByRole("heading", { name: "Your profile", exact: true }),
  ).toBeFocused();
  await expect(page.getByRole("tooltip")).toHaveCount(0);
  await expect(dialog.getByText(/Off by default/)).toHaveCount(0);
  const checkbox = dialog.getByRole("checkbox", {
    name: "Email me when a new mindflow version is available.",
    exact: true,
  });
  await expect(checkbox).not.toBeChecked();
  await expect(checkbox.locator("..")).toHaveCSS("flex-direction", "row");
  await activate(page, checkbox);
  await expect(checkbox).toBeChecked();
  await expect(checkbox).toBeEnabled();
  await activate(
    page,
    dialog.getByRole("button", { name: "Close dialog", exact: true }),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Your profile", exact: true }),
  );
  await expect(checkbox).toBeChecked();
  await activate(page, checkbox);
  await expect(checkbox).not.toBeChecked();
});

test("report activity badges clear independently for administrators and authors", async ({
  page,
  browser,
}) => {
  test.setTimeout(90000);
  await login(page);
  const email = `feedback-${Date.now()}@test.invalid`,
    password = "feedback test passphrase";
  const token = await invite(page, email);
  expect(
    (
      await page.request.post("/api/v1/auth/accept-invite", {
        data: { token, password, displayName: "Feedback Tester" },
      })
    ).ok(),
  ).toBe(true);
  const context = await browser.newContext({
    viewport: { width: 1100, height: 850 },
  });
  const guest = await context.newPage();
  try {
    await guest.goto("http://127.0.0.1:8087/");
    await guest.getByLabel("Email address", { exact: true }).fill(email);
    await guest.getByLabel("Password", { exact: true }).fill(password);
    await activate(
      guest,
      guest.getByRole("button", { name: "Sign in", exact: true }),
    );
    await expect(
      guest.getByRole("heading", { name: "Workspaces", exact: true }),
    ).toBeVisible();
    const me = await (await guest.request.get("/api/v1/auth/me")).json();
    const response = await guest.request.post("/api/v1/feedback", {
      headers: { "X-CSRF-Token": me.csrf },
      data: {
        kind: "feature",
        title: "Unread example",
        body: "A useful improvement.",
      },
    });
    expect(response.ok()).toBe(true);
    await page.bringToFront();
    await page.evaluate(() => window.dispatchEvent(new Event("focus")));
    await expect(page.getByTestId("admin-feedback-unread")).toHaveText("1");
    await activate(
      page,
      page.getByRole("button", { name: "Administration", exact: true }),
    );
    const admin = page.getByRole("dialog", {
      name: "Administration",
      exact: true,
    });
    await activate(
      page,
      admin.getByRole("button", { name: /^Feedback & bugs/ }),
    );
    await activate(page, admin.getByRole("button", { name: /Unread example/ }));
    const ticket = page.getByRole("dialog", {
      name: "Unread example",
      exact: true,
    });
    await expect(page.getByTestId("admin-feedback-unread")).toHaveCount(0);
    await ticket
      .getByLabel("Add a comment", { exact: true })
      .fill("Could you add an example?");
    await activate(
      page,
      ticket.getByRole("button", { name: "Send comment", exact: true }),
    );
    await expect(
      ticket.getByText("Could you add an example?", { exact: true }),
    ).toBeVisible();
    await guest.bringToFront();
    await guest.evaluate(() => window.dispatchEvent(new Event("focus")));
    await expect(guest.getByTestId("feedback-unread")).toHaveText("1");
    await activate(
      guest,
      guest.getByRole("button", { name: "Feedback & Bugreport", exact: true }),
    );
    const list = guest.getByRole("dialog", {
      name: "Feedback & Bugreport",
      exact: true,
    });
    await activate(guest, list.getByRole("button", { name: /Unread example/ }));
    await expect(
      guest
        .getByRole("dialog", { name: "Unread example", exact: true })
        .getByText("Could you add an example?", { exact: true }),
    ).toBeVisible();
    await expect(guest.getByTestId("feedback-unread")).toHaveCount(0);
  } finally {
    await context.close();
  }
});
