import { expect, test } from "@playwright/test";
import { activate, create, login } from "./helpers";
import { TabletInput } from "./tablet-input";
import type { CanvasDocument } from "../src/model";

test("pressure ink retains its profile through hold, editing history, and reload", async ({
  page,
  browserName,
}, info) => {
  await login(page);
  await create(page, `Pressure ${info.project.name}`);
  await activate(
    page,
    page.getByRole("button", { name: "Pen (P)", exact: true }),
  );
  const pressure = page.getByRole("checkbox", {
    name: "Pressure-sensitive ink",
    exact: true,
  });
  await expect(pressure).toBeChecked();
  const smoothing = page.getByLabel("Stroke stabilization", { exact: true });
  await expect(smoothing).toHaveValue("light");
  await smoothing.selectOption("strong");
  await page.getByLabel("Pen thickness", { exact: true }).press("End");
  await expect(page.getByLabel("Pen thickness", { exact: true })).toHaveValue(
    "40",
  );
  const input = new TabletInput(page);
  if (browserName === "chromium") await input.native();
  const box = (await page.getByTestId("canvas").boundingBox())!;
  const id = page.url().split("/pages/")[1];
  const snapshot = async () => {
    await expect(page.getByTestId("save-status")).toHaveText("Saved");
    return (
      await (await page.request.get(`/api/v1/pages/${id}/document`)).json()
    ).document as CanvasDocument;
  };
  await input.pen("down", box.x + 250, box.y + 260, 0.1);
  for (const [x, y, p] of [
    [300, 280, 0.2],
    [350, 240, 0.4],
    [430, 285, 0.6],
    [500, 240, 0.9],
    [600, 260, 0.8],
  ])
    await input.pen("move", box.x + x, box.y + y, p);
  await page.waitForTimeout(850);
  await input.pen("move", box.x + 650, box.y + 320, 0.7);
  await input.pen("up", box.x + 650, box.y + 320, 0);
  const stroke = (await snapshot()).elements[0];
  expect(stroke.pressures!.length).toBe(stroke.points!.length / 2);
  expect(stroke.pressures![0]).toBeCloseTo(0.1);
  expect(stroke.pressures!.at(-1)).toBeCloseTo(0.7);
  expect(Math.max(...stroke.pressures!)).toBeGreaterThan(0.85);
  expect(stroke.width).toBeCloseTo(400);
  expect(stroke.height).toBeCloseTo(60);
  expect(stroke.points!.length).toBeGreaterThan(4);
  for (let i = 0; i < stroke.points!.length; i += 2)
    expect(stroke.points![i]).toBeCloseTo(stroke.points![i + 1]);
  await page.screenshot({
    path: `../.cache/e2e/pressure-ink-${info.project.name}.png`,
  });
  await activate(page, page.getByRole("button", { name: "Undo", exact: true }));
  expect((await snapshot()).elements).toHaveLength(0);
  await activate(page, page.getByRole("button", { name: "Redo", exact: true }));
  expect((await snapshot()).elements[0]).toEqual(stroke);
  await activate(
    page,
    page.getByRole("button", { name: "Pen (P)", exact: true }),
  );
  await activate(
    page,
    page.getByRole("checkbox", { name: "Pressure-sensitive ink", exact: true }),
  );
  await smoothing.selectOption("off");
  await input.pen("down", box.x + 300, box.y + 420, 0.1);
  await input.pen("move", box.x + 450, box.y + 450, 0.9);
  await input.pen("up", box.x + 500, box.y + 420, 0);
  expect((await snapshot()).elements[1].pressures).toBeUndefined();
  await activate(
    page,
    page.getByRole("button", { name: "Highlighter (M)", exact: true }),
  );
  await smoothing.selectOption("strong");
  await expect(
    page.getByRole("checkbox", { name: "Pressure-sensitive ink", exact: true }),
  ).toHaveCount(0);
  await input.pen("down", box.x + 300, box.y + 500, 0.1);
  await input.pen("move", box.x + 400, box.y + 520, 0.9);
  await input.pen("move", box.x + 500, box.y + 500, 0.6);
  await page.waitForTimeout(850);
  await input.pen("up", box.x + 500, box.y + 500, 0);
  const marker = (await snapshot()).elements[2];
  expect(marker.highlight).toBe(true);
  expect(marker.pressures).toBeUndefined();
  expect(marker.points).toHaveLength(4);
  await page.reload();
  await expect(page.getByTestId("canvas")).toBeVisible();
  expect((await snapshot()).elements[0]).toEqual(stroke);
});

test("coalesced Pencil samples keep their individual pressure values", async ({
  page,
}, info) => {
  await login(page);
  await create(page, `Coalesced ${info.project.name}`);
  await activate(
    page,
    page.getByRole("button", { name: "Pen (P)", exact: true }),
  );
  await page
    .getByLabel("Stroke stabilization", { exact: true })
    .selectOption("off");
  const box = (await page.getByTestId("canvas").boundingBox())!;
  const input = new TabletInput(page);
  await input.pen("down", box.x + 300, box.y + 300, 0.1);
  await page.locator(".konvajs-content").evaluate((surface, box) => {
    const sample = (x: number, p: number) =>
      new PointerEvent("pointermove", {
        bubbles: true,
        pointerId: 99,
        pointerType: "pen",
        isPrimary: true,
        button: 0,
        buttons: 1,
        clientX: box.x + x,
        clientY: box.y + 300,
        pressure: p,
      });
    const event = sample(400, 0.9);
    Object.defineProperty(event, "getCoalescedEvents", {
      value: () => [sample(350, 0.2), sample(400, 0.9)],
    });
    surface.dispatchEvent(event);
  }, box);
  await input.pen("up", box.x + 400, box.y + 300, 0);
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const id = page.url().split("/pages/")[1];
  const doc = (
    await (await page.request.get(`/api/v1/pages/${id}/document`)).json()
  ).document as CanvasDocument;
  expect(doc.elements[0].pressures).toHaveLength(3);
  expect(doc.elements[0].pressures![1]).toBeCloseTo(0.2);
  expect(doc.elements[0].pressures![2]).toBeCloseTo(0.9);
});
