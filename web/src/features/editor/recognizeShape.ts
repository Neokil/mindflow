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
function bounds(points: Point[], angle: number, robust = false) {
  const cos = Math.cos(angle),
    sin = Math.sin(angle),
    local = points.map((p) => ({
      x: p.x * cos + p.y * sin,
      y: -p.x * sin + p.y * cos,
    }));
  const xs = local.map((p) => p.x).sort((a, b) => a - b),
    ys = local.map((p) => p.y).sort((a, b) => a - b);
  const trim = robust ? Math.floor(points.length * 0.025) : 0;
  const minX = xs[trim],
    maxX = xs[xs.length - 1 - trim],
    minY = ys[trim],
    maxY = ys[ys.length - 1 - trim];
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
    distance(points[0], points.at(-1)!) > Math.max(8 / zoom, diagonal * 0.34)
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
  if (Math.abs(area) / 2 < axis.width * axis.height * 0.32) return;
  const direction = area >= 0 ? 1 : -1,
    fractions = Array.from({ length: 129 }, (_, i) => i / 128);
  // Remove small hand oscillations before scoring geometry. Arc-length sampling
  // makes slow corners and fast sides contribute equally.
  const smooth = outline.map((point, i) => {
    const previous = outline[(i + outline.length - 1) % outline.length],
      next = outline[(i + 1) % outline.length];
    return {
      x: (previous.x + 2 * point.x + next.x) / 4,
      y: (previous.y + 2 * point.y + next.y) / 4,
    };
  });
  let ellipse: { fit: ReturnType<typeof bounds>; error: number } | undefined;
  for (let step = 0; step < 18; step++) {
    const fit = bounds(smooth, (step * Math.PI) / 36, true);
    const errors = fit.local
      .map((p) => Math.abs(Math.hypot(p.x, p.y) - 1))
      .sort((a, b) => a - b);
    const error =
      errors
        .slice(0, Math.ceil(errors.length * 0.94))
        .reduce((a, b) => a + b, 0) / Math.ceil(errors.length * 0.94);
    const a = fit.width / 2,
      b = fit.height / 2,
      perimeter =
        Math.PI * (3 * (a + b) - Math.sqrt((3 * a + b) * (a + 3 * b)));
    const coverage = new Set(
      fit.local.map((p) => Math.floor(wrap(Math.atan2(p.y, p.x) / tau) * 12)),
    ).size;
    if (
      error < 0.145 &&
      errors[Math.floor(errors.length * 0.9)] < 0.32 &&
      length / perimeter < 1.65 &&
      coverage >= 10 &&
      (!ellipse || error < ellipse.error)
    )
      ellipse = { fit, error };
  }
  let best: { fit: ReturnType<typeof bounds>; error: number } | undefined;
  for (let step = 0; step < 36; step++) {
    const fit = bounds(smooth, (step * Math.PI) / 72, true);
    const errors = fit.local.map((p) =>
        Math.abs(1 - Math.max(Math.abs(p.x), Math.abs(p.y))),
      ),
      error = errors.reduce((a, b) => a + b, 0) / outline.length;
    if (error > 0.15 || length / (2 * (fit.width + fit.height)) > 1.65)
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
          Math.min(...fit.local.map((p) => Math.hypot(p.x - x, p.y - y))) > 0.6,
      )
    )
      continue;
    const straightEdges =
      fit.local.reduce((sum, p, i) => {
        const next = fit.local[(i + 1) % fit.local.length],
          dx = Math.abs(next.x - p.x),
          dy = Math.abs(next.y - p.y);
        return (
          sum + (Math.min(dx, dy) / (Math.hypot(dx, dy) || 1) < 0.3 ? 1 : 0)
        );
      }, 0) / fit.local.length;
    const score = error - (straightEdges > 0.6 ? 0.035 : 0);
    if (!best || score < best.error) best = { fit, error: score };
  }
  if (ellipse && (!best || ellipse.error + 0.012 < best.error)) {
    const { fit } = ellipse,
      start = fit.local[0],
      angle = Math.atan2(start.y, start.x);
    return {
      kind: "circle",
      center: fit.center,
      fractions,
      pointAt: (f) =>
        fit.world(
          (Math.cos(angle + direction * tau * f) * fit.width) / 2,
          (Math.sin(angle + direction * tau * f) * fit.height) / 2,
        ),
    };
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
