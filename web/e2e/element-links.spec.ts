import { selectTool } from "./helpers";
import { TabletInput } from "./tablet-input";
import { test, expect } from "@playwright/test";
import { activate, create, login } from "./helpers";
import type { CanvasDocument } from "../src/model";

async function document(page: import("@playwright/test").Page) {
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  return (
    await (
      await page.request.get(
        `/api/v1/pages/${page.url().split("/pages/")[1]}/document`,
      )
    ).json()
  ).document as CanvasDocument;
}

test("align objects and navigate an element link across pages", async ({
  page,
  browserName,
}, info) => {
  await login(page);
  await create(page, `Navigation ${info.project.name}`);
  const box = (await page.getByTestId("canvas").boundingBox())!;
  const draw = async (x: number, y: number) => {
    await selectTool(page, "Rectangle");
    await page.mouse.move(box.x + x, box.y + y);
    await page.mouse.down();
    await page.mouse.move(box.x + x + 90, box.y + y + 60);
    await page.mouse.up();
  };
  await draw(250, 220);
  await draw(500, 330);
  await activate(
    page,
    page.getByRole("button", { name: "Select (V)", exact: true }),
  );
  await page.keyboard.press("ControlOrMeta+a");
  await activate(
    page,
    page.getByRole("button", { name: "Align selection", exact: true }),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Align left", exact: true }),
  );
  const aligned = await document(page);
  expect(aligned.elements[0].x).toBeCloseTo(aligned.elements[1].x);
  const firstURL = page.url();
  await activate(
    page,
    page.getByRole("button", { name: "Add page", exact: true }),
  );
  await page.getByLabel("Name", { exact: true }).fill("Link destination");
  await activate(page, page.getByRole("button", { name: "Save", exact: true }));
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  await expect(
    page.getByRole("tab", { name: "Link destination", exact: true }),
  ).toHaveAttribute("aria-selected", "true");
  await expect(page).not.toHaveURL(firstURL);
  await expect(page.getByTestId("element-count")).toHaveText("0 elements");
  const secondURL = page.url();
  await draw(380, 240);
  const target = (await document(page)).elements[0];
  await activate(
    page,
    page.getByRole("tab", { name: "Untitled", exact: true }),
  );
  await expect(page).toHaveURL(firstURL);
  await activate(
    page,
    page.getByRole("button", { name: "Select (V)", exact: true }),
  );
  await page.keyboard.press("ControlOrMeta+a");
  await activate(
    page,
    page.getByRole("button", { name: "Link selection", exact: true }),
  );
  await page
    .getByLabel("Target page", { exact: true })
    .selectOption(secondURL.split("/pages/")[1]);
  await page
    .getByLabel("Target element", { exact: true })
    .selectOption(target.id);
  await activate(
    page,
    page.getByRole("button", { name: "Save link", exact: true }),
  );
  const linked = await document(page);
  expect(linked.elements.every((e) => e.link?.elementId === target.id)).toBe(
    true,
  );
  await page.reload();
  await expect(page.getByTestId("canvas")).toBeVisible();
  // Fit-all on open positions the two aligned shapes around the middle of the canvas.
  const canvas = page.locator(".konvajs-content");
  const source = (await document(page)).elements[0];
  const all = (await document(page)).elements;
  const { fitCamera } = await import("../src/geometry");
  const bounds = (await canvas.boundingBox())!;
  const camera = fitCamera(all, { width: bounds.width, height: bounds.height });
  const input = new TabletInput(page);
  if (browserName === "chromium") await input.native();
  const sourceX = bounds.x + camera.x + source.x * camera.zoom,
    sourceY = bounds.y + camera.y + source.y * camera.zoom;
  await input.pen("down", sourceX, sourceY);
  await input.pen("up", sourceX, sourceY);
  await page.waitForTimeout(350);
  await input.pen("down", sourceX, sourceY);
  await input.pen("up", sourceX, sourceY);
  await expect(
    page.getByRole("complementary", { name: "Object properties", exact: true }),
  ).toBeVisible();
  await page.waitForTimeout(500);
  await expect(page).toHaveURL(firstURL);
  await activate(
    page,
    page.getByRole("button", { name: "Close settings", exact: true }),
  );
  await page.mouse.click(
    bounds.x + camera.x + source.x * camera.zoom,
    bounds.y + camera.y + source.y * camera.zoom,
  );
  await expect(page).toHaveURL(secondURL);
  await expect(
    page.getByRole("tab", { name: "Link destination", exact: true }),
  ).toHaveAttribute("aria-selected", "true");
});
