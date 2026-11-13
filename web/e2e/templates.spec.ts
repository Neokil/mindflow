import { test, expect } from "@playwright/test";
import { create, login } from "./helpers";
import type { CanvasDocument } from "../src/model";

test("saves a locked image and text as a workspace template and inserts independent editable copies", async ({
  page,
}) => {
  await login(page);
  await create(page, `Templates ${Date.now()}`);
  const box = (await page.getByTestId("canvas").boundingBox())!;
  const image = await page.evaluate(() => {
    const canvas = document.createElement("canvas");
    canvas.width = 400;
    canvas.height = 100;
    const context = canvas.getContext("2d")!;
    context.fillStyle = "#2658ad";
    context.fillRect(0, 0, 400, 100);
    return canvas.toDataURL("image/png").split(",")[1];
  });
  await page.getByRole("button", { name: "Add images", exact: true }).click();
  await page.locator('input[type="file"]').setInputFiles({
    name: "banner.png",
    mimeType: "image/png",
    buffer: Buffer.from(image, "base64"),
  });
  await expect(
    page.getByText("Drag to size image", { exact: false }),
  ).toBeVisible();
  await page.mouse.click(box.x + 600, box.y + 300);
  await page.getByRole("button", { name: "Text (T)", exact: true }).click();
  await page.mouse.click(box.x + 600, box.y + 300);
  await page.getByLabel("Edit canvas text").fill("Reusable heading");
  await page.getByLabel("Edit canvas text").press("Control+Enter");
  await page.keyboard.press("Control+a");
  await page.keyboard.press("Control+g");
  await page.getByRole("button", { name: "Lock groups", exact: true }).click();
  await page
    .getByRole("button", { name: "Save as template", exact: true })
    .click();
  await page.getByLabel("Template name", { exact: true }).fill("Heading");
  await page
    .getByRole("button", { name: "Save template", exact: true })
    .click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  const firstPage = page.url().split("/pages/")[1];
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const original = (
    await (await page.request.get(`/api/v1/pages/${firstPage}/document`)).json()
  ).document as CanvasDocument;
  await page.getByRole("button", { name: "Add page", exact: true }).click();
  await page.getByLabel("Name", { exact: true }).fill("Other page");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByTestId("element-count")).toHaveText("0 elements");
  await page
    .getByRole("button", { name: "Workspace templates", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Insert Heading", exact: true })
    .click();
  await expect(
    page.getByText("Click to place template.", { exact: false }),
  ).toBeVisible();
  await expect(page.getByTestId("element-count")).toHaveText("0 elements");
  await page.mouse.click(box.x + 600, box.y + 300);
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const secondPage = page.url().split("/pages/")[1];
  const inserted = (
    await (
      await page.request.get(`/api/v1/pages/${secondPage}/document`)
    ).json()
  ).document as CanvasDocument;
  expect(
    inserted.elements.every(
      (element) =>
        !element.groupId &&
        !element.locked &&
        !original.elements.some((source) => source.id === element.id),
    ),
  ).toBe(true);
  expect(inserted.elements[0].assetId).toBe(original.elements[0].assetId);
  await page.mouse.click(box.x + 1000, box.y + 500);
  await page.mouse.dblclick(box.x + 600, box.y + 300);
  await page.getByLabel("Edit canvas text").fill("Edited copy");
  await page.getByLabel("Edit canvas text").press("Control+Enter");
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const unchanged = (
    await (await page.request.get(`/api/v1/pages/${firstPage}/document`)).json()
  ).document as CanvasDocument;
  expect(
    unchanged.elements.find((element) => element.type === "text")?.text,
  ).toBe("Reusable heading");
  await page
    .getByRole("button", { name: "Workspace templates", exact: true })
    .click();
  page.once("dialog", (dialog) => dialog.accept());
  await page
    .getByRole("button", { name: "Delete template Heading", exact: true })
    .click();
  await expect(
    page.getByText("No templates yet.", { exact: false }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Close dialog", exact: true }).click();
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  await page.reload();
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  const saved = (
    await (
      await page.request.get(`/api/v1/pages/${secondPage}/document`)
    ).json()
  ).document as CanvasDocument;
  expect(saved.elements.find((element) => element.type === "text")?.text).toBe(
    "Edited copy",
  );
  expect(
    (
      await page.request.get(`/api/v1/assets/${saved.elements[0].assetId}`)
    ).ok(),
  ).toBe(true);
});
