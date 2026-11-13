import Konva from "konva";
import type { CanvasElement } from "../../model";
import { wholeFontSize } from "../../model";
export function listRows(element: CanvasElement) {
  const fontSize = wholeFontSize(element.fontSize || 24);
  const indent = fontSize * 1.8;
  let top = 0;
  const rows = (element.text || "").split("\n").map((text, index) => {
    const node = new Konva.Text({
      text: text || " ",
      width: Math.max(1, element.width - indent),
      fontSize,
      fontFamily: "Arial",
      fontStyle:
        `${element.bold ? "bold" : ""} ${element.italic ? "italic" : ""}`.trim() ||
        "normal",
      lineHeight: 1.3,
      wrap: "word",
    });
    const height = Math.max(fontSize * 1.3, node.height());
    node.destroy();
    const row = { text, index, top, height };
    top += height;
    return row;
  });
  const free = Math.max(0, element.height - top);
  return {
    rows,
    indent,
    fontSize,
    offset:
      element.verticalAlign === "bottom"
        ? free
        : element.verticalAlign === "middle"
          ? free / 2
          : 0,
  };
}
// Keep checkmarks attached to unchanged lines when items are inserted/deleted.
export function retainCheckedLines(
  before: string,
  after: string,
  checked: number[] = [],
) {
  const available = new Map<string, boolean[]>();
  before.split("\n").forEach((line, index) => {
    const states = available.get(line) || [];
    states.push(checked.includes(index));
    available.set(line, states);
  });
  return after
    .split("\n")
    .flatMap((line, index) => (available.get(line)?.shift() ? [index] : []));
}
