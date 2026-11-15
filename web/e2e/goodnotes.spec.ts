import { test, expect } from "@playwright/test";
import { resolve } from "node:path";
import { readFile } from "node:fs/promises";
import { unzipSync } from "fflate";
import { activate, login } from "./helpers";

test("native Goodnotes import and export retain separate editable elements", async ({
  page,
}) => {
  await login(page);
  await activate(
    page,
    page.getByRole("button", { name: "Import workspace", exact: true }),
  );
  await page
    .getByLabel("Workspace export", { exact: true })
    .setInputFiles(
      resolve("../docs/examples/mindflow-compatibility.goodnotes"),
    );
  await activate(
    page,
    page.getByRole("button", { name: "Import", exact: true }),
  );
  await expect(
    page.getByText(
      "Imported 1 pages and 4 elements into a private workspace.",
      { exact: true },
    ),
  ).toBeVisible();
  await activate(
    page,
    page.getByRole("button", { name: "Open workspace", exact: true }),
  );
  await expect(page.getByTestId("element-count")).toHaveText("4 elements");
  await activate(
    page,
    page.getByRole("button", { name: "Export workspace", exact: true }),
  );
  await page.getByLabel("Export format").selectOption("goodnotes");
  const downloading = page.waitForEvent("download");
  await activate(
    page,
    page.getByRole("button", { name: "Export", exact: true }),
  );
  const file = await downloading;
  expect(file.suggestedFilename()).toMatch(/\.goodnotes$/);
  const raw = await readFile((await file.path())!);
  const files = unzipSync(raw);
  expect(files["document.info.pb"]).toBeDefined();
  expect(files["index.notes.pb"]).toBeDefined();
  await expect(
    page.getByText("Goodnotes native export is experimental.", {
      exact: false,
    }),
  ).toBeVisible();
  await activate(
    page,
    page.getByRole("button", { name: "Close", exact: true }),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Import pages", exact: true }),
  );
  await page.getByLabel("Workspace export", { exact: true }).setInputFiles({
    name: "roundtrip.goodnotes",
    mimeType: "application/zip",
    buffer: raw,
  });
  await activate(
    page,
    page.getByRole("button", { name: "Import", exact: true }),
  );
  // Export introduces its own blank PDF background while preserving source elements.
  await expect(
    page.getByText("Imported 1 pages and 5 elements into this workspace.", {
      exact: true,
    }),
  ).toBeVisible();
});

test("operator's private Goodnotes migration fixture", async ({ page }) => {
  test.skip(
    !process.env.MINDFLOW_GOODNOTES_FIXTURE,
    "Private sample is not stored in the repository",
  );
  test.setTimeout(180000);
  await login(page);
  await activate(
    page,
    page.getByRole("button", { name: "Import workspace", exact: true }),
  );
  await page
    .getByLabel("Workspace export", { exact: true })
    .setInputFiles(process.env.MINDFLOW_GOODNOTES_FIXTURE!);
  await activate(
    page,
    page.getByRole("button", { name: "Import", exact: true }),
  );
  await expect(
    page.getByRole("button", { name: "Open workspace", exact: true }),
  ).toBeVisible({ timeout: 160000 });
});
