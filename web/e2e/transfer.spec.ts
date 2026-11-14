import { test, expect, type Page } from "@playwright/test";
import { resolve } from "node:path";
import { readFile } from "node:fs/promises";
import { unzipSync } from "fflate";
import { PDFDocument } from "pdf-lib";
import { activate, login } from "./helpers";
async function example(page: Page, name: string) {
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
  await page.getByLabel("Imported workspace name", { exact: true }).fill(name);
  await activate(
    page,
    page.getByRole("button", { name: "Import", exact: true }),
  );
  await expect(
    page.getByText(
      "Imported 2 pages and 6 elements into a private workspace.",
      { exact: true },
    ),
  ).toBeVisible();
  await activate(
    page,
    page.getByRole("button", { name: "Open workspace", exact: true }),
  );
  await expect(page.getByTestId("element-count")).toHaveText("3 elements");
}
async function download(page: Page) {
  const pending = page.waitForEvent("download");
  await activate(
    page,
    page.getByRole("button", { name: "Export", exact: true }),
  );
  const file = await pending,
    path = await file.path();
  await expect(page.getByText("Export ready.", { exact: false })).toBeVisible();
  return { name: file.suggestedFilename(), raw: await readFile(path!) };
}
test("native workspace and selected pages round-trip through import", async ({
  page,
}, info) => {
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await example(page, `Transfer ${info.project.name}`);
  await activate(
    page,
    page.getByRole("button", { name: "Export workspace", exact: true }),
  );
  const file = await download(page);
  const manifest = JSON.parse(
    new TextDecoder().decode(unzipSync(file.raw)["manifest.json"]),
  );
  expect(file.name).toMatch(/\.mindflow$/);
  expect(manifest.pages).toHaveLength(2);
  expect(manifest.assets).toHaveLength(1);
  await activate(
    page,
    page.getByRole("button", { name: "Close", exact: true }),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Import pages", exact: true }),
  );
  await page.getByLabel("Workspace export", { exact: true }).setInputFiles({
    name: file.name,
    mimeType: "application/zip",
    buffer: file.raw,
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
  await activate(
    page,
    page
      .getByRole("button", { name: "Page options for Page 2", exact: true })
      .last(),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Export page", exact: true }),
  );
  const selected = await download(page),
    one = JSON.parse(
      new TextDecoder().decode(unzipSync(selected.raw)["manifest.json"]),
    );
  expect(one.kind).toBe("pages");
  expect(one.pages).toHaveLength(1);
  expect(one.assets).toHaveLength(0);
  await activate(
    page,
    page.getByRole("button", { name: "Close", exact: true }),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Back to workspaces", exact: true }),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Import workspace", exact: true }),
  );
  await page.getByLabel("Workspace export", { exact: true }).setInputFiles({
    name: selected.name,
    mimeType: "application/zip",
    buffer: selected.raw,
  });
  await page
    .getByLabel("Imported workspace name", { exact: true })
    .fill(`Restored ${info.project.name}`);
  await activate(
    page,
    page.getByRole("button", { name: "Import", exact: true }),
  );
  await expect(
    page.getByText(
      "Imported 1 pages and 3 elements into a private workspace.",
      { exact: true },
    ),
  ).toBeVisible();
  await activate(
    page,
    page.getByRole("button", { name: "Open workspace", exact: true }),
  );
  await expect(page.getByRole("tab")).toHaveCount(1);
  await expect(page.getByTestId("element-count")).toHaveText("3 elements");
  expect(errors).toEqual([]);
});
test("PDF and PNG outputs contain each rendered canvas page", async ({
  page,
}, info) => {
  test.setTimeout(90000);
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await example(page, `Visual ${info.project.name}`);
  await activate(
    page,
    page.getByRole("button", { name: "Export workspace", exact: true }),
  );
  await page.getByLabel("Export format", { exact: true }).selectOption("pdf");
  await page
    .getByLabel("Export resolution", { exact: true })
    .selectOption("1024");
  await page.screenshot({
    path: `../.cache/e2e/export-dialog-${info.project.name}.png`,
  });
  const pdf = await download(page);
  expect(pdf.name).toMatch(/\.pdf$/);
  expect((await PDFDocument.load(pdf.raw)).getPageCount()).toBe(2);
  await page.getByLabel("Export format", { exact: true }).selectOption("png");
  const zip = await download(page),
    images = unzipSync(zip.raw);
  expect(zip.name).toMatch(/-images.zip$/);
  expect(Object.keys(images)).toEqual(["001-Page 1.png", "002-Page 2.png"]);
  for (const raw of Object.values(images)) {
    expect(Array.from(raw.slice(0, 8))).toEqual([
      137, 80, 78, 71, 13, 10, 26, 10,
    ]);
    const view = new DataView(raw.buffer, raw.byteOffset, raw.byteLength);
    expect(Math.max(view.getUint32(16), view.getUint32(20))).toBe(1024);
  }
  const red = await page.evaluate(async (bytes) => {
    const url = URL.createObjectURL(
      new Blob([new Uint8Array(bytes)], { type: "image/png" }),
    );
    try {
      const image = new Image();
      image.src = url;
      await image.decode();
      const canvas = document.createElement("canvas");
      canvas.width = image.width;
      canvas.height = image.height;
      const ctx = canvas.getContext("2d")!;
      ctx.drawImage(image, 0, 0);
      const pixels = ctx.getImageData(0, 0, canvas.width, canvas.height).data;
      let red = 0;
      for (let i = 0; i < pixels.length; i += 4)
        if (pixels[i] > 150 && pixels[i + 1] < 130 && pixels[i + 2] < 130)
          red++;
      return red;
    } finally {
      URL.revokeObjectURL(url);
    }
  }, Array.from(images["001-Page 1.png"]));
  expect(red).toBeGreaterThan(10);
  await page
    .getByLabel("Export pages", { exact: true })
    .selectOption("selected");
  await activate(
    page,
    page.getByRole("checkbox", { name: "Page 2", exact: true }),
  );
  await expect(
    page.getByRole("checkbox", { name: "Page 2", exact: true }),
  ).toBeChecked();
  // WebKit on Linux sanitizes spaces in the suggested download name.
  expect((await download(page)).name).toMatch(/^Page[ _]2\.png$/);
  expect(errors).toEqual([]);
});
