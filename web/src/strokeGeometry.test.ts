import { expect, it } from "vitest";
import { pressureWidth, strokeOutline } from "./strokeGeometry";
it("pressure ribbons have bounded variable widths and round endpoint caps", () => {
  const outline = strokeOutline(
    [
      { x: 0, y: 0, pressure: 0 },
      { x: 50, y: 0, pressure: 0.5 },
      { x: 100, y: 0, pressure: 1 },
    ],
    20,
  );
  expect(pressureWidth(20)).toBe(20);
  expect(pressureWidth(20, 0)).toBe(4);
  expect(pressureWidth(20, 1)).toBe(20);
  expect(outline[1]).toBe(2);
  expect(outline[5]).toBe(10);
  expect(Math.min(...outline.filter((_, i) => i % 2 === 0))).toBeCloseTo(-2);
  expect(Math.max(...outline.filter((_, i) => i % 2 === 0))).toBeCloseTo(110);
  expect(outline.every(Number.isFinite)).toBe(true);
});
it("repeated points, dots, and reversals remain finite", () => {
  for (const coords of [
    [
      [0, 0],
      [0, 0],
    ],
    [
      [0, 0],
      [10, 0],
      [0, 0],
    ],
    [[0, 0]],
  ]) {
    const outline = strokeOutline(
      coords.map(([x, y]) => ({ x, y, pressure: 1 })),
      10,
    );
    expect(outline.length).toBeGreaterThan(4);
    expect(outline.every(Number.isFinite)).toBe(true);
  }
});
