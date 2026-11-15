import { test, expect } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { login, activate } from "./helpers";
import { pdfFixture } from "./pdf-fixture";

test("bulk import keeps successful notebooks and PDFs when another file fails", async ({
  page,
}) => {
  await login(page);
  const initial = await (await page.request.get("/api/v1/workspaces")).json();
  await activate(
    page,
    page.getByRole("button", { name: "Import workspace", exact: true }),
  );
  await page.getByLabel("Workspace export", { exact: true }).setInputFiles([
    {
      name: "Migration.bfly",
      mimeType: "application/zip",
      buffer: await readFile(
        "../internal/workspace/butterfly/testdata/two-pages.bfly",
      ),
    },
    {
      name: "Broken.bfly",
      mimeType: "application/zip",
      buffer: Buffer.from("not a ZIP"),
    },
    { name: "Notes.pdf", mimeType: "application/pdf", buffer: pdfFixture() },
  ]);
  await activate(
    page,
    page.getByRole("button", { name: "Import", exact: true }),
  );
  const rows = page.locator(".import-results > li");
  await expect(
    rows.filter({ hasText: "Migration.bfly" }).getByRole("status"),
  ).toHaveText("Imported 2 pages and 6 elements into a private workspace.");
  await expect(
    rows.filter({ hasText: "Broken.bfly" }).getByRole("alert"),
  ).toBeVisible();
  const pdf = rows.filter({ hasText: "Notes.pdf" });
  await expect(pdf.getByRole("status")).toHaveText(
    "Imported 2 pages and 2 elements into a private workspace.",
  );
  await expect(
    page.getByRole("heading", { name: "Optimise imported images?" }),
  ).toBeVisible();
  await activate(
    page,
    page.getByRole("button", { name: "Optimise images", exact: true }),
  );
  await expect(
    page.getByRole("heading", { name: "Optimise imported images?" }),
  ).toHaveCount(0);
  const result = await (await page.request.get("/api/v1/workspaces")).json();
  expect(result).toHaveLength(initial.length + 2);
  await activate(
    page,
    page.getByRole("button", { name: "Import remaining files", exact: true }),
  );
  await expect(
    page.getByRole("button", { name: "Import remaining files", exact: true }),
  ).toBeEnabled();
  expect(
    await (await page.request.get("/api/v1/workspaces")).json(),
  ).toHaveLength(result.length);
  await activate(
    page,
    pdf.getByRole("button", { name: "Open workspace", exact: true }),
  );
  await expect(page.getByRole("tab")).toHaveCount(2);
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
});
