import { selectTool } from "./helpers";
import { expect, test } from "@playwright/test";
import { create, login } from "./helpers";
import type { CanvasDocument } from "../src/model";

test("opaque object settings show real colors and canvas creation defaults persist independently", async ({
  page,
}, info) => {
  page.on("dialog", (dialog) => dialog.accept());
  await page.emulateMedia({ colorScheme: "light" });
  await login(page);
  await create(page, `Canvas colors ${info.project.name} ${Date.now()}`);
  const box = (await page.getByTestId("canvas").boundingBox())!;
  const pageID = page.url().split("/pages/")[1];
  const workspaceID = page.url().split("/workspaces/")[1].split("/pages/")[0];
  const saved = async () => {
    await expect(page.getByTestId("save-status")).toHaveText("Saved");
    return (
      await (await page.request.get(`/api/v1/pages/${pageID}/document`)).json()
    ).document as CanvasDocument;
  };
  const add = async (tool: string, x: number, y: number) => {
    await selectTool(page, tool);
    await page.mouse.click(box.x + x, box.y + y);
  };
  const text = async (value: string, x: number, y: number) => {
    await add("Text (T)", x, y);
    await page.getByLabel("Edit canvas text").fill(value);
    await page.getByLabel("Edit canvas text").press("Control+Enter");
  };
  const picker = (label: string) =>
    page.getByLabel(label, { exact: true }).locator("..");
  const properties = page.locator(".object-inspector");
  await add("Rectangle", 450, 300);
  let document = await saved();
  expect(document.elements[0].fill).toBe("#1b5e3b");
  expect(document.elements[0].border).toBe("#d9a441");
  await page.mouse.click(box.x + 450, box.y + 300, { button: "right" });
  await expect(properties).toBeVisible();
  await expect(properties).toHaveCSS("background-color", "rgb(255, 255, 255)");
  await expect(picker("Shape fill").locator(".color-picker-color")).toHaveCSS(
    "background-color",
    "rgb(27, 94, 59)",
  );
  await expect(
    picker("Element color").locator(".color-picker-color"),
  ).toHaveCSS("background-color", "rgb(217, 164, 65)");
  await page.getByLabel("Shape fill", { exact: true }).fill("#112233");
  await expect(picker("Shape fill").locator(".color-picker-color")).toHaveCSS(
    "background-color",
    "rgb(17, 34, 51)",
  );
  await page.screenshot({
    path: `../.cache/e2e/object-settings-${info.project.name}-light.png`,
  });
  await properties
    .getByRole("button", { name: "Close settings", exact: true })
    .click();
  await add("Circle", 650, 300);
  await text("Golden text", 850, 300);
  document = await saved();
  expect(document.elements[1].fill).toBe("#1b5e3b");
  expect(document.elements[2].color).toBe("#d9a441");
  await page
    .getByRole("button", { name: "Canvas settings", exact: true })
    .click();
  await page.getByLabel("Default shape fill", { exact: true }).fill("#abcdef");
  await page
    .getByLabel("Default shape border", { exact: true })
    .fill("#bc9012");
  await page.getByLabel("Default text color", { exact: true }).fill("#123abc");
  await page
    .getByLabel("Canvas settings panel")
    .getByRole("button", { name: "Close settings", exact: true })
    .click();
  await add("Rectangle", 450, 500);
  await text("Custom defaults", 850, 500);
  document = await saved();
  expect(document.settings).toMatchObject({
    defaultFill: "#abcdef",
    defaultBorder: "#bc9012",
    defaultTextColor: "#123abc",
  });
  expect(document.elements[0].fill).toBe("#112233");
  expect(document.elements[1].fill).toBe("#1b5e3b");
  expect(document.elements[2].color).toBe("#d9a441");
  expect(document.elements[3]).toMatchObject({
    fill: "#abcdef",
    border: "#bc9012",
  });
  expect(document.elements[4].color).toBe("#123abc");
  await page.reload();
  await expect(page.getByTestId("canvas")).toBeVisible();
  await page
    .getByRole("button", { name: "Canvas settings", exact: true })
    .click();
  await expect(
    page.getByLabel("Default shape fill", { exact: true }),
  ).toHaveValue("#abcdef");
  await expect(
    page.getByLabel("Default shape border", { exact: true }),
  ).toHaveValue("#bc9012");
  await expect(
    page.getByLabel("Default text color", { exact: true }),
  ).toHaveValue("#123abc");
  await expect(
    page.getByLabel("Default shape fill", { exact: true }),
  ).toBeDisabled();
  // Existing imported colors can include alpha; the native RGB input alone cannot show them.
  const me = await (await page.request.get("/api/v1/auth/me")).json();
  const alpha: CanvasDocument = {
    ...document,
    elements: [
      {
        id: "alpha-shape",
        type: "shape",
        shape: "rectangle",
        x: 0,
        y: 0,
        width: 200,
        height: 100,
        rotation: 0,
        fill: "#12345680",
        border: "#d9a441",
        borderWidth: 2,
      },
    ],
  };
  const response = await page.request.post("/api/v1/pages/recovery-copy", {
    headers: { "X-CSRF-Token": me.csrf },
    data: { workspaceId: workspaceID, name: "Alpha colors", document: alpha },
  });
  expect(response.ok(), await response.text()).toBe(true);
  const result = await response.json();
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto(`/#/workspaces/${workspaceID}/pages/${result.id}`);
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  // Konva batches its visible and hit-canvas updates on the next frame.
  await page.evaluate(
    () =>
      new Promise<void>((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
      ),
  );
  const alphaBox = (await page.getByTestId("canvas").boundingBox())!;
  await page.mouse.click(
    alphaBox.x + alphaBox.width / 2,
    alphaBox.y + alphaBox.height / 2,
    { button: "right" },
  );
  await expect(properties).toHaveCSS("background-color", "rgb(34, 35, 40)");
  await expect(page.getByLabel("Shape fill", { exact: true })).toHaveValue(
    "#123456",
  );
  await expect(picker("Shape fill").locator(".color-picker-value")).toHaveText(
    "#12345680",
  );
  await expect(picker("Shape fill").locator(".color-picker-color")).toHaveCSS(
    "background-color",
    /rgba\(18, 52, 86, 0\.5/,
  );
  await page.screenshot({
    path: `../.cache/e2e/object-settings-${info.project.name}-dark.png`,
  });
  await properties
    .getByRole("button", { name: "Remove fill", exact: true })
    .click();
  await expect(picker("Shape fill").locator(".color-picker-value")).toHaveText(
    "Transparent",
  );
  await expect(picker("Shape fill").locator(".color-picker-color")).toHaveCSS(
    "background-color",
    "rgba(0, 0, 0, 0)",
  );
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
});
