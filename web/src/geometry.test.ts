import { describe, expect, it } from "vitest";
import {
  fullyContained,
  lassoSelection,
  worldPoint,
  zoomAt,
  snap,
} from "./geometry";
import type { CanvasElement } from "./model";
const rect: CanvasElement = {
  id: "one",
  type: "shape",
  shape: "rectangle",
  x: 5,
  y: 5,
  width: 8,
  height: 8,
  rotation: 0,
};
describe("canvas geometry", () => {
  it("keeps the cursor world point fixed while zooming", () => {
    const camera = { x: -30, y: 90, zoom: 2 };
    const p = { x: 121, y: 333 };
    expect(worldPoint(p, zoomAt(p, camera, 1.7))).toEqual(
      worldPoint(p, camera),
    );
  });
  it("requires full containment and handles a concave lasso crossing an element", () => {
    const box = [
      { x: 0, y: 0 },
      { x: 10, y: 0 },
      { x: 10, y: 10 },
      { x: 0, y: 10 },
    ];
    expect(fullyContained(rect, box)).toBe(true);
    expect(fullyContained({ ...rect, rotation: Math.PI / 4 }, box)).toBe(false);
    const notch = [
      { x: 0, y: 0 },
      { x: 10, y: 0 },
      { x: 10, y: 10 },
      { x: 6, y: 10 },
      { x: 6, y: 3 },
      { x: 4, y: 3 },
      { x: 4, y: 10 },
      { x: 0, y: 10 },
    ];
    expect(fullyContained(rect, notch)).toBe(false);
  });
  it("selects a group only when every member is enclosed", () => {
    expect(
      lassoSelection(
        [
          { ...rect, groupId: "g" },
          { ...rect, id: "two", groupId: "g", x: 30 },
        ],
        [
          { x: 0, y: 0 },
          { x: 10, y: 0 },
          { x: 10, y: 10 },
          { x: 0, y: 10 },
        ],
      ),
    ).toEqual([]);
  });
  it("snaps negative coordinates correctly", () => {
    expect(snap(-29, 20)).toBe(-20);
    expect(snap(-31, 20)).toBe(-40);
  });
});
