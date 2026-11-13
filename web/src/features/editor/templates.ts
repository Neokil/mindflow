import { footprint } from "../../geometry";
import type { CanvasElement, Point } from "../../model";
import { duplicateElements } from "../../model";

// Each insertion has fresh identities and no groups, locks, or links to its source.
export function instantiateTemplate(
  elements: CanvasElement[],
  center: Point = { x: 0, y: 0 },
) {
  if (!elements.length) return [];
  const points = elements.flatMap(footprint);
  const x =
    (Math.min(...points.map((p) => p.x)) +
      Math.max(...points.map((p) => p.x))) /
    2;
  const y =
    (Math.min(...points.map((p) => p.y)) +
      Math.max(...points.map((p) => p.y))) /
    2;
  return duplicateElements(elements, 0).map((element) => ({
    ...element,
    x: element.x - x + center.x,
    y: element.y - y + center.y,
    groupId: undefined,
    locked: undefined,
  }));
}
