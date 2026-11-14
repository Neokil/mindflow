import { selectTool } from "./helpers";
import { expect, test } from "@playwright/test";
import { create, login } from "./helpers";
import type { CanvasDocument } from "../src/model";

test("fast mouse releases use final coordinates even before a preview renders", async ({
  page,
}) => {
  await login(page);
  await create(page, `Fast mouse ${Date.now()}`);
  await selectTool(page, "Rectangle");
  // All events run in one task, without a React render between move and release.
  await page.locator(".konvajs-content").evaluate((surface) => {
    const box = surface.getBoundingClientRect();
    for (const [type, x, y, buttons] of [
      ["pointerdown", 300, 220, 1],
      ["pointermove", 450, 300, 1],
      ["pointerup", 500, 340, 0],
    ] as const) {
      surface.dispatchEvent(
        new PointerEvent(type, {
          bubbles: true,
          pointerType: "mouse",
          pointerId: 1,
          button: 0,
          buttons,
          clientX: box.x + x,
          clientY: box.y + y,
        }),
      );
    }
  });
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const id = page.url().split("/pages/")[1];
  const doc = (
    await (await page.request.get(`/api/v1/pages/${id}/document`)).json()
  ).document as CanvasDocument;
  expect(doc.elements[0]).toMatchObject({ width: 200, height: 120 });
  await selectTool(page, "Rectangle");
  await page.locator(".konvajs-content").evaluate((surface) => {
    const box = surface.getBoundingClientRect();
    for (const [type, x, y, buttons] of [
      ["pointerdown", 300, 400, 1],
      ["pointermove", 450, 480, 1],
      ["pointerup", 500, 520, 0],
    ] as const) {
      // A release outside the stage must still finish the mouse gesture.
      (type === "pointerup" ? window : surface).dispatchEvent(
        new PointerEvent(type, {
          bubbles: true,
          pointerType: "mouse",
          pointerId: 1,
          button: 0,
          buttons,
          clientX: box.x + x,
          clientY: box.y + y,
        }),
      );
    }
  });
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const after = (
    await (await page.request.get(`/api/v1/pages/${id}/document`)).json()
  ).document as CanvasDocument;
  expect(after.elements[1]).toMatchObject({ width: 200, height: 120 });
});

test("losing mouse capture preserves creation and an object drag", async ({
  page,
}) => {
  await login(page);
  await create(page, `Mouse capture ${Date.now()}`);
  const surface = page.locator(".konvajs-content");
  const box = (await surface.boundingBox())!;
  const rememberPointer = () =>
    surface.evaluate((element) => {
      element.addEventListener(
        "pointerdown",
        (event) => {
          element.setAttribute(
            "data-active-pointer",
            String((event as PointerEvent).pointerId),
          );
        },
        { once: true },
      );
    });
  const releaseCapture = () =>
    surface.evaluate((element) => {
      const id = Number(element.getAttribute("data-active-pointer"));
      const target = [element, ...element.querySelectorAll("canvas")].find(
        (node) => node.hasPointerCapture(id),
      );
      if (!target) throw new Error("Expected a captured mouse pointer");
      target.releasePointerCapture(id);
    });
  await selectTool(page, "Rectangle");
  await rememberPointer();
  await page.mouse.move(box.x + 300, box.y + 220);
  await page.mouse.down();
  await page.mouse.move(box.x + 500, box.y + 340);
  await releaseCapture();
  await page.mouse.move(box.x + 500, box.y + 340);
  await page.mouse.up();
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const id = page.url().split("/pages/")[1];
  const snapshot = async () => {
    await expect(page.getByTestId("save-status")).toHaveText("Saved");
    return (
      await (await page.request.get(`/api/v1/pages/${id}/document`)).json()
    ).document as CanvasDocument;
  };
  const before = (await snapshot()).elements[0];
  expect(before).toMatchObject({ width: 200, height: 120 });
  await rememberPointer();
  await page.mouse.move(box.x + 400, box.y + 280);
  await page.mouse.down();
  await page.mouse.move(box.x + 440, box.y + 300, { steps: 3 });
  await releaseCapture();
  await page.mouse.move(box.x + 440, box.y + 300);
  await page.mouse.up();
  const after = (await snapshot()).elements[0];
  expect(after.x).toBeCloseTo(before.x + 40);
  expect(after.y).toBeCloseTo(before.y + 20);
  // A subsequent gesture must work normally after capture loss.
  await page.mouse.move(box.x + 440, box.y + 300);
  await page.mouse.down();
  await page.mouse.move(box.x + 460, box.y + 310, { steps: 3 });
  await page.mouse.up();
  expect((await snapshot()).elements[0].x).toBeCloseTo(before.x + 60);
});
