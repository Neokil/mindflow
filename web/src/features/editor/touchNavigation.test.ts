import { expect, it } from "vitest";
import { touchCamera } from "./touchNavigation";
import { worldPoint } from "../../geometry";

it("keeps the canvas point between the fingers anchored while panning and pinching", () => {
  const camera = { x: 20, y: 30, zoom: 2 };
  const start: [{ x: number; y: number }, { x: number; y: number }] = [
    { x: 100, y: 100 },
    { x: 200, y: 100 },
  ];
  const target: [{ x: number; y: number }, { x: number; y: number }] = [
    { x: 150, y: 160 },
    { x: 350, y: 160 },
  ];
  const result = touchCamera(camera, start, target);
  expect(result.zoom).toBe(4);
  expect(worldPoint({ x: 250, y: 160 }, result)).toEqual(
    worldPoint({ x: 150, y: 100 }, camera),
  );
  expect(
    touchCamera(camera, start, [
      { x: 120, y: 130 },
      { x: 220, y: 130 },
    ]),
  ).toEqual({ x: 40, y: 60, zoom: 2 });
});
it("clamps extreme pinches and handles coincident fingers without invalid camera values", () => {
  const camera = { x: 0, y: 0, zoom: 1 };
  expect(
    touchCamera(
      camera,
      [
        { x: 0, y: 0 },
        { x: 1, y: 0 },
      ],
      [
        { x: 0, y: 0 },
        { x: 10000, y: 0 },
      ],
    ).zoom,
  ).toBe(20);
  expect(
    touchCamera(
      camera,
      [
        { x: 0, y: 0 },
        { x: 100, y: 0 },
      ],
      [
        { x: 1, y: 1 },
        { x: 1, y: 1 },
      ],
    ).zoom,
  ).toBe(0.05);
});
