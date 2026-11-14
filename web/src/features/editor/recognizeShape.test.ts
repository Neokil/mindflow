import { expect, it } from "vitest";
import { InkStroke } from "./InkStroke";
import { recognizeShape } from "./recognizeShape";
import type { Point } from "../../model";
const circle = Array.from({ length: 81 }, (_, i) => ({
  x: 200 + 100 * Math.cos((i / 80) * Math.PI * 2) + ((i % 3) - 1) * 1.2,
  y: 150 + 65 * Math.sin((i / 80) * Math.PI * 2) + ((i % 4) - 1.5),
}));
function rectangle(angle = 0) {
  const corners = [
    [-100, -60],
    [100, -60],
    [100, 60],
    [-100, 60],
    [-100, -60],
  ];
  return corners
    .slice(0, -1)
    .flatMap(([x, y], edge) =>
      Array.from({ length: 20 }, (_, i) => {
        const next = corners[edge + 1],
          fraction = i / 20,
          px = x + (next[0] - x) * fraction,
          py = y + (next[1] - y) * fraction;
        return {
          x: 300 + px * Math.cos(angle) - py * Math.sin(angle) + ((i % 3) - 1),
          y:
            200 +
            px * Math.sin(angle) +
            py * Math.cos(angle) +
            (i % 2 ? 0.5 : -0.5),
        };
      }),
    )
    .concat([
      {
        x: 300 - 100 * Math.cos(angle) + 60 * Math.sin(angle),
        y: 200 - 100 * Math.sin(angle) - 60 * Math.cos(angle),
      },
    ]);
}
it("recognizes imperfect closed circles and rotated rectangles in either direction", () => {
  expect(recognizeShape(circle, 1)?.kind).toBe("circle");
  for (const angle of [0, Math.PI / 7, Math.PI / 4])
    for (const points of [rectangle(angle), rectangle(angle).reverse()]) {
      const shape = recognizeShape(points, 1);
      expect(shape?.kind).toBe("rectangle");
      expect(shape!.pointAt(0).x).toBeCloseTo(shape!.pointAt(1).x);
      expect(shape!.pointAt(0).y).toBeCloseTo(shape!.pointAt(1).y);
    }
});
it("leaves open arcs, small loops, and self-crossing strokes unrecognized", () => {
  expect(recognizeShape(circle.slice(0, 55), 1)).toBeUndefined();
  expect(
    recognizeShape(
      circle.map((p) => ({ x: p.x / 20, y: p.y / 20 })),
      1,
    ),
  ).toBeUndefined();
  expect(
    recognizeShape(
      [
        { x: 0, y: 0 },
        { x: 100, y: 100 },
        { x: 0, y: 100 },
        { x: 100, y: 0 },
        { x: 0, y: 0 },
      ],
      1,
    ),
  ).toBeUndefined();
});
it("retains pressure, closure, color and marker style while hold-resizing a fitted outline", () => {
  for (const marker of [false, true]) {
    const ink = new InkStroke(circle[0], !marker, "light", 1, 0.1);
    circle
      .slice(1)
      .forEach((point, i) =>
        ink.add(point, 0.1 + (0.8 * i) / (circle.length - 2)),
      );
    ink.optimize(circle.at(-1)!);
    expect(ink.shapeKind).toBe("circle");
    expect(ink.optimized).toBe(true);
    const before = ink.element("circle", "#d9a441", 8, marker),
      center = { x: before.x, y: before.y };
    const end = circle.at(-1)!,
      resized = {
        x: center.x + (end.x - center.x) * 1.5,
        y: center.y + (end.y - center.y) * 1.5,
      };
    ink.add(resized, 0.7);
    ink.finish(resized);
    const after = ink.element("circle", "#d9a441", 8, marker);
    expect(after.width / before.width).toBeCloseTo(1.5, 1);
    expect(after.color).toBe("#d9a441");
    expect(after.points!.slice(0, 2)).toEqual(after.points!.slice(-2));
    if (marker) {
      expect(after.highlight).toBe(true);
      expect(after.pressures).toBeUndefined();
    } else {
      expect(after.pressures!.length).toBe(after.points!.length / 2);
      expect(Math.max(...after.pressures!)).toBeGreaterThan(0.85);
      expect(after.pressures!.at(-1)).toBe(0.7);
    }
  }
});
it("keeps the existing line hold behavior for open strokes", () => {
  const points: Point[] = [
    { x: 0, y: 0 },
    { x: 30, y: 10 },
    { x: 60, y: -5 },
    { x: 100, y: 0 },
  ];
  const ink = new InkStroke(points[0], false, "off", 1, 0.5);
  points.slice(1).forEach((p) => ink.add(p, 0.5));
  ink.optimize(points.at(-1)!);
  expect(ink.straight).toBe(true);
  expect(ink.samples).toHaveLength(2);
});
