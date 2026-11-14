import { footprint } from "../../geometry";
import type { CanvasElement } from "../../model";
export type Alignment =
  "left" | "center" | "right" | "top" | "middle" | "bottom";
function units(elements: CanvasElement[]) {
  const groups = new Map<string, CanvasElement[]>();
  for (const e of elements) {
    const id = e.groupId ? `group:${e.groupId}` : `element:${e.id}`;
    groups.set(id, [...(groups.get(id) || []), e]);
  }
  return [...groups.values()].map((members) => {
    const points = members.flatMap((e) => {
      const pad = e.type === "stroke" ? (e.penWidth || 3) / 2 : 0;
      return footprint(e).flatMap((p) => [
        { x: p.x - pad, y: p.y - pad },
        { x: p.x + pad, y: p.y + pad },
      ]);
    });
    return {
      members,
      left: Math.min(...points.map((p) => p.x)),
      right: Math.max(...points.map((p) => p.x)),
      top: Math.min(...points.map((p) => p.y)),
      bottom: Math.max(...points.map((p) => p.y)),
    };
  });
}
export function canAlign(elements: CanvasElement[]) {
  return (
    !elements.some((e) => e.locked) &&
    new Set(
      elements.map((e) =>
        e.groupId ? `group:${e.groupId}` : `element:${e.id}`,
      ),
    ).size > 1
  );
}
export function alignElements(
  elements: CanvasElement[],
  ids: string[],
  alignment: Alignment,
): CanvasElement[] {
  const selected = elements.filter((e) => ids.includes(e.id));
  if (!canAlign(selected)) return elements;
  const blocks = units(selected),
    horizontal = ["left", "center", "right"].includes(alignment);
  const start = Math.min(...blocks.map((b) => (horizontal ? b.left : b.top))),
    end = Math.max(...blocks.map((b) => (horizontal ? b.right : b.bottom)));
  const target =
    alignment === "left" || alignment === "top"
      ? start
      : alignment === "right" || alignment === "bottom"
        ? end
        : (start + end) / 2;
  const offsets = new Map<string, number>();
  for (const block of blocks) {
    const position =
      alignment === "left"
        ? block.left
        : alignment === "right"
          ? block.right
          : alignment === "top"
            ? block.top
            : alignment === "bottom"
              ? block.bottom
              : horizontal
                ? (block.left + block.right) / 2
                : (block.top + block.bottom) / 2;
    for (const e of block.members) offsets.set(e.id, target - position);
  }
  return elements.map((e) =>
    offsets.has(e.id)
      ? {
          ...e,
          [horizontal ? "x" : "y"]:
            (horizontal ? e.x : e.y) + offsets.get(e.id)!,
        }
      : e,
  );
}
