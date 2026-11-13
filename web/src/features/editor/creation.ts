import { snap } from "../../geometry";
import type { CanvasDocument, Point } from "../../model";

export function creationBounds(
  start: Point,
  end: Point,
  square: boolean,
  settings: CanvasDocument["settings"],
) {
  const grid = (value: number) =>
    settings.snap ? snap(value, settings.gridSize) : value;
  const origin = { x: grid(start.x), y: grid(start.y) };
  let dx = grid(end.x) - origin.x,
    dy = grid(end.y) - origin.y;
  if (square) {
    const side = Math.max(Math.abs(dx), Math.abs(dy));
    dx = (dx < 0 ? -1 : 1) * side;
    dy = (dy < 0 ? -1 : 1) * side;
  }
  return {
    x: origin.x + dx / 2,
    y: origin.y + dy / 2,
    width: Math.max(0.1, Math.abs(dx)),
    height: Math.max(0.1, Math.abs(dy)),
  };
}
