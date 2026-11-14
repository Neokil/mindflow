import type { Point } from "./model";
export type InkSample = Point & { pressure: number };
export function pressureWidth(width: number, pressure?: number) {
  return pressure === undefined ? width : width * (0.2 + 0.8 * pressure);
}
// A single filled ribbon avoids seams between variable-width stroke segments.
export function strokeOutline(samples: InkSample[], width: number): number[] {
  if (!samples.length) return [];
  const points = samples.filter(
    (p, index) =>
      !index ||
      Math.hypot(p.x - samples[index - 1].x, p.y - samples[index - 1].y) > 1e-8,
  );
  if (points.length === 1) {
    const p = points[0],
      radius = pressureWidth(width, p.pressure) / 2;
    return Array.from({ length: 24 }, (_, i) => [
      p.x + Math.cos((i * Math.PI) / 12) * radius,
      p.y + Math.sin((i * Math.PI) / 12) * radius,
    ]).flat();
  }
  const normals = points.map((p, index) => {
    const before = points[Math.max(0, index - 1)],
      after = points[Math.min(points.length - 1, index + 1)];
    let dx = after.x - before.x,
      dy = after.y - before.y;
    if (Math.hypot(dx, dy) < 1e-8) {
      dx = p.x - before.x;
      dy = p.y - before.y;
    }
    const length = Math.hypot(dx, dy) || 1;
    return { x: -dy / length, y: dx / length };
  });
  const left = points.map((p, i) => {
    const r = pressureWidth(width, p.pressure) / 2;
    return { x: p.x + normals[i].x * r, y: p.y + normals[i].y * r };
  });
  const right = points.map((p, i) => {
    const r = pressureWidth(width, p.pressure) / 2;
    return { x: p.x - normals[i].x * r, y: p.y - normals[i].y * r };
  });
  function cap(index: number, start: number) {
    const p = points[index],
      radius = pressureWidth(width, p.pressure) / 2,
      angle = Math.atan2(normals[index].y, normals[index].x) + start;
    return Array.from({ length: 8 }, (_, i) => ({
      x: p.x + Math.cos(angle - ((i + 1) * Math.PI) / 8) * radius,
      y: p.y + Math.sin(angle - ((i + 1) * Math.PI) / 8) * radius,
    }));
  }
  return [
    ...left,
    ...cap(points.length - 1, 0),
    ...right.reverse(),
    ...cap(0, Math.PI),
  ].flatMap((p) => [p.x, p.y]);
}
