import { expect, test } from "@playwright/test";
import { activate, create, login, selectTool } from "./helpers";
import { TabletInput } from "./tablet-input";
import type { CanvasDocument } from "../src/model";
test("shape dropdown, tooltips and minimal header controls work together", async ({
  page,
  browserName,
  isMobile,
}, info) => {
  await login(page);
  await create(page, `Tools ${info.project.name}`);
  await expect(
    page.getByRole("button", { name: "Workspace trash", exact: true }),
  ).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Rectangle", exact: true }),
  ).toHaveCount(0);
  await expect(
    page
      .getByRole("button", { name: "Import pages", exact: true })
      .locator("svg"),
  ).toHaveClass(/lucide-download/);
  await expect(
    page
      .getByRole("button", { name: "Export workspace", exact: true })
      .locator("svg"),
  ).toHaveClass(/lucide-upload/);
  const toggle = page.getByRole("button", {
    name: "Hide app header and pages",
    exact: true,
  });
  const style = await toggle.evaluate((element) => ({
    border: getComputedStyle(element).borderWidth,
    right: element.getBoundingClientRect().right,
    width: innerWidth,
  }));
  expect(style.border).toBe("0px");
  expect(style.right).toBe(style.width);
  const undo = page.getByRole("button", { name: "Undo", exact: true });
  await expect(undo).toBeDisabled();
  await undo.hover();
  await expect(page.getByRole("tooltip")).toHaveText("Undo");
  const shapes = page.getByRole("button", { name: "Shapes", exact: true });
  await shapes.focus();
  await expect(page.getByRole("tooltip")).toHaveText(
    "Shapes: rectangle, circle, or star",
  );
  await expect(shapes).toHaveAttribute(
    "aria-describedby",
    (await page.getByRole("tooltip").getAttribute("id"))!,
  );
  await page.keyboard.press("Escape");
  await expect(page.getByRole("tooltip")).toHaveCount(0);
  await expect(shapes).not.toHaveAttribute("aria-describedby");
  if (isMobile) {
    if (browserName === "chromium") {
      const input = new TabletInput(page);
      await input.native();
      const box = (await shapes.boundingBox())!;
      const x = box.x + box.width / 2,
        y = box.y + box.height / 2;
      await input.touch("down", 10, x, y);
      await expect(page.getByRole("tooltip")).toBeVisible();
      await input.touch("up", 10, x, y);
    } else {
      await shapes.dispatchEvent("pointerdown", {
        pointerType: "touch",
        pointerId: 10,
        clientX: 100,
        clientY: 200,
      });
      await expect(page.getByRole("tooltip")).toBeVisible();
      await shapes.dispatchEvent("pointerup", {
        pointerType: "touch",
        pointerId: 10,
      });
      await shapes.dispatchEvent("click");
    }
    await expect(shapes).toHaveAttribute("aria-expanded", "false");
  }
  await activate(page, shapes);
  await expect(
    page.getByRole("menu", { name: "Shape tools", exact: true }),
  ).toBeVisible();
  await expect(page.getByRole("menuitemradio")).toHaveCount(3);
  await page.keyboard.press("End");
  await expect(
    page.getByRole("menuitemradio", { name: "Star", exact: true }),
  ).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(shapes).toHaveAttribute("aria-pressed", "true");
  await expect(
    page.getByRole("menu", { name: "Shape tools", exact: true }),
  ).toHaveCount(0);
  await selectTool(page, "Rectangle");
  const box = (await page.getByTestId("canvas").boundingBox())!;
  await page.mouse.move(box.x + 300, box.y + 220);
  await page.mouse.down();
  await page.mouse.move(box.x + 460, box.y + 320);
  await page.mouse.up();
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  await activate(
    page,
    page.getByRole("button", { name: "Selection properties", exact: true }),
  );
  const close = page.getByRole("button", {
    name: "Close settings",
    exact: true,
  });
  await close.focus();
  await expect(page.getByRole("tooltip")).toHaveText("Close settings");
  await activate(page, close);
  await expect(
    page.getByLabel("Object properties", { exact: true }),
  ).toHaveCount(0);
  await page.screenshot({
    path: `../.cache/e2e/canvas-tools-${info.project.name}.png`,
  });
  await activate(
    page,
    page.getByRole("button", { name: "Import pages", exact: true }),
  );
  const dialog = page.getByRole("dialog");
  const dismiss = dialog.getByRole("button", {
    name: "Close dialog",
    exact: true,
  });
  await dialog.locator("input").first().focus();
  await dismiss.focus();
  await expect(dialog.getByRole("tooltip")).toHaveText("Close dialog");
  await activate(page, dismiss);
});

