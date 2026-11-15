import { test, expect } from "@playwright/test";
import { resolve } from "node:path";
import { readFile } from "node:fs/promises";
import { activate, login } from "./helpers";

test("editable native OneNote notebooks export and import through the UI", async ({
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
      resolve("../internal/workspace/butterfly/testdata/two-pages.bfly"),
    );
  await activate(
    page,
    page.getByRole("button", { name: "Import", exact: true }),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Open workspace", exact: true }),
  );
  await expect(page.getByTestId("element-count")).toHaveText("3 elements");
  await activate(
    page,
    page.getByRole("button", { name: "Export workspace", exact: true }),
  );
  await page.getByLabel("Export format").selectOption("onepkg");
  const downloading = page.waitForEvent("download");
  await activate(
    page,
    page.getByRole("button", { name: "Export", exact: true }),
  );
  const file = await downloading;
  expect(file.suggestedFilename()).toMatch(/\.onepkg$/);
  const raw = await readFile((await file.path())!);
  expect(raw.subarray(0, 4).toString()).toBe("MSCF");
  await expect(
    page.getByText("Ink remains editable", { exact: false }),
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
    name: "roundtrip.onepkg",
    mimeType: "application/octet-stream",
    buffer: raw,
  });
  await activate(
    page,
    page.getByRole("button", { name: "Import", exact: true }),
  );
  await expect(
    page.getByText("Imported 2 pages and 6 elements into this workspace.", {
      exact: true,
    }),
  ).toBeVisible();
  await activate(
    page,
    page.getByRole("button", { name: "Open imported page", exact: true }),
  );
  await expect(page.getByRole("tab")).toHaveCount(4);
  await expect(page.getByTestId("element-count")).toHaveText("3 elements");
});
