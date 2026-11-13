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
