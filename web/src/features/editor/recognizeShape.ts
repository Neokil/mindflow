import type { Point } from "../../model";
export type RecognizedShape = {
  kind: "circle" | "rectangle";
  center: Point;
  fractions: number[];
  pointAt: (fraction: number) => Point;
};
const tau = Math.PI * 2;
const distance = (a: Point, b: Point) => Math.hypot(a.x - b.x, a.y - b.y);
const wrap = (value: number) => ((value % 1) + 1) % 1;
export function pathFractions(points: Point[]) {
  const lengths = [0];
  for (let i = 1; i < points.length; i++)
    lengths.push(lengths[i - 1] + distance(points[i - 1], points[i]));
  const total = lengths.at(-1) || 1;
  return lengths.map((length) => length / total);
}
function samplePath(points: Point[], count: number) {
  const fractions = pathFractions(points);
  let index = 1;
  return Array.from({ length: count }, (_, i) => {
    const fraction = i / (count - 1);
    while (index < points.length - 1 && fractions[index] < fraction) index++;
    const previous = points[index - 1],
      next = points[index],
      part =
        (fraction - fractions[index - 1]) /
        (fractions[index] - fractions[index - 1] || 1);
    return {
      x: previous.x + (next.x - previous.x) * part,
      y: previous.y + (next.y - previous.y) * part,
    };
  });
}
function bounds(points: Point[], angle: number) {
  const cos = Math.cos(angle),
    sin = Math.sin(angle),
    local = points.map((p) => ({
      x: p.x * cos + p.y * sin,
      y: -p.x * sin + p.y * cos,
    }));
  const minX = Math.min(...local.map((p) => p.x)),
    maxX = Math.max(...local.map((p) => p.x)),
    minY = Math.min(...local.map((p) => p.y)),
    maxY = Math.max(...local.map((p) => p.y));
  const cx = (minX + maxX) / 2,
    cy = (minY + maxY) / 2,
    width = maxX - minX,
    height = maxY - minY;
  const world = (x: number, y: number) => ({
    x: (cx + x) * cos - (cy + y) * sin,
    y: (cx + x) * sin + (cy + y) * cos,
  });
  return {
    width,
    height,
    center: world(0, 0),
    world,
    local: local.map((p) => ({
      x: (p.x - cx) / (width / 2 || 1),
      y: (p.y - cy) / (height / 2 || 1),
    })),
  };
}
// Only sufficiently closed, single-loop outlines are recognized; doodles stay freehand.
export function recognizeShape(
  points: Point[],
  zoom: number,
): RecognizedShape | undefined {
  if (points.length < 5) return;
  const axis = bounds(points, 0),
    diagonal = Math.hypot(axis.width, axis.height);
  if (
    Math.min(axis.width, axis.height) * zoom < 18 ||
    diagonal * zoom < 30 ||
    distance(points[0], points.at(-1)!) > Math.max(8 / zoom, diagonal * 0.18)
  )
    return;
  const sampled = samplePath([...points, points[0]], 65),
    outline = sampled.slice(0, -1);
  let area = 0,
    length = 0;
  for (let i = 1; i < sampled.length; i++) {
    area += sampled[i - 1].x * sampled[i].y - sampled[i].x * sampled[i - 1].y;
    length += distance(sampled[i - 1], sampled[i]);
  }
  if (Math.abs(area) / 2 < axis.width * axis.height * 0.45) return;
  const direction = area >= 0 ? 1 : -1,
    fractions = Array.from({ length: 129 }, (_, i) => i / 128);
  const circle = bounds(outline, 0),
    radial = circle.local.map((p) => Math.abs(Math.hypot(p.x, p.y) - 1));
  const circleError = radial.reduce((a, b) => a + b, 0) / outline.length;
  const radiusA = circle.width / 2,
    radiusB = circle.height / 2,
    perimeter =
      Math.PI *
      (3 * (radiusA + radiusB) -
        Math.sqrt((3 * radiusA + radiusB) * (radiusA + 3 * radiusB)));
  if (
    circleError < 0.085 &&
    Math.max(...radial) < 0.25 &&
    length / perimeter < 1.35
  ) {
    const start = circle.local[0],
      angle = Math.atan2(start.y, start.x);
    return {
      kind: "circle",
      center: circle.center,
      fractions,
      pointAt: (f) =>
        circle.world(
          Math.cos(angle + direction * tau * f) * radiusA,
          Math.sin(angle + direction * tau * f) * radiusB,
        ),
    };
  }
  let best: { fit: ReturnType<typeof bounds>; error: number } | undefined;
  for (let step = 0; step < 36; step++) {
    const fit = bounds(outline, (step * Math.PI) / 72);
    const errors = fit.local.map(
        (p) => 1 - Math.max(Math.abs(p.x), Math.abs(p.y)),
      ),
      error = errors.reduce((a, b) => a + b, 0) / outline.length;
    if (error > 0.075 || length / (2 * (fit.width + fit.height)) > 1.35)
      continue;
    const corners = [
      [-1, -1],
      [1, -1],
      [1, 1],
      [-1, 1],
    ];
    if (
      corners.some(
        ([x, y]) =>
          Math.min(...fit.local.map((p) => Math.hypot(p.x - x, p.y - y))) >
          0.32,
      )
    )
      continue;
    if (!best || error < best.error) best = { fit, error };
  }
  if (!best) return;
  const { fit } = best,
    halfW = fit.width / 2,
    halfH = fit.height / 2;
  const corners = [
      { x: -halfW, y: -halfH },
      { x: halfW, y: -halfH },
      { x: halfW, y: halfH },
      { x: -halfW, y: halfH },
    ],
    edges = [fit.width, fit.height, fit.width, fit.height],
    perim = 2 * (fit.width + fit.height);
  const start = { x: fit.local[0].x * halfW, y: fit.local[0].y * halfH };
  let startPhase = 0,
    nearest = Infinity,
    offset = 0;
  for (let i = 0; i < 4; i++) {
    const a = corners[i],
      b = corners[(i + 1) % 4],
      dx = b.x - a.x,
      dy = b.y - a.y;
    const part = Math.max(
      0,
      Math.min(
        1,
        ((start.x - a.x) * dx + (start.y - a.y) * dy) / (dx * dx + dy * dy),
      ),
    );
    const gap = distance(start, { x: a.x + dx * part, y: a.y + dy * part });
    if (gap < nearest) {
      nearest = gap;
      startPhase = (offset + edges[i] * part) / perim;
    }
    offset += edges[i];
  }
  offset = 0;
  for (let i = 0; i < 4; i++) {
    fractions.push(wrap(direction * (offset / perim - startPhase)));
    offset += edges[i];
  }
  return {
    kind: "rectangle",
    center: fit.center,
    fractions,
    pointAt: (f) => {
      let position = wrap(startPhase + direction * f) * perim;
      for (let i = 0; i < 4; i++) {
        if (position <= edges[i] || i === 3) {
          const a = corners[i],
            b = corners[(i + 1) % 4],
            part = position / edges[i];
          return fit.world(a.x + (b.x - a.x) * part, a.y + (b.y - a.y) * part);
        }
        position -= edges[i];
      }
      return fit.world(-halfW, -halfH);
    },
  };
}
