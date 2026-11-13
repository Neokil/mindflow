import { expect, test } from "@playwright/test";
import { login, create } from "./helpers";
import type { CanvasDocument } from "../src/model";
import { pdfFixture } from "./pdf-fixture";
test("imports PDF images onto this canvas, separate pages, and a retained viewer", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (err) => errors.push(err.message));
  await login(page);
  await create(page, `PDF tools ${Date.now()}`);
  const id = page.url().split("/pages/")[1];
  const choose = async (mode: string) => {
    await page.getByRole("button", { name: "Import PDF", exact: true }).click();
    await page.getByLabel("PDF file", { exact: true }).setInputFiles({
      name: "example.pdf",
      mimeType: "application/pdf",
      buffer: pdfFixture(),
    });
    await page.getByLabel("PDF placement").selectOption(mode);
    await page
      .getByRole("dialog", { name: "Import PDF", exact: true })
      .getByRole("button", { name: "Import PDF", exact: true })
      .click();
    await expect(
      page.getByRole("dialog", { name: "Import PDF", exact: true }),
    ).toHaveCount(0, { timeout: 30000 });
  };
  await choose("current");
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  let doc = (
    await (await page.request.get(`/api/v1/pages/${id}/document`)).json()
  ).document as CanvasDocument;
  expect(doc.elements.every((e) => e.type === "image")).toBe(true);
  await choose("viewer");
  await expect(page.getByTestId("element-count")).toHaveText("3 elements");
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  doc = (await (await page.request.get(`/api/v1/pages/${id}/document`)).json())
    .document as CanvasDocument;
  const pdf = doc.elements[2];
  expect(pdf.type).toBe("pdf");
  const raw = await page.request.get(`/api/v1/assets/${pdf.assetId}`);
  expect(raw.headers()["content-type"]).toBe("application/pdf");
  expect(raw.headers()["content-disposition"]).toContain("attachment");
  await page
    .getByRole("button", { name: "Selection properties", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Open PDF viewer", exact: true })
    .click();
  await expect(
    page.getByRole("img", { name: "PDF page 1", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Next page", exact: true }).click();
  await expect(
    page.getByRole("img", { name: "PDF page 2", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Close dialog", exact: true }).click();
  await page
    .getByRole("button", { name: "Close settings", exact: true })
    .click();
  await choose("pages");
  await expect(page.getByRole("tab", { name: /example.*1/ })).toBeVisible();
  await expect(page.getByRole("tab", { name: /example.*2/ })).toBeVisible();
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  expect(errors).toEqual([]);
});
