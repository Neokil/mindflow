import { footprint } from "../../geometry";
import type { CanvasDocument } from "../../model";
export function exportBounds(doc: CanvasDocument) {
  if (!doc.elements.length)
    return { x: -500, y: -350, width: 1000, height: 700 };
  let left = Infinity,
    top = Infinity,
    right = -Infinity,
    bottom = -Infinity;
  for (const element of doc.elements) {
    const pad = element.type === "stroke" ? (element.penWidth || 3) / 2 : 0;
    for (const p of footprint(element)) {
      left = Math.min(left, p.x - pad);
      top = Math.min(top, p.y - pad);
      right = Math.max(right, p.x + pad);
      bottom = Math.max(bottom, p.y + pad);
    }
  }
  return {
    x: left - 32,
    y: top - 32,
    width: right - left + 64,
    height: bottom - top + 64,
  };
}
export type ExportRegion = {
  x: number;
  y: number;
  width: number;
  height: number;
};
export function exportSize(
  doc: CanvasDocument,
  resolution: number,
  region?: ExportRegion,
) {
  const bounds = region || exportBounds(doc);
  if (
    !Object.values(bounds).every(Number.isFinite) ||
    bounds.width <= 0 ||
    bounds.height <= 0 ||
    !Number.isFinite(resolution) ||
    resolution <= 0 ||
    resolution > 4096
  )
    throw new Error(
      "Choose a valid export area and resolution up to 4,096 pixels.",
    );
  const scale = resolution / Math.max(bounds.width, bounds.height);
  return {
    bounds,
    scale,
    width: Math.max(1, Math.ceil(bounds.width * scale)),
    height: Math.max(1, Math.ceil(bounds.height * scale)),
  };
}
export function safeFilename(name: string) {
  return (
    name
      .replace(/[^\p{L}\p{N} _.-]/gu, "_")
      .replace(/^[. ]+|[. ]+$/g, "")
      .slice(0, 100) || "workspace"
  );
}