test("holding Pencil and marker loops produces clean closed shapes with saved pressure", async ({
  page,
  browserName,
}, info) => {
  await login(page);
  await create(page, `Shape assist ${info.project.name}`);
  const input = new TabletInput(page);
  if (browserName === "chromium") await input.native();
  const box = (await page.getByTestId("canvas").boundingBox())!;
  const path = async (points: { x: number; y: number }[]) => {
    await input.pen("down", box.x + points[0].x, box.y + points[0].y, 0.1);
    for (let i = 1; i < points.length; i++)
      await input.pen(
        "move",
        box.x + points[i].x,
        box.y + points[i].y,
        0.1 + (0.8 * i) / (points.length - 1),
      );
    await page.waitForTimeout(850);
    const end = points.at(-1)!;
    await input.pen("up", box.x + end.x, box.y + end.y, 0);
    await expect(page.getByTestId("save-status")).toHaveText("Saved");
  };
  const snapshot = async () =>
    (
      await (
        await page.request.get(
          `/api/v1/pages/${page.url().split("/pages/")[1]}/document`,
        )
      ).json()
    ).document as CanvasDocument;
  await activate(
    page,
    page.getByRole("button", { name: "Pen (P)", exact: true }),
  );
  await path(
    Array.from({ length: 65 }, (_, i) => ({
      x: 350 + 85 * Math.cos((i / 64) * Math.PI * 2) + ((i % 3) - 1) * 2,
      y: 300 + 85 * Math.sin((i / 64) * Math.PI * 2) + ((i % 4) - 1.5) * 2,
    })),
  );
  const circle = (await snapshot()).elements[0];
  expect(circle.points!.slice(0, 2)).toEqual(circle.points!.slice(-2));
  for (let i = 0; i < circle.points!.length; i += 2)
    expect(
      Math.hypot(circle.points![i] * 2 - 1, circle.points![i + 1] * 2 - 1),
    ).toBeCloseTo(1, 1);
  expect(Math.max(...circle.pressures!)).toBeGreaterThan(0.85);
  await activate(
    page,
    page.getByRole("button", { name: "Highlighter (M)", exact: true }),
  );
  const corners = [
    [480, 400],
    [680, 400],
    [680, 540],
    [480, 540],
    [480, 400],
  ];
  await path(
    corners
      .slice(0, -1)
      .flatMap(([x, y], edge) =>
        Array.from({ length: 10 }, (_, i) => ({
          x: x + ((corners[edge + 1][0] - x) * i) / 10 + ((i % 3) - 1),
          y: y + ((corners[edge + 1][1] - y) * i) / 10 + (i % 2 ? 1 : -1),
        })),
      )
      .concat([{ x: 480, y: 400 }]),
  );
  const saved = await snapshot(),
    rectangle = saved.elements[1];
  expect(rectangle.highlight).toBe(true);
  expect(rectangle.pressures).toBeUndefined();
  expect(rectangle.points!.slice(0, 2)).toEqual(rectangle.points!.slice(-2));
  for (let i = 0; i < rectangle.points!.length; i += 2) {
    const x = rectangle.points![i],
      y = rectangle.points![i + 1];
    expect(Math.min(x, 1 - x, y, 1 - y)).toBeLessThan(0.02);
  }
  await page.screenshot({
    path: `../.cache/e2e/shape-assist-${info.project.name}.png`,
  });
  await page.reload();
  await expect(page.getByTestId("canvas")).toBeVisible();
  expect(await snapshot()).toEqual(saved);
});

test("Shift constrains held ink shapes and lines before release", async ({
  page,
}, info) => {
  await login(page);
  await create(page, `Shift assistance ${info.project.name}`);
  const box = (await page.getByTestId("canvas").boundingBox())!;
  const draw = async (points: { x: number; y: number }[]) => {
    await page.mouse.move(box.x + points[0].x, box.y + points[0].y);
    await page.mouse.down();
    for (const point of points.slice(1))
      await page.mouse.move(box.x + point.x, box.y + point.y);
    await page.waitForTimeout(850);
    // No pointer movement follows: the keyboard must update the held preview.
    await page.keyboard.down("Shift");
    await page.mouse.up();
    await page.keyboard.up("Shift");
    await expect(page.getByTestId("save-status")).toHaveText("Saved");
  };
  await activate(
    page,
    page.getByRole("button", { name: "Pen (P)", exact: true }),
  );
  await draw(
    Array.from({ length: 49 }, (_, i) => ({
      x: 300 + 90 * Math.cos((i / 48) * Math.PI * 2),
      y: 240 + 50 * Math.sin((i / 48) * Math.PI * 2),
    })),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Highlighter (M)", exact: true }),
  );
  const corners = [
    [490, 330],
    [690, 330],
    [690, 440],
    [490, 440],
    [490, 330],
  ];
  await draw(
    corners
      .slice(0, -1)
      .flatMap(([x, y], edge) =>
        Array.from({ length: 10 }, (_, i) => ({
          x: x + ((corners[edge + 1][0] - x) * i) / 10,
          y: y + ((corners[edge + 1][1] - y) * i) / 10,
        })),
      )
      .concat([{ x: 490, y: 330 }]),
  );
  await activate(
    page,
    page.getByRole("button", { name: "Pen (P)", exact: true }),
  );
  await draw([
    { x: 160, y: 450 },
    { x: 200, y: 465 },
    { x: 245, y: 450 },
    { x: 300, y: 480 },
  ]);
  const saved = (
    await (
      await page.request.get(
        `/api/v1/pages/${page.url().split("/pages/")[1]}/document`,
      )
    ).json()
  ).document as CanvasDocument;
  expect(saved.elements).toHaveLength(3);
  for (const shape of saved.elements.slice(0, 2))
    expect(shape.width / shape.height).toBeCloseTo(1, 2);
  const line = saved.elements[2];
  expect(line.width).toBeGreaterThan(100);
  expect(line.height).toBeLessThan(1);
  for (let i = 1; i < line.points!.length; i += 2)
    expect(line.points![i]).toBe(0);
});
