import { selectTool } from "./helpers";
import { test, expect } from "@playwright/test";
import type { Page } from "@playwright/test";
import { login, create } from "./helpers";
import type { CanvasDocument, Workspace } from "../src/model";
async function canvasPoint(page: Page, x: number, y: number) {
  const box = await page.getByTestId("canvas").boundingBox();
  if (!box) throw new Error("Canvas missing");
  return { x: box.x + x, y: box.y + y };
}
async function clickCanvas(page: Page, x: number, y: number) {
  const p = await canvasPoint(page, x, y);
  await page.mouse.click(p.x, p.y);
}
async function document(page: Page, name: string) {
  const workspaces = await (
    await page.request.get("/api/v1/workspaces")
  ).json();
  const ws = workspaces.find((w: { name: string }) => w.name === name);
  const pages = await (
    await page.request.get(`/api/v1/workspaces/${ws.id}/pages`)
  ).json();
  return (
    await (
      await page.request.get(`/api/v1/pages/${pages[0].id}/document`)
    ).json()
  ).document as CanvasDocument;
}
test("compose, transform, undo, and persist a private canvas", async ({
  page,
}, info) => {
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  page.on("dialog", (d) => d.accept());
  await login(page);
  const name = `Ideas ${info.project.name} ${Date.now()}`;
  await create(page, name);
  await selectTool(page, "Rectangle");
  await clickCanvas(page, 480, 300);
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  await selectTool(page, "Circle");
  await clickCanvas(page, 760, 370);
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  await page.getByRole("button", { name: "Text (T)", exact: true }).click();
  await clickCanvas(page, 540, 500);
  await page.getByLabel("Edit canvas text").fill("A thought worth keeping.");
  await page.getByLabel("Edit canvas text").press("Control+Enter");
  await page.getByRole("button", { name: "Pen (P)", exact: true }).click();
  const from = await canvasPoint(page, 800, 500);
  await page.mouse.move(from.x, from.y);
  await page.mouse.down();
  for (let i = 1; i <= 12; i++)
    await page.mouse.move(from.x + i * 9, from.y + Math.sin(i / 2) * 30);
  await page.mouse.up();
  await expect(page.getByTestId("element-count")).toHaveText("4 elements");
  await page.getByRole("button", { name: "Undo", exact: true }).click();
  await expect(page.getByTestId("element-count")).toHaveText("3 elements");
  await page.getByRole("button", { name: "Redo", exact: true }).click();
  await expect(page.getByTestId("element-count")).toHaveText("4 elements");
  await page.getByRole("button", { name: "Select (V)", exact: true }).click();
  const start = await canvasPoint(page, 350, 180);
  await page.mouse.move(start.x, start.y);
  await page.mouse.down();
  for (const [x, y] of [
    [950, 180],
    [950, 600],
    [350, 600],
    [350, 180],
  ]) {
    const p = await canvasPoint(page, x, y);
    await page.mouse.move(p.x, p.y, { steps: 5 });
  }
  await page.mouse.up();
  await expect(page.getByText("4 selected", { exact: true })).toBeVisible();
  await page
    .getByRole("button", { name: "Selection properties", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Group selection", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Close settings", exact: true })
    .click();
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const saved = await document(page, name);
  expect(saved.elements).toHaveLength(4);
  expect(new Set(saved.elements.map((e) => e.groupId)).size).toBe(1);
  expect(saved.elements.find((e) => e.type === "text")?.text).toBe(
    "A thought worth keeping.",
  );
  await page.screenshot({
    path: `../.cache/e2e/editor-${info.project.name}.png`,
  });
  await page
    .getByRole("button", { name: "Back to workspaces", exact: true })
    .click();
  await page.getByRole("button", { name: `Open ${name}`, exact: true }).click();
  await expect(page.getByTestId("element-count")).toHaveText("4 elements");
  expect(errors).toEqual([]);
});
test("takeover fences an offline editor and recovers local changes", async ({
  page,
  browser,
}, info) => {
  let cancelUnload = false;
  page.on("dialog", (d) =>
    d.type() === "beforeunload" && cancelUnload ? d.dismiss() : d.accept(),
  );
  await login(page);
  const name = `Recovery ${info.project.name} ${Date.now()}`;
  await create(page, name);
  await selectTool(page, "Rectangle");
  await clickCanvas(page, 450, 350);
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const viewerContext = await browser.newContext({
    viewport: { width: 1440, height: 1000 },
  });
  const viewer = await viewerContext.newPage();
  viewer.on("dialog", (d) => d.accept());
  await login(viewer);
  await viewer
    .getByRole("button", { name: `Open ${name}`, exact: true })
    .click();
  await expect(
    viewer.getByRole("button", { name: "Take control", exact: true }),
  ).toBeVisible();
  await page.context().setOffline(true);
  await selectTool(page, "Star");
  await clickCanvas(page, 750, 430);
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  cancelUnload = true;
  const warning = page.waitForEvent("dialog");
  await page.close({ runBeforeUnload: true });
  expect((await warning).type()).toBe("beforeunload");
  expect(page.isClosed()).toBe(false);
  cancelUnload = false;
  await viewer.bringToFront();
  await viewer
    .getByRole("button", { name: "Take control", exact: true })
    .click();
  await expect(
    viewer.getByRole("button", { name: "Take control", exact: true }),
  ).toHaveCount(0, { timeout: 15000 });
  await selectTool(viewer, "Circle");
  await clickCanvas(viewer, 650, 350);
  await expect(viewer.getByTestId("save-status")).toHaveText("Saved");
  await page.context().setOffline(false);
  await expect(
    page.getByRole("button", { name: "Save as a separate page", exact: true }),
  ).toBeVisible({ timeout: 15000 });
  await page
    .getByRole("button", { name: "Save as a separate page", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Save recovered page", exact: true })
    .click();
  await expect(page.getByRole("tab", { name: /recovered/ })).toBeVisible();
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  const original = await document(viewer, name);
  expect(original.elements.some((e) => e.shape === "circle")).toBe(true);
  expect(original.elements.some((e) => e.shape === "star")).toBe(false);
  await viewerContext.close();
});
test("image placement, independent resize handles, rotation, and variants", async ({
  page,
}, info) => {
  page.on("dialog", (d) => d.accept());
  await login(page);
  const name = `Images ${info.project.name} ${Date.now()}`;
  await create(page, name);
  // Generate a reasonably large image with the browser's standard canvas API.
  const encoded = await page.evaluate(() => {
    const canvas = document.createElement("canvas");
    canvas.width = 1200;
    canvas.height = 800;
    const context = canvas.getContext("2d")!;
    context.fillStyle = "#dcd5f3";
    context.fillRect(0, 0, 1200, 800);
    context.fillStyle = "#8c80bd";
    context.fillRect(100, 100, 500, 300);
    return canvas.toDataURL("image/png").split(",")[1];
  });
  await page.locator("input[type=file]").setInputFiles({
    name: "inspiration.png",
    mimeType: "image/png",
    buffer: Buffer.from(encoded, "base64"),
  });
  await expect(
    page.getByText("Drag to size image", { exact: false }),
  ).toBeVisible();
  await expect(page.getByTestId("element-count")).toHaveText("0 elements");
  await clickCanvas(page, 650, 370);
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const before = (await document(page, name)).elements[0];
  expect(before.width / before.height).toBeCloseTo(1.5, 2);
  const center = await canvasPoint(page, 650, 370); // The corner is at width/2, height/2 plus the 4px selection padding.
  await page.mouse.move(
    center.x + before.width / 2 + 4,
    center.y + before.height / 2 + 4,
  );
  await page.mouse.down();
  await page.mouse.move(
    center.x + before.width / 2 + 75,
    center.y + before.height / 2 + 55,
    { steps: 8 },
  );
  await page.mouse.up();
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const corner = (await document(page, name)).elements[0];
  expect(corner.width).toBeGreaterThan(before.width);
  expect(corner.width / corner.height).toBeCloseTo(
    before.width / before.height,
    1,
  );
  // Side handle changes width independently. Its center shifts by half of the resize delta.
  const left = center.x - before.width / 2;
  const top = center.y - before.height / 2;
  const right = left + corner.width;
  const middleY = top + corner.height / 2;
  await page.mouse.move(right + 4, middleY);
  await page.mouse.down();
  await page.mouse.move(right + 74, middleY, { steps: 8 });
  await page.mouse.up();
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const side = (await document(page, name)).elements[0];
  expect(side.width).toBeGreaterThan(corner.width + 30);
  expect(side.height).toBeCloseTo(corner.height, 0);
  // Rotation handle sits 25px above the top-center, adjusted for Konva’s padding convention.
  const midX = center.x + (side.x - before.x);
  const currentTop = center.y + (side.y - before.y) - side.height / 2;
  await page.mouse.move(midX, currentTop - 21);
  await page.mouse.down();
  await page.mouse.move(midX + 90, currentTop + 15, { steps: 8 });
  await page.mouse.up();
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const rotated = (await document(page, name)).elements[0];
  expect(Math.abs(rotated.rotation)).toBeGreaterThan(0.1);
  await expect
    .poll(
      async () => {
        const meta = await (
          await page.request.get(`/api/v1/assets/${before.assetId}/meta`)
        ).json();
        return meta.variants.length;
      },
      { timeout: 15000 },
    )
    .toBeGreaterThan(0);
  const response = await page.request.get(`/api/v1/assets/${before.assetId}`);
  expect(response.ok()).toBe(true);
  expect(response.headers()["cache-control"]).toBe("no-store");
  expect(await response.body()).toEqual(Buffer.from(encoded, "base64"));
  await page.evaluate((data) => {
    const bytes = Uint8Array.from(atob(data), (c) => c.charCodeAt(0));
    const transfer = new DataTransfer();
    transfer.items.add(new File([bytes], "pasted.png", { type: "image/png" }));
    const event = new ClipboardEvent("paste", {
      bubbles: true,
      clipboardData: transfer,
    });
    // Supply explicit data for this synthetic paste fixture; it does not
    // access the operating system clipboard.
    Object.defineProperty(event, "clipboardData", { value: transfer });
    document.body.dispatchEvent(event);
  }, encoded);
  await expect(
    page.getByText("Drag to size image", { exact: false }),
  ).toBeVisible();
  await clickCanvas(page, 300, 450);
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  await page.evaluate((data) => {
    const bytes = Uint8Array.from(atob(data), (c) => c.charCodeAt(0));
    const transfer = new DataTransfer();
    transfer.items.add(new File([bytes], "dropped.png", { type: "image/png" }));
    document.querySelector('[data-testid="canvas"]')!.dispatchEvent(
      new DragEvent("drop", {
        bubbles: true,
        cancelable: true,
        dataTransfer: transfer,
      }),
    );
  }, encoded);
  await expect(
    page.getByText("Drag to size image", { exact: false }),
  ).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(
    page.getByText("Drag to size image", { exact: false }),
  ).toHaveCount(0);
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const source = (await document(page, name)).elements[1];
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  await clickCanvas(page, 300, 450);
  await page
    .getByRole("button", { name: "Selection properties", exact: true })
    .click();
  await page.getByRole("button", { name: "Crop image", exact: true }).click();
  await expect(page.getByTestId("crop-selection")).toBeVisible();
  const preview = await page.getByTestId("crop-preview").boundingBox();
  if (!preview) throw new Error("Crop preview missing");
  await page.mouse.move(
    preview.x + preview.width * 0.2,
    preview.y + preview.height * 0.2,
  );
  await page.mouse.down();
  await page.mouse.move(
    preview.x + preview.width * 0.8,
    preview.y + preview.height * 0.8,
    { steps: 8 },
  );
  await page.mouse.up();
  await expect(page.getByLabel("Crop left (%)", { exact: true })).toHaveValue(
    "20",
  );
  await page.getByLabel("Crop left (%)", { exact: true }).fill("25");
  await page.getByLabel("Crop top (%)", { exact: true }).fill("10");
  await page.getByLabel("Crop right (%)", { exact: true }).fill("25");
  await page.getByLabel("Crop bottom (%)", { exact: true }).fill("10");
  await page.getByRole("button", { name: "Apply crop", exact: true }).click();
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const cropped = (await document(page, name)).elements[1];
  expect(cropped.crop).toEqual({ x: 0.25, y: 0.1, width: 0.5, height: 0.8 });
  expect(cropped.width).toBeCloseTo(source.width / 2);
  expect(cropped.height).toBeCloseTo(source.height * 0.8);
  await page.getByRole("button", { name: "Undo", exact: true }).click();
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  expect((await document(page, name)).elements[1].crop).toBeUndefined();
  await page.getByRole("button", { name: "Redo", exact: true }).click();
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  await page.getByRole("button", { name: "Crop image", exact: true }).click();
  await page.getByRole("button", { name: "Reset crop", exact: true }).click();
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  expect((await document(page, name)).elements[1].crop).toEqual(cropped.crop);
  await page
    .getByRole("button", { name: "Back to workspaces", exact: true })
    .click();
  await page.getByRole("button", { name: `Open ${name}`, exact: true }).click();
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  await expect(async () => {
    await clickCanvas(page, 300, 450);
    await expect(
      page.getByRole("button", { name: "Selection properties", exact: true }),
    ).toBeVisible({ timeout: 500 });
  }).toPass({ timeout: 10000 });
  await page
    .getByRole("button", { name: "Selection properties", exact: true })
    .click();
  await page.getByRole("button", { name: "Crop image", exact: true }).click();
  await expect(page.getByLabel("Crop left (%)", { exact: true })).toHaveValue(
    "25",
  );
  await page.screenshot({
    path: `../.cache/e2e/crop-${info.project.name}.png`,
  });
  await page.getByRole("button", { name: "Reset crop", exact: true }).click();
  await page.getByRole("button", { name: "Apply crop", exact: true }).click();
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const restored = (await document(page, name)).elements[1];
  expect(restored.width).toBeCloseTo(source.width);
  expect(restored.height).toBeCloseTo(source.height);
  expect(restored.assetId).toBe(source.assetId);
});
test("move cursors, whole font sizes and vertical text alignment", async ({
  page,
}, info) => {
  await login(page);
  const name = `Text controls ${info.project.name} ${Date.now()}`;
  await create(page, name);
  await page.getByRole("button", { name: "Text (T)", exact: true }).click();
  await clickCanvas(page, 450, 300);
  await page.getByLabel("Edit canvas text").fill("Aligned text");
  await page.getByLabel("Edit canvas text").press("Control+Enter");
  const center = await canvasPoint(page, 450, 300);
  await page.mouse.move(center.x + 1, center.y);
  const surface = page.locator('[data-testid="canvas"] .konvajs-content');
  await expect(async () => {
    await page.mouse.move(center.x + 2, center.y);
    await page.mouse.move(center.x + 1, center.y);
    await expect(surface).toHaveCSS("cursor", "grab", { timeout: 500 });
  }).toPass({ timeout: 10000 });
  await page.mouse.down();
  await page.mouse.move(center.x + 8, center.y, { steps: 3 });
  await expect(surface).toHaveCSS("cursor", "grabbing");
  await page.mouse.up();
  await expect(surface).toHaveCSS("cursor", "grab");
  // Undo just the move so the text box has known coordinates for the render check.
  await page.getByRole("button", { name: "Undo", exact: true }).click();
  await page
    .getByRole("button", { name: "Selection properties", exact: true })
    .click();
  const font = page.getByLabel("Font size", { exact: true });
  await expect(font).toHaveAttribute("step", "1");
  await font.fill("32");
  await font.fill("32.5");
  await expect(font).toHaveValue("32");
  await page
    .getByLabel("Vertical alignment", { exact: true })
    .selectOption("bottom");
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const saved = (await document(page, name)).elements[0];
  expect(saved.fontSize).toBe(32);
  expect(saved.verticalAlign).toBe("bottom");
  // Text pixels must be in the lower half of the box, not merely stored as an option.
  await expect
    .poll(() =>
      page
        .locator('[data-testid="canvas"] canvas')
        .first()
        .evaluate((canvas) => {
          const node = canvas as HTMLCanvasElement;
          const scale = node.width / node.getBoundingClientRect().width;
          const pixels = node
            .getContext("2d")!
            .getImageData(330 * scale, 250 * scale, 240 * scale, 100 * scale);
          for (let y = 0; y < pixels.height; y++)
            for (let x = 0; x < pixels.width; x++)
              if (pixels.data[(y * pixels.width + x) * 4 + 3] > 0)
                return y / scale;
          return -1;
        }),
    )
    .toBeGreaterThan(50);
  await page.getByRole("button", { name: "Edit text", exact: true }).click();
  await expect
    .poll(async () =>
      parseFloat(
        await page
          .getByLabel("Edit canvas text")
          .evaluate((node) => getComputedStyle(node).paddingTop),
      ),
    )
    .toBeGreaterThan(50);
  await page.getByLabel("Edit canvas text").press("Control+Enter");
  await expect(
    page.getByLabel("Object properties", { exact: true }),
  ).toHaveCount(0);
  await page.mouse.move(center.x + 124, center.y + 54);
  await page.mouse.down();
  await page.mouse.move(center.x + 161, center.y + 69, { steps: 8 });
  await page.mouse.up();
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const resized = (await document(page, name)).elements[0];
  expect(resized.fontSize).toBeGreaterThan(32);
  expect(Number.isInteger(resized.fontSize)).toBe(true);
  await page
    .getByRole("button", { name: "Back to workspaces", exact: true })
    .click();
  await page.getByRole("button", { name: `Open ${name}`, exact: true }).click();
  await page.getByRole("button", { name: "Select (V)", exact: true }).click();
  await page.getByRole("button", { name: "Pen (P)", exact: true }).click();
  await page.mouse.move(center.x + 1, center.y);
  await expect(surface).toHaveCSS("cursor", "crosshair");
});

test("large page remains interactive with 3000 elements and image placements", async ({
  page,
}, info) => {
  await login(page);
  const name = `Large canvas ${info.project.name} ${Date.now()}`;
  await create(page, name);
  const all = await (await page.request.get("/api/v1/workspaces")).json();
  const ws = all.find((w: Workspace) => w.name === name);
  const pages = await (
    await page.request.get(`/api/v1/workspaces/${ws.id}/pages`)
  ).json();
  const me = await (await page.request.get("/api/v1/auth/me")).json();
  const headers = { "X-CSRF-Token": me.csrf };
  const clientId = "performance-fixture";
  let lease = await (
    await page.request.post(`/api/v1/pages/${pages[0].id}/lease`, {
      headers,
      data: { clientId, action: "takeover" },
    })
  ).json();
  if (lease.request) {
    await page.getByRole("button", { name: /^Accept \(\d+\)$/ }).click();
    lease = await (
      await page.request.get(
        `/api/v1/pages/${pages[0].id}/lease?clientId=${clientId}`,
      )
    ).json();
  }
  const encoded = await page.evaluate(() => {
    const c = document.createElement("canvas");
    c.width = 600;
    c.height = 400;
    const ctx = c.getContext("2d")!;
    ctx.fillStyle = "#e2dcef";
    ctx.fillRect(0, 0, 600, 400);
    return c.toDataURL().split(",")[1];
  });
  const asset = await (
    await page.request.post("/api/v1/assets", {
      headers,
      multipart: {
        file: {
          name: "fixture.png",
          mimeType: "image/png",
          buffer: Buffer.from(encoded, "base64"),
        },
      },
    })
  ).json();
  const elements = Array.from({ length: 3000 }, (_, i) => ({
    id: `fixture-${i}`,
    type:
      i < 200
        ? "image"
        : i % 7 === 0
          ? "text"
          : i % 11 === 0
            ? "stroke"
            : "shape",
    x: (i % 60) * 150 - 4500,
    y: Math.floor(i / 60) * 110 - 2750,
    width: 110,
    height: 70,
    rotation: (i % 8) * 0.03,
    ...(i < 200
      ? { assetId: asset.id }
      : i % 7 === 0
        ? {
            text: `Thought ${i}`,
            fontSize: 16,
            align: "left",
            color: "#716593",
          }
        : i % 11 === 0
          ? {
              points: [0, 0, 0.3, 0.7, 0.6, 0.2, 1, 1],
              penWidth: 2,
              color: "#716593",
            }
          : {
              shape: i % 2 ? "circle" : "rectangle",
              fill: "#e2dcef",
              border: "#ada0cb",
              borderWidth: 1,
            }),
  }));
  const response = await page.request.put(
    `/api/v1/pages/${pages[0].id}/document`,
    {
      headers,
      data: {
        baseRevision: 0,
        epoch: lease.epoch,
        clientId,
        mutationId: `fixture-${Date.now()}`,
        document: {
          schemaVersion: 1,
          elements,
          settings: { background: "dots", gridSize: 20, snap: false },
        },
      },
    },
  );
  expect(response.ok()).toBe(true);
  await page.request.post(`/api/v1/pages/${pages[0].id}/lease`, {
    headers,
    data: { clientId, epoch: lease.epoch, action: "release" },
  });
  page.on("dialog", (d) => d.accept());
  await expect(
    page.getByRole("button", { name: "Take control", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Take control", exact: true }).click();
  await expect(page.getByTestId("element-count")).toHaveText("3000 elements");
  const imageReady = page.waitForResponse(
    (response) =>
      response.url().includes(`/assets/${asset.id}`) &&
      response.request().resourceType() === "image",
  );
  await page.getByRole("button", { name: "Fit content", exact: true }).click();
  await (await imageReady).finished();
  // Exclude initial transfer/decoding: the design's frame target is for loaded content.
  await page.evaluate(
    () =>
      new Promise<void>((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
      ),
  );
  // Measure real animation frames during user input; one image source is reused
  // at 200 different placements, so this is not a 200-unique-originals memory test.
  await page.evaluate(() => {
    (window as Window & { __frames: number[]; __measure: boolean }).__frames =
      [];
    let previous = performance.now();
    (window as Window & { __frames: number[]; __measure: boolean }).__measure =
      true;
    const tick = (time: number) => {
      if (
        !(window as Window & { __frames: number[]; __measure: boolean })
          .__measure
      )
        return;
      (
        window as Window & { __frames: number[]; __measure: boolean }
      ).__frames.push(time - previous);
      previous = time;
      requestAnimationFrame(tick);
    };
    requestAnimationFrame(tick);
  });
  const p = await canvasPoint(page, 650, 400);
  await page.keyboard.down("Control");
  await page.mouse.move(p.x, p.y);
  await page.mouse.down();
  for (let i = 0; i < 45; i++)
    await page.mouse.move(p.x + i * 3, p.y + Math.sin(i / 5) * 30);
  await page.mouse.up();
  await page.keyboard.up("Control");
  const frames = await page.evaluate(() => {
    (window as Window & { __frames: number[]; __measure: boolean }).__measure =
      false;
    return (window as Window & { __frames: number[]; __measure: boolean })
      .__frames as number[];
  });
  const sorted = frames.slice(1).sort((a, b) => a - b);
  const median = sorted[Math.floor(sorted.length * 0.5)],
    p95 = sorted[Math.floor(sorted.length * 0.95)];
  console.log(
    `${info.project.name}: 3000 elements / 200 image placements; frame median=${median?.toFixed(1)}ms, p95=${p95?.toFixed(1)}ms`,
  );
  expect(sorted.length).toBeGreaterThan(5);
  expect(p95).toBeLessThan(250);
});
