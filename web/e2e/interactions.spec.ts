import { selectTool } from "./helpers";
import { test, expect } from "@playwright/test";
import { login, create } from "./helpers";
import type { CanvasDocument } from "../src/model";

test("drag creation, separate inspectors, middle pan and profile avatars", async ({
  page,
}, info) => {
  await login(page);
  await page.getByRole("button", { name: "Your profile", exact: true }).click();
  const encoded = await page.evaluate(() => {
    const canvas = document.createElement("canvas");
    canvas.width = 32;
    canvas.height = 32;
    const context = canvas.getContext("2d")!;
    context.fillStyle = "#5963dc";
    context.fillRect(0, 0, 32, 32);
    return canvas.toDataURL("image/png").split(",")[1];
  });
  await page.getByLabel("Upload avatar", { exact: true }).setInputFiles({
    name: "avatar.png",
    mimeType: "image/png",
    buffer: Buffer.from(encoded, "base64"),
  });
  await expect(page.locator(".profile-identity .avatar img")).toBeVisible();
  await page.getByRole("button", { name: "Close dialog", exact: true }).click();
  await expect(page.locator(".account-profile img")).toBeVisible();
  const name = `Gestures ${info.project.name} ${Date.now()}`;
  await create(page, name);
  const box = (await page.getByTestId("canvas").boundingBox())!;
  const from = { x: box.x + 700, y: box.y + 450 };
  await selectTool(page, "Rectangle");
  await page.mouse.move(from.x, from.y);
  await page.mouse.down();
  await page.evaluate(
    () =>
      new Promise<void>((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
      ),
  );
  const flash = await page
    .locator('[data-testid="canvas"] canvas')
    .last()
    .evaluate((canvas) => {
      const element = canvas as HTMLCanvasElement;
      const scale = element.width / element.getBoundingClientRect().width;
      return element
        .getContext("2d")!
        .getImageData(640 * scale, 410 * scale, 1, 1).data[3];
    });
  expect(flash).toBe(0);
  await page.mouse.move(from.x - 200, from.y - 100, { steps: 8 });
  await expect(page.getByTestId("element-count")).toHaveText("0 elements");
  await page.screenshot({
    path: `../.cache/e2e/creation-preview-${info.project.name}.png`,
  });
  await page.mouse.up();
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  await page
    .getByRole("button", { name: "Canvas settings", exact: true })
    .click();
  await expect(page.getByLabel("Grid spacing", { exact: true })).toBeVisible();
  await expect(page.getByLabel("Element color", { exact: true })).toHaveCount(
    0,
  );
  await page
    .getByRole("button", { name: "Selection properties", exact: true })
    .click();
  await expect(page.getByLabel("Element color", { exact: true })).toBeVisible();
  await expect(page.getByLabel("Grid spacing", { exact: true })).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Canvas settings", exact: true }),
  ).toHaveClass(/active/);
  await page
    .getByLabel("Canvas settings panel", { exact: true })
    .getByRole("button", { name: "Close settings", exact: true })
    .click();
  const floating = await page
    .getByLabel("Object properties", { exact: true })
    .boundingBox();
  const properties = await page
    .getByRole("button", { name: "Selection properties", exact: true })
    .boundingBox();
  expect(floating!.y + floating!.height).toBeLessThanOrEqual(properties!.y + 1);
  await page
    .getByRole("button", { name: "Close settings", exact: true })
    .click();
  await page.mouse.click(from.x - 100, from.y - 50, { button: "right" });
  await expect(page.getByLabel("Element color", { exact: true })).toBeVisible();
  const contextPanel = await page
    .getByLabel("Object properties", { exact: true })
    .boundingBox();
  expect(contextPanel!.x).toBeCloseTo(from.x - 100, 0);
  await page
    .getByRole("button", { name: "Close settings", exact: true })
    .click();
  const before = await page
    .getByTestId("canvas")
    .evaluate((el) => getComputedStyle(el).backgroundPosition);
  await page.mouse.move(box.x + 100, box.y + 200);
  await page.mouse.down({ button: "middle" });
  await page.mouse.move(box.x + 180, box.y + 240, { steps: 6 });
  await page.mouse.up({ button: "middle" });
  const after = await page
    .getByTestId("canvas")
    .evaluate((el) => getComputedStyle(el).backgroundPosition);
  expect(after).not.toBe(before);
  await page.getByRole("button", { name: "Text (T)", exact: true }).click();
  await page.mouse.move(box.x + 400, box.y + 500);
  await page.keyboard.down("Shift");
  await page.mouse.down();
  await page.mouse.move(box.x + 560, box.y + 580, { steps: 8 });
  await page.mouse.up();
  await page.keyboard.up("Shift");
  await page.getByLabel("Edit canvas text", { exact: true }).fill("Centered");
  await page
    .getByLabel("Edit canvas text", { exact: true })
    .press("Control+Enter");
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const workspaces = await (
    await page.request.get("/api/v1/workspaces")
  ).json();
  const workspace = workspaces.find(
    (value: { name: string }) => value.name === name,
  );
  const pages = await (
    await page.request.get(`/api/v1/workspaces/${workspace.id}/pages`)
  ).json();
  const data = (await (
    await page.request.get(`/api/v1/pages/${pages[0].id}/document`)
  ).json()) as { document: CanvasDocument };
  const rectangle = data.document.elements[0],
    text = data.document.elements[1];
  expect(rectangle.width).toBeCloseTo(200);
  expect(rectangle.height).toBeCloseTo(100);
  expect(text.width).toBeCloseTo(text.height);
  expect(text.align).toBe("center");
  expect(text.verticalAlign).toBe("middle");
});
