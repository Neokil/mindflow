import { expect, test } from "@playwright/test";
import { login, create, activate } from "./helpers";
import { TabletInput } from "./tablet-input";
import type { CanvasDocument } from "../src/model";

test("lists, checkboxes and translucent highlights save and undo", async ({
  page,
  isMobile,
}) => {
  await login(page);
  await create(page, `Text tools ${Date.now()}`);
  const box = (await page.getByTestId("canvas").boundingBox())!;
  const id = page.url().split("/pages/")[1];
  const snapshot = async () => {
    await expect(page.getByTestId("save-status")).toHaveText("Saved");
    return (
      await (await page.request.get(`/api/v1/pages/${id}/document`)).json()
    ).document as CanvasDocument;
  };
  await activate(
    page,
    page.getByRole("button", { name: "Text (T)", exact: true }),
  );
  const input = new TabletInput(page);
  if (isMobile)
    await input.drag(
      { x: box.x + 400, y: box.y + 250 },
      { x: box.x + 750, y: box.y + 450 },
    );
  else {
    await page.mouse.move(box.x + 400, box.y + 250);
    await page.mouse.down();
    await page.mouse.move(box.x + 750, box.y + 450);
    await page.mouse.up();
  }
  await page.getByLabel("Edit canvas text").fill("First task\nSecond task");
  await activate(
    page,
    page.getByRole("button", { name: "Checklist", exact: true }),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Done editing text", exact: true }),
  );
  expect((await snapshot()).elements[0].listStyle).toBe("check");
  if (isMobile) await page.touchscreen.tap(box.x + 415, box.y + 268);
  else await page.mouse.click(box.x + 415, box.y + 268);
  expect((await snapshot()).elements[0].checked).toEqual([0]);
  await activate(page, page.getByRole("button", { name: "Undo", exact: true }));
  expect((await snapshot()).elements[0].checked || []).toEqual([]);
  await activate(
    page,
    page.getByRole("button", { name: "Highlighter (M)", exact: true }),
  );
  if (isMobile) {
    await input.pen("down", box.x + 460, box.y + 270);
    await input.pen("move", box.x + 650, box.y + 270);
    await input.pen("up", box.x + 650, box.y + 270);
  } else {
    await page.mouse.move(box.x + 460, box.y + 270);
    await page.mouse.down();
    await page.mouse.move(box.x + 650, box.y + 270, { steps: 8 });
    await page.mouse.up();
  }
  const doc = await snapshot();
  expect(doc.elements[1]).toMatchObject({
    type: "stroke",
    highlight: true,
    penWidth: 24,
  });
  await page.reload();
  await expect(page.getByTestId("canvas")).toBeVisible();
  expect((await snapshot()).elements[1].highlight).toBe(true);
});

for (const highlight of [false, true]) {
  test(`${highlight ? "marker" : "pen"} straightens on hold and keeps ordinary freehand strokes`, async ({
    page,
    isMobile,
    browserName,
  }) => {
    await login(page);
    await create(page, `Hold ink ${highlight} ${Date.now()}`);
    await activate(
      page,
      page.getByRole("button", {
        name: highlight ? "Highlighter (M)" : "Pen (P)",
        exact: true,
      }),
    );
    const box = (await page.getByTestId("canvas").boundingBox())!;
    const input = new TabletInput(page);
    if (isMobile && browserName === "chromium") await input.native();
    const point = (x: number, y: number) => ({ x: box.x + x, y: box.y + y });
    const down = async (x: number, y: number) => {
      const p = point(x, y);
      if (isMobile) await input.pen("down", p.x, p.y);
      else {
        await page.mouse.move(p.x, p.y);
        await page.mouse.down();
      }
    };
    const move = async (x: number, y: number) => {
      const p = point(x, y);
      if (isMobile) await input.pen("move", p.x, p.y);
      else await page.mouse.move(p.x, p.y);
    };
    const up = async (x: number, y: number) => {
      const p = point(x, y);
      if (isMobile) await input.pen("up", p.x, p.y);
      else {
        await page.mouse.move(p.x, p.y);
        await page.mouse.up();
      }
    };
    const id = page.url().split("/pages/")[1];
    const snapshot = async () => {
      await expect(page.getByTestId("save-status")).toHaveText("Saved");
      return (
        await (await page.request.get(`/api/v1/pages/${id}/document`)).json()
      ).document as CanvasDocument;
    };
    const squiggle = async () => {
      await down(250, 250);
      await move(290, 275);
      await move(340, 230);
      await move(400, 280);
      await move(460, 250);
    };
    await squiggle();
    await up(460, 250);
    await expect(page.getByTestId("element-count")).toHaveText("1 element");
    expect((await snapshot()).elements[0].points!.length).toBeGreaterThan(4);
    await squiggle();
    // Holding is the actual gesture under test, rather than a wait for rendering.
    await page.waitForTimeout(850);
    await expect(page.getByTestId("element-count")).toHaveText("1 element");
    // After straightening, dragging adjusts the endpoint without restoring the squiggle.
    await move(480, 290);
    await up(480, 290);
    await expect(page.getByTestId("element-count")).toHaveText("2 elements");
    const straight = (await snapshot()).elements[1];
    expect(straight.points).toEqual([0, 0, 1, 1]);
    expect(straight.width).toBeCloseTo(230);
    expect(straight.height).toBeCloseTo(40);
    expect(!!straight.highlight).toBe(highlight);
    expect(straight.penWidth).toBe(highlight ? 24 : 3);
    await activate(
      page,
      page.getByRole("button", { name: "Undo", exact: true }),
    );
    expect((await snapshot()).elements).toHaveLength(1);
    await activate(
      page,
      page.getByRole("button", { name: "Redo", exact: true }),
    );
    expect((await snapshot()).elements[1].points).toEqual(straight.points);
    if (!highlight) {
      await squiggle();
      await page.waitForTimeout(850);
      if (isMobile && browserName !== "chromium") {
        const p = point(460, 250);
        await input.pen("cancel", p.x, p.y);
      } else {
        await page.keyboard.press("Escape");
        await up(460, 250);
      }
      await expect(page.getByTestId("element-count")).toHaveText("2 elements");
      await activate(
        page,
        page.getByRole("button", { name: "Pen (P)", exact: true }),
      );
      await squiggle();
      await up(460, 250);
      await expect(page.getByTestId("element-count")).toHaveText("3 elements");
      expect((await snapshot()).elements[2].points!.length).toBeGreaterThan(4);
    }
  });
}
