import type { CanvasElement, ImageCrop } from "../../model";
import { localToWorld } from "../../geometry";

export const fullImageCrop: ImageCrop = { x: 0, y: 0, width: 1, height: 1 };

// Keep the source pixels at their existing world positions, including rotated images.
export function croppedElement(
  element: CanvasElement,
  crop: ImageCrop,
): Partial<CanvasElement> {
  const previous = element.crop || fullImageCrop;
  const sourceWidth = element.width / previous.width;
  const sourceHeight = element.height / previous.height;
  const center = localToWorld(
    {
      x:
        (crop.x + crop.width / 2 - previous.x - previous.width / 2) *
        sourceWidth,
      y:
        (crop.y + crop.height / 2 - previous.y - previous.height / 2) *
        sourceHeight,
    },
    element,
  );
  return {
    ...center,
    width: Math.max(0.1, Math.min(1e6, sourceWidth * crop.width)),
    height: Math.max(0.1, Math.min(1e6, sourceHeight * crop.height)),
    crop: { ...crop },
  };
}
