import { expect, it } from "vitest";
import type { CanvasElement } from "../../model";
import { instantiateTemplate } from "./templates";

it("creates independent unlocked elements while preserving relative placement and image references", () => {
  const source: CanvasElement[] = [
    {
      id: "banner",
      type: "image",
      assetId: "original",
      x: 300,
      y: 100,
      width: 400,
      height: 100,
      rotation: 0,
      groupId: "group",
      locked: true,
      crop: { x: 0, y: 0, width: 1, height: 0.5 },
    },
    {
      id: "title",
      type: "text",
      text: "Heading",
      x: 320,
      y: 110,
      width: 100,
      height: 40,
      rotation: 0,
      groupId: "group",
      locked: true,
    },
  ];
  const first = instantiateTemplate(source, { x: 500, y: 400 });
  const second = instantiateTemplate(source);
  expect(first.map((element) => [element.x, element.y])).toEqual([
    [500, 400],
    [520, 410],
  ]);
  expect(first.every((element) => !element.groupId && !element.locked)).toBe(
    true,
  );
  expect(
    new Set([...source, ...first, ...second].map((element) => element.id)).size,
  ).toBe(6);
  first[0].crop!.height = 1;
  first[1].text = "Changed";
  expect(source[0].crop!.height).toBe(0.5);
  expect(second[1].text).toBe("Heading");
  expect(first[0].assetId).toBe("original");
});
