import { test, expect } from "@playwright/test";
import { resolve } from "node:path";
import { login } from "./helpers";
import type { CanvasDocument } from "../src/model";

test("imports Butterfly pages, embedded image, pen, shapes and white text", async ({
  page,
}, info) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await login(page);
  await page
    .getByRole("button", { name: "Import workspace", exact: true })
    .click();
  await page
    .getByLabel("Workspace export", { exact: true })
    .setInputFiles(
      process.env.MINDFLOW_IMPORT_SAMPLE ||
        resolve("../internal/workspace/butterfly/testdata/two-pages.bfly"),
    );
  const name = `Butterfly ${info.project.name} ${Date.now()}`;
  await page.getByLabel("Imported workspace name", { exact: true }).fill(name);
  await page.getByRole("button", { name: "Import", exact: true }).click();
  await expect(
    page.getByText(
      "Imported 2 pages and 6 elements into a private workspace.",
      { exact: true },
    ),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Conversion notes", exact: true }),
  ).toBeVisible();
  const imageLoaded = page.waitForResponse(
    (response) =>
      response.url().includes("/api/v1/assets/") &&
      response.request().resourceType() === "image",
  );
  await page
    .getByRole("button", { name: "Open workspace", exact: true })
    .click();
  await expect(
    page.getByRole("tab", { name: "Page 1", exact: true }),
  ).toHaveAttribute("aria-selected", "true");
  await expect(page.getByTestId("element-count")).toHaveText("3 elements");
  await expect(page.getByTestId("canvas")).toHaveCSS(
    "background-color",
    "rgb(0, 0, 0)",
  );
  await (await imageLoaded).finished();
  await page.evaluate(
    () =>
      new Promise<void>((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
      ),
  );
  const workspaces = await (
    await page.request.get("/api/v1/workspaces")
  ).json();
  const workspace = workspaces.find(
    (entry: { name: string }) => entry.name === name,
  );
  const pages = await (
    await page.request.get(`/api/v1/workspaces/${workspace.id}/pages`)
  ).json();
  expect(pages.map((entry: { name: string }) => entry.name)).toEqual([
    "Page 1",
    "Page 2",
  ]);
  const first = (await (
    await page.request.get(`/api/v1/pages/${pages[0].id}/document`)
  ).json()) as { document: CanvasDocument };
  expect(
    first.document.elements.some(
      (entry) => entry.type === "stroke" && entry.color === "#ff5722",
    ),
  ).toBe(true);
  const image = first.document.elements.find(
    (entry) => entry.type === "image",
  )!;
  expect(
    (await page.request.get(`/api/v1/assets/${image.assetId}`)).headers()[
      "content-type"
    ],
  ).toBe("image/png");
  await page.screenshot({
    path: `../.cache/e2e/butterfly-page1-${info.project.name}.png`,
  });
  await page.getByRole("tab", { name: "Page 2", exact: true }).click();
  await expect(page.getByTestId("element-count")).toHaveText("3 elements");
  await expect(page.getByTestId("canvas")).toHaveCSS(
    "background-color",
    "rgb(0, 0, 0)",
  );
  const second = (await (
    await page.request.get(`/api/v1/pages/${pages[1].id}/document`)
  ).json()) as { document: CanvasDocument };
  expect(
    second.document.elements.some(
      (entry) =>
        entry.type === "text" &&
        entry.color === "#ffffff" &&
        entry.text === "Text on second page",
    ),
  ).toBe(true);
  expect(
    second.document.elements.some(
      (entry) => entry.shape === "circle" && entry.border === "#ffffff",
    ),
  ).toBe(true);
  expect(
    second.document.elements.some(
      (entry) => entry.shape === "rectangle" && entry.border === "#f44336",
    ),
  ).toBe(true);
  await expect
    .poll(() =>
      page
        .locator('[data-testid="canvas"] canvas')
        .first()
        .evaluate((canvas) => {
          const context = (canvas as HTMLCanvasElement).getContext("2d")!;
          const pixels = context.getImageData(
            0,
            0,
            canvas.width,
            canvas.height,
          ).data;
          let white = 0,
            red = 0;
          for (let index = 0; index < pixels.length; index += 4) {
            if (pixels[index + 3] < 100) continue;
            if (
              pixels[index] > 220 &&
              pixels[index + 1] > 220 &&
              pixels[index + 2] > 220
            )
              white++;
            if (
              pixels[index] > 180 &&
              pixels[index + 1] < 120 &&
              pixels[index + 2] < 120
            )
              red++;
          }
          return white > 20 && red > 20;
        }),
    )
    .toBe(true);
  await page.screenshot({
    path: `../.cache/e2e/butterfly-page2-${info.project.name}.png`,
  });
  await page.getByRole("button", { name: "Rectangle", exact: true }).click();
  const bounds = (await page.getByTestId("canvas").boundingBox())!;
  await page.mouse.move(bounds.x + 150, bounds.y + 100);
  await page.mouse.down();
  await page.mouse.move(bounds.x + 250, bounds.y + 180, { steps: 6 });
  await page.mouse.up();
  await expect(page.getByTestId("element-count")).toHaveText("4 elements");
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const saved = (await (
    await page.request.get(`/api/v1/pages/${pages[1].id}/document`)
  ).json()) as { document: CanvasDocument };
  expect(saved.document.settings.backgroundColor).toBe("#000000");
  expect(errors).toEqual([]);
});
