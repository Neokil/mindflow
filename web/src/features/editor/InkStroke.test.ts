import { expect, it } from "vitest";
import { InkStroke } from "./InkStroke";

it("reduces hand jitter while preserving the actual release position", () => {
  const make = (mode: "off" | "light" | "strong") => {
    const ink = new InkStroke({ x: 0, y: 0 }, false, mode, 1, 0.5);
    for (let i = 1; i <= 80; i++) ink.add({ x: i, y: i % 2 ? 1 : -1 }, 0.5);
    ink.finish({ x: 80, y: 0 });
    return ink;
  };
  const noise = (ink: InkStroke) =>
    ink.samples.reduce((sum, p) => sum + Math.abs(p.y), 0) / ink.samples.length;
  expect(noise(make("strong"))).toBeLessThan(noise(make("light")));
  expect(noise(make("light"))).toBeLessThan(noise(make("off")));
  expect(make("strong").samples.at(-1)).toMatchObject({ x: 80, y: 0 });
});
it("straightens from the raw endpoint and preserves a varying pressure profile", () => {
  const ink = new InkStroke({ x: 0, y: 0 }, true, "strong", 2, 0.1);
  ink.add({ x: 20, y: 30 }, 0.3);
  ink.add({ x: 40, y: -20 }, 0.8);
  ink.add({ x: 80, y: 1 }, 0.9);
  ink.straighten({ x: 80, y: 0 });
  expect(ink.straight).toBe(true);
  ink.add({ x: 100, y: 50 }, 0.7);
  ink.finish({ x: 120, y: 60 });
  expect(ink.samples.length).toBeGreaterThan(2);
  for (const point of ink.samples) expect(point.y).toBeCloseTo(point.x / 2);
  const element = ink.element("pen", "#000000", 10, false);
  expect(element.pressures?.length).toBe(element.points!.length / 2);
  expect(element.pressures?.[0]).toBe(0.1);
  expect(element.pressures?.at(-1)).toBe(0.7);
});
it("keeps mouse ink constant, collapses a held constant-pressure line, and ignores release zero", () => {
  const mouse = new InkStroke({ x: 0, y: 0 }, false, "light", 1, 0.1);
  mouse.add({ x: 40, y: 10 }, 0.9);
  mouse.straighten({ x: 80, y: 0 });
  expect(
    mouse.element("mouse", "#000000", 10, false).pressures,
  ).toBeUndefined();
  expect(mouse.samples).toHaveLength(2);
  const pen = new InkStroke({ x: 0, y: 0 }, true, "light", 1, 0.8);
  pen.add({ x: 40, y: 10 }, 0.8);
  pen.straighten({ x: 80, y: 0 });
  pen.finish({ x: 90, y: 0 });
  expect(pen.samples).toHaveLength(2);
  expect(pen.samples.at(-1)?.pressure).toBe(0.8);
});
it("leaves dots and nearly closed gestures freehand and bounds dense samples", () => {
  const dot = new InkStroke({ x: 0, y: 0 }, true, "light", 1, 0.4);
  dot.straighten({ x: 1, y: 1 });
  expect(dot.straight).toBe(false);
  const element = dot.element("dot", "#000000", 10, false);
  expect(element.points).toHaveLength(4);
  expect(element.pressures).toHaveLength(2);
  const dense = new InkStroke({ x: 0, y: 0 }, true, "off", 1, 0.5);
  for (let i = 1; i <= 20005; i++) dense.add({ x: i, y: 0 }, 0.5);
  dense.finish({ x: 21000, y: 0 });
  expect(dense.samples).toHaveLength(20000);
  expect(dense.samples.at(-1)?.x).toBe(21000);
});
