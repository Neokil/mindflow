import Konva from "konva";
import type { CanvasElement } from "../../model";
import { wholeFontSize } from "../../model";

export function textVerticalOffset(
  element: CanvasElement,
  value: string,
): number {
  if (!element.verticalAlign || element.verticalAlign === "top") return 0;
  const measure = new Konva.Text({
    text: value || " ",
    width: element.width,
    fontSize: wholeFontSize(element.fontSize || 24),
    fontFamily: "Arial",
    fontStyle:
      `${element.bold ? "bold" : ""} ${element.italic ? "italic" : ""}`.trim() ||
      "normal",
    lineHeight: 1.3,
    wrap: "word",
  });
  const free = Math.max(0, element.height - measure.height());
  measure.destroy();
  return element.verticalAlign === "bottom" ? free : free / 2;
}
