import { expect, test } from "@playwright/test";
import { activate, create, login, selectTool } from "./helpers";
import { TabletInput } from "./tablet-input";
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
