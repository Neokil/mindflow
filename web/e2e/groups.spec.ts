import { selectTool } from "./helpers";
import { test, expect } from "@playwright/test";
import { create, login } from "./helpers";

test("locked groups resist movement, editing and deletion until unlocked", async ({
  page,
}) => {
  page.on("dialog", (dialog) => dialog.accept());
  await login(page);
  await create(page, `Locked groups ${Date.now()}`);
  const box = (await page.getByTestId("canvas").boundingBox())!;
  await selectTool(page, "Rectangle");
  await page.mouse.click(box.x + 450, box.y + 300);
  await page.getByRole("button", { name: "Text (T)", exact: true }).click();
  await page.mouse.click(box.x + 700, box.y + 300);
  await page.getByLabel("Edit canvas text").fill("A fixed heading");
  await page.getByLabel("Edit canvas text").press("Control+Enter");
  await page.keyboard.press("Control+a");
  await page.keyboard.press("Control+g");
  await page.getByRole("button", { name: "Lock groups", exact: true }).click();
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  const pageID = page.url().split("/pages/")[1];
  const snapshot = async () =>
    (await (await page.request.get(`/api/v1/pages/${pageID}/document`)).json())
      .document;
  const locked = await snapshot();
  expect(
    locked.elements.every((element: { locked: boolean }) => element.locked),
  ).toBe(true);
  await page.mouse.move(box.x + 450, box.y + 300);
  await page.mouse.down();
  await page.mouse.move(box.x + 530, box.y + 370, { steps: 8 });
  await page.mouse.up();
  await page.keyboard.press("Delete");
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  await page.mouse.dblclick(box.x + 700, box.y + 300);
  await expect(page.getByLabel("Edit canvas text")).toHaveCount(0);
  await page
    .getByRole("button", { name: "Selection properties", exact: true })
    .click();
  await expect(
    page.getByLabel("Element color", { exact: true }),
  ).toBeDisabled();
  await page
    .getByRole("button", { name: "Close settings", exact: true })
    .click();
  expect(await snapshot()).toEqual(locked);
  await page.reload();
  await expect(page.getByTestId("canvas")).toBeVisible();
  await page
    .getByRole("button", { name: "Force control", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Shapes", exact: true }),
  ).toBeEnabled();
  await page.keyboard.press("Control+a");
  await page.keyboard.press("Delete");
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  await page
    .getByRole("button", { name: "Unlock groups", exact: true })
    .click();
  await page.keyboard.press("Delete");
  await expect(page.getByTestId("element-count")).toHaveText("0 elements");
  await page.keyboard.press("Control+z");
  await expect(page.getByTestId("element-count")).toHaveText("2 elements");
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  expect(
    (await snapshot()).elements.every(
      (element: { locked?: boolean }) => !element.locked,
    ),
  ).toBe(true);
});
