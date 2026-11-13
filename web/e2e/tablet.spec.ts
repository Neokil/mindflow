import { test, expect } from "@playwright/test";
import type { Page } from "@playwright/test";
import { create, login } from "./helpers";
import { TabletInput } from "./tablet-input";
import type { CanvasDocument } from "../src/model";

async function prepare(page: Page, browserName: string, name: string) {
  await login(page);
  await create(page, `${name} ${Date.now()}`);
  const input = new TabletInput(page);
  if (browserName === "chromium") await input.native();
  const box = (await page.getByTestId("canvas").boundingBox())!;
  await page.evaluate(
    () =>
      new Promise<void>((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
      ),
  );
  const point = (x: number, y: number) => ({ x: box.x + x, y: box.y + y });
  const id = page.url().split("/pages/")[1];
  const document = async () => {
    await expect(page.getByTestId("save-status")).toHaveText("Saved");
    return (
      await (await page.request.get(`/api/v1/pages/${id}/document`)).json()
    ).document as CanvasDocument;
  };
  return { input, point, document, box };
}

test("finger creation, Pencil ink, palm rejection, cancellation and two-finger navigation", async ({
  page,
  browserName,
}, info) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  const { input, point, document } = await prepare(
    page,
    browserName,
    "Tablet drawing",
  );
  const url = page.url();
  let navigations = 0;
  page.on("framenavigated", (frame) => {
    if (frame === page.mainFrame()) navigations++;
  });
  await expect(page.locator(".konvajs-content")).toHaveCSS(
    "touch-action",
    "none",
  );
  await expect(page.locator("html")).toHaveCSS("overscroll-behavior-y", "none");
  await page.getByRole("button", { name: "Rectangle", exact: true }).tap();
  await input.drag(point(250, 180), point(450, 320));
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  expect((await document()).elements[0]).toMatchObject({
    type: "shape",
    width: 200,
    height: 140,
  });
  // A pinch that starts during creation discards the unfinished preview.
  await page.getByRole("button", { name: "Circle", exact: true }).tap();
  const first = point(300, 400),
    second = point(500, 400);
  await input.touch("down", 1, first.x, first.y);
  await input.touch("move", 1, first.x + 40, first.y + 40);
  await input.touch("down", 2, second.x, second.y);
  await input.touch("up", 2, second.x, second.y);
  await input.touch("up", 1, first.x + 40, first.y + 40);
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  await page.getByRole("button", { name: "Pen (P)", exact: true }).tap();
  const a = point(280, 400),
    b = point(420, 470);
  await input.pen("down", a.x, a.y);
  // A palm/finger while Pencil is in contact must not pan or add another stroke.
  const palm = point(600, 500);
  await input.touch("down", 5, palm.x, palm.y);
  await input.pen("move", (a.x + b.x) / 2, a.y + 20);
  await input.touch("move", 5, palm.x + 30, palm.y + 40);
  await input.pen("move", b.x, b.y);
  await input.pen("up", b.x, b.y);
  await input.touch("up", 5, palm.x + 30, palm.y + 40);
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  const before = await document();
  expect(before.elements[1].type).toBe("stroke");
  expect(before.elements[1].points!.length).toBeGreaterThan(4);
  const p = point(300, 350),
    q = point(430, 350);
  await input.touch("down", 1, p.x, p.y);
  await input.touch("down", 2, q.x, q.y);
  await input.touch("move", 2, q.x + 120, q.y + 40);
  await input.touch("up", 2, q.x + 120, q.y + 40);
  await input.touch("move", 1, p.x + 20, p.y + 30);
  await input.touch("up", 1, p.x + 20, p.y + 30);
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  expect((await document()).elements).toEqual(before.elements);
  await expect(page.locator(".zoom-value")).not.toHaveText("100%");
  await input.touch("down", 1, p.x, p.y);
  await input.touch("move", 1, p.x + 50, p.y + 50);
  await input.touch("cancel", 1, p.x + 50, p.y + 50);
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  await page.getByLabel("Pencil only", { exact: true }).check();
  const background = await page
    .getByTestId("canvas")
    .evaluate((element) => element.style.backgroundPosition);
  await input.drag(point(300, 400), point(350, 450));
  expect(
    await page
      .getByTestId("canvas")
      .evaluate((element) => element.style.backgroundPosition),
  ).not.toBe(background);
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  expect(await page.evaluate(() => ({ x: scrollX, y: scrollY }))).toEqual({
    x: 0,
    y: 0,
  });
  expect(page.url()).toBe(url);
  expect(navigations).toBe(0);
  expect(errors).toEqual([]);
  await page.screenshot({
    path: `../.cache/e2e/tablet-drawing-${info.project.name}.png`,
  });
});

