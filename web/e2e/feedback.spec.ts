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
  await form.getByLabel("Report type", { exact: true }).selectOption("bug");
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
  await expect(form.getByAltText("Image preview")).toBeVisible();
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
  await expect(ticket.getByAltText("Report image")).toBeVisible();
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
  const search = admin.getByRole("searchbox", {
    name: "Search tickets",
    exact: true,
  });
  for (const query of ["CANVAS", "visual issue", "no longer reproducible"]) {
    await search.fill(query);
    await expect(
      admin.getByRole("button", { name: /Canvas feedback example/ }),
    ).toBeVisible();
  }
  await search.fill("not present in any ticket");
  await expect(
    admin.getByText("No reports match your search.", { exact: true }),
  ).toBeVisible();
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

test("new report is one form with aligned image controls and file-picker cancellation preserves the draft", async ({
  page,
}) => {
  await login(page);
  await activate(
    page,
    page.getByRole("button", { name: "Feedback & Bugreport", exact: true }),
  );
  const list = page.getByRole("dialog", {
    name: "Feedback & Bugreport",
    exact: true,
  });
  await activate(
    page,
    list.getByRole("button", { name: "New report", exact: true }),
  );
  const form = page.getByRole("dialog", { name: "New report", exact: true });
  await expect(form.getByLabel("Report type", { exact: true })).toHaveValue(
    "bug",
  );
  await form
    .getByLabel("Title", { exact: true })
    .fill("Image attachment example");
  await form
    .getByLabel("Details", { exact: true })
    .fill("Keep this draft when cancelling the image picker.");
  const capture = form.getByRole("button", {
    name: "Capture current view",
    exact: true,
  });
  const upload = form.getByRole("button", {
    name: "Upload image",
    exact: true,
  });
  const a = (await capture.boundingBox())!,
    b = (await upload.boundingBox())!;
  expect(Math.abs(a.y + a.height / 2 - b.y - b.height / 2)).toBeLessThan(1);
  expect(Math.abs(a.height - b.height)).toBeLessThan(1);
  const input = form.getByLabel("Upload feedback image", { exact: true });
  // Dispatch in the page realm, matching the native input cancel event.
  await input.evaluate((element) =>
    element.dispatchEvent(new Event("cancel", { bubbles: true })),
  );
  await expect(form).toBeVisible();
  await expect(form.getByLabel("Title", { exact: true })).toHaveValue(
    "Image attachment example",
  );
  for (const [type, extension] of [
    ["image/jpeg", "jpg"],
    ["image/webp", "webp"],
  ]) {
    const encoded = await page.evaluate((type) => {
      const canvas = document.createElement("canvas");
      canvas.width = 32;
      canvas.height = 24;
      const context = canvas.getContext("2d")!;
      context.fillStyle = "#1b5e3b";
      context.fillRect(0, 0, 32, 24);
      return canvas.toDataURL(type).split(",")[1];
    }, type);
    const image = {
      name: `example.${extension}`,
      mimeType: type,
      buffer: Buffer.from(encoded, "base64"),
    };
    if (extension === "jpg") {
      const chooser = page.waitForEvent("filechooser");
      await activate(page, upload);
      await (await chooser).setFiles(image);
    } else await input.setInputFiles(image);
    await expect(form.getByAltText("Image preview")).toBeVisible();
    await expect(form.getByAltText("Image preview")).toHaveAttribute(
      "src",
      /^data:image\/png;base64,/,
    );
    await expect(
      form.getByRole("button", { name: "Send report", exact: true }),
    ).toBeEnabled();
  }
  await form.getByLabel("Report type", { exact: true }).selectOption("feature");
  await expect(form.getByLabel("Title", { exact: true })).toHaveValue(
    "Image attachment example",
  );
  await activate(
    page,
    form.getByRole("button", { name: "Send report", exact: true }),
  );
  const ticket = page.getByRole("dialog", {
    name: "Image attachment example",
    exact: true,
  });
  await expect(ticket.getByText(/Feature request · Created/)).toBeVisible();
  await expect(ticket.getByAltText("Report image")).toBeVisible();
});
