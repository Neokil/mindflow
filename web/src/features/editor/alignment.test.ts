import { expect, it } from "vitest";
import { alignElements, canAlign } from "./alignment";
import type { CanvasElement } from "../../model";
const element = (
  id: string,
  x: number,
  y: number,
  width = 100,
  height = 60,
): CanvasElement => ({
  id,
  type: "shape",
  shape: "rectangle",
  x,
  y,
  width,
  height,
  rotation: 0,
});
it("aligns edges and centers along either axis without changing other geometry", () => {
  const a = element("a", 0, 0),
    b = element("b", 200, 100, 200, 100),
    source = [a, b];
  expect(alignElements(source, ["a", "b"], "left")[1].x).toBe(50);
  expect(alignElements(source, ["a", "b"], "right")[0].x).toBe(250);
  expect(alignElements(source, ["a", "b"], "center").map((e) => e.x)).toEqual([
    125, 125,
  ]);
  expect(alignElements(source, ["a", "b"], "top")[1].y).toBe(20);
  expect(alignElements(source, ["a", "b"], "bottom")[0].y).toBe(120);
  expect(alignElements(source, ["a", "b"], "middle").map((e) => e.y)).toEqual([
    60, 60,
  ]);
  expect(source[0]).toEqual(a);
});
it("keeps group members together and protects locked selections", () => {
  const a = { ...element("a", 0, 0), groupId: "g" },
    b = { ...element("b", 120, 0), groupId: "g" },
    c = element("c", 400, 100);
  const aligned = alignElements([a, b, c], ["a", "b", "c"], "left");
  expect(aligned[1].x - aligned[0].x).toBe(120);
  expect(aligned[2].x).toBe(0);
  expect(canAlign([a, b])).toBe(false);
  const source = [{ ...a, locked: true }, { ...b, locked: true }, c];
  expect(alignElements(source, ["a", "b", "c"], "top")).toBe(source);
});
it("uses the bounds of rotated objects", () => {
  const a = { ...element("a", 0, 0, 200, 100), rotation: Math.PI / 2 },
    b = element("b", 300, 200);
  expect(alignElements([a, b], ["a", "b"], "top")[1].y).toBeCloseTo(-70);
});
