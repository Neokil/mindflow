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
export function exportSize(doc: CanvasDocument, resolution: number) {
  const bounds = exportBounds(doc),
    scale = resolution / Math.max(bounds.width, bounds.height);
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