test("touch moves and resizes objects, text has a Done button, and Pan works without a keyboard", async ({
  page,
  browserName,
}) => {
  const { input, point, document } = await prepare(
    page,
    browserName,
    "Tablet editing",
  );
  await page.getByRole("button", { name: "Rectangle", exact: true }).tap();
  await input.drag(point(250, 180), point(450, 320));
  await document();
  await page.getByRole("button", { name: "Select (V)", exact: true }).tap();
  await input.drag(point(350, 250), point(410, 290));
  let doc = await document();
  const moved = doc.elements[0];
  expect(moved.x).toBeCloseTo(
    410 - (await page.getByTestId("canvas").boundingBox())!.width / 2,
    0,
  );
  // Starting navigation during a native Konva object drag restores its position.
  const d = point(410, 290),
    other = point(650, 400);
  await input.touch("down", 1, d.x, d.y);
  await input.touch("move", 1, d.x + 30, d.y + 20);
  await input.touch("down", 2, other.x, other.y);
  await input.touch("up", 2, other.x, other.y);
  await input.touch("up", 1, d.x + 30, d.y + 20);
  expect((await document()).elements[0]).toEqual(moved);
  // The enlarged bottom-right Transformer anchor remains usable by a finger.
  await input.drag(point(514, 364), point(574, 414));
  doc = await document();
  expect(doc.elements[0].width).toBeGreaterThan(moved.width + 30);
  expect(doc.elements[0].height).toBeGreaterThan(moved.height + 20);
  await page.getByRole("button", { name: "Text (T)", exact: true }).tap();
  await input.drag(point(250, 500), point(550, 600));
  await page.getByLabel("Edit canvas text").fill("Written on a tablet");
  // Emulate an overlaid software keyboard: the layout viewport stays full size.
  await page.evaluate(() => {
    Object.defineProperty(visualViewport!, "height", {
      configurable: true,
      value: 500,
    });
    visualViewport!.dispatchEvent(new Event("resize"));
  });
  await expect
    .poll(async () => {
      const done = await page
        .getByRole("button", { name: "Done editing text", exact: true })
        .boundingBox();
      return done!.y + done!.height;
    })
    .toBeLessThanOrEqual(500);
  await page
    .getByRole("button", { name: "Done editing text", exact: true })
    .tap();
  await expect(page.getByLabel("Edit canvas text")).toHaveCount(0);
  await page.evaluate(() => {
    Reflect.deleteProperty(visualViewport!, "height");
    visualViewport!.dispatchEvent(new Event("resize"));
  });
  expect(
    (await document()).elements.some(
      (element) => element.text === "Written on a tablet",
    ),
  ).toBe(true);
  // Double tapping an existing text element must reopen the editor as well.
  await input.touch("down", 1, point(400, 550).x, point(400, 550).y);
  await input.touch("up", 1, point(400, 550).x, point(400, 550).y);
  await input.touch("down", 1, point(400, 550).x, point(400, 550).y);
  await input.touch("up", 1, point(400, 550).x, point(400, 550).y);
  await expect(page.getByLabel("Edit canvas text")).toBeVisible();
  await page
    .getByRole("button", { name: "Done editing text", exact: true })
    .tap();
  await page.getByRole("button", { name: "Pan (H)", exact: true }).tap();
  const before = (await document()).elements;
  await input.drag(point(300, 400), point(360, 450));
  expect((await document()).elements).toEqual(before);
  await page.setViewportSize({ width: 1194, height: 834 });
  await expect(page.getByTestId("canvas")).toBeVisible();
  expect(
    await page.evaluate(
      () => window.document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
});

test("image upload, object properties and comments work without a mouse", async ({
  page,
  browserName,
}) => {
  const { input, point, document } = await prepare(
    page,
    browserName,
    "Tablet images",
  );
  const image = await page.evaluate(() => {
    const canvas = window.document.createElement("canvas");
    canvas.width = 200;
    canvas.height = 100;
    const context = canvas.getContext("2d")!;
    context.fillStyle = "#4caf50";
    context.fillRect(0, 0, 200, 100);
    return canvas.toDataURL("image/png").split(",")[1];
  });
  await page.locator('input[type="file"]').setInputFiles({
    name: "tablet.png",
    mimeType: "image/png",
    buffer: Buffer.from(image, "base64"),
  });
  await expect(
    page.getByText("Drag to size image", { exact: false }),
  ).toBeVisible();
  await input.drag(point(250, 220), point(550, 370));
  expect((await document()).elements[0]).toMatchObject({
    type: "image",
    width: 300,
    height: 150,
  });
  await page
    .getByRole("button", { name: "Selection properties", exact: true })
    .tap();
  await expect(
    page.getByLabel("Object properties", { exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Close settings", exact: true }).tap();
  await page.getByRole("button", { name: "Add comment", exact: true }).tap();
  const p = point(500, 500);
  await input.touch("down", 1, p.x, p.y);
  await input.touch("up", 1, p.x, p.y);
  await page
    .getByLabel("Comment text", { exact: true })
    .fill("A tablet comment");
  await page.getByRole("button", { name: "Post comment", exact: true }).tap();
  const author = await (await page.request.get("/api/v1/auth/me")).json();
  await page
    .getByRole("button", {
      name: `Comment by ${author.user.displayName}`,
      exact: true,
    })
    .tap();
  await expect(
    page.getByText("A tablet comment", { exact: true }),
  ).toBeVisible();
  await page
    .getByLabel("Reply to comment", { exact: true })
    .fill("And a reply");
  await page.getByRole("button", { name: "Reply", exact: true }).tap();
  await expect(page.getByText("And a reply", { exact: true })).toBeVisible();
});

test("Pencil double taps open object properties beside the pen, including for text", async ({
  page,
  browserName,
}) => {
  const { input, point } = await prepare(
    page,
    browserName,
    "Pencil properties",
  );
  await page.getByRole("button", { name: "Rectangle", exact: true }).tap();
  await input.drag(point(250, 220), point(450, 340));
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const doubleTap = async (x: number, y: number) => {
    const p = point(x, y);
    for (let tap = 0; tap < 2; tap++) {
      await input.pen("down", p.x, p.y);
      await input.pen("up", p.x, p.y);
      if (tap === 0)
        await expect(
          page.getByLabel("Object properties", { exact: true }),
        ).toHaveCount(0);
    }
    await expect(
      page.getByLabel("Object properties", { exact: true }),
    ).toBeVisible();
    const panel = (await page
      .getByLabel("Object properties", { exact: true })
      .boundingBox())!;
    expect(panel.x).toBeCloseTo(p.x, 0);
    await expect(page.getByLabel("Edit canvas text")).toHaveCount(0);
    await page
      .getByRole("button", { name: "Close settings", exact: true })
      .tap();
  };
  await doubleTap(350, 280);
  await page.getByRole("button", { name: "Text (T)", exact: true }).tap();
  await input.drag(point(250, 500), point(450, 600));
  await page.getByLabel("Edit canvas text").fill("Pencil properties");
  await page
    .getByRole("button", { name: "Done editing text", exact: true })
    .tap();
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  await doubleTap(350, 550);
});
