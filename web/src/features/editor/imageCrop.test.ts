import { expect, it } from "vitest";
import { croppedElement, fullImageCrop } from "./imageCrop";
import type { CanvasElement } from "../../model";

it("keeps cropped pixels in position on a rotated image and restores the original bounds", () => {
  const image: CanvasElement = {
    id: "image",
    type: "image",
    x: 100,
    y: 200,
    width: 400,
    height: 200,
    rotation: Math.PI / 2,
  };
  const crop = croppedElement(image, {
    x: 0.25,
    y: 0,
    width: 0.5,
    height: 0.5,
  });
  expect(crop.width).toBe(200);
  expect(crop.height).toBe(100);
  expect(crop.x).toBeCloseTo(150);
  expect(crop.y).toBeCloseTo(200);
  const restored = croppedElement({ ...image, ...crop }, fullImageCrop);
  expect(restored.width).toBe(400);
  expect(restored.height).toBe(200);
  expect(restored.x).toBeCloseTo(image.x);
  expect(restored.y).toBeCloseTo(image.y);
});
