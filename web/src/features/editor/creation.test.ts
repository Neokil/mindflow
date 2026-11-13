import { describe, expect, it } from "vitest";
import { creationBounds } from "./creation";
const settings = { background: "dots" as const, gridSize: 20, snap: false };
describe("drag creation", () => {
  it("normalizes reverse drags around the element center", () => {
    expect(
      creationBounds({ x: 100, y: 80 }, { x: 20, y: 30 }, false, settings),
    ).toEqual({ x: 60, y: 55, width: 80, height: 50 });
  });
  it("preserves a square and snaps endpoints", () => {
    expect(
      creationBounds({ x: 3, y: 4 }, { x: -72, y: 29 }, true, {
        ...settings,
        snap: true,
      }),
    ).toEqual({ x: -40, y: 40, width: 80, height: 80 });
  });
});
