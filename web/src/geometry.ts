import type { CanvasElement, Point } from "./model";
const EPS = 1e-7;
export const snap = (n: number, grid: number) => Math.round(n / grid) * grid;
export function worldPoint(p: Point, camera: Point & { zoom: number }): Point {
  return {
    x: (p.x - camera.x) / camera.zoom,
    y: (p.y - camera.y) / camera.zoom,
  };
}
export function zoomAt(
  p: Point,
  camera: Point & { zoom: number },
  factor: number,
) {
  const world = worldPoint(p, camera);
  const zoom = Math.max(0.05, Math.min(20, camera.zoom * factor));
  return { x: p.x - world.x * zoom, y: p.y - world.y * zoom, zoom };
}
export function localToWorld(p: Point, e: CanvasElement): Point {
  const c = Math.cos(e.rotation),
    s = Math.sin(e.rotation);
  return { x: e.x + p.x * c - p.y * s, y: e.y + p.x * s + p.y * c };
}
const cross = (a: Point, b: Point, c: Point) =>
  (b.x - a.x) * (c.y - a.y) - (b.y - a.y) * (c.x - a.x);
function onSegment(p: Point, a: Point, b: Point) {
  return (
    Math.abs(cross(a, b, p)) < EPS &&
    p.x >= Math.min(a.x, b.x) - EPS &&
    p.x <= Math.max(a.x, b.x) + EPS &&
    p.y >= Math.min(a.y, b.y) - EPS &&
    p.y <= Math.max(a.y, b.y) + EPS
  );
}
export function pointInPolygon(p: Point, poly: Point[]) {
  let inside = false;
  for (let i = 0, j = poly.length - 1; i < poly.length; j = i++) {
    const a = poly[i],
      b = poly[j];
    if (onSegment(p, a, b)) return true;
    if (
      a.y > p.y !== b.y > p.y &&
      p.x < ((b.x - a.x) * (p.y - a.y)) / (b.y - a.y) + a.x
    )
      inside = !inside;
  }
  return inside;
}
export function segmentInside(a: Point, b: Point, polygon: Point[]) {
  if (!pointInPolygon(a, polygon) || !pointInPolygon(b, polygon)) return false;
  const dx = b.x - a.x,
    dy = b.y - a.y;
  const ts = [0, 1];
  for (let i = 0; i < polygon.length; i++) {
    const p = polygon[i],
      q = polygon[(i + 1) % polygon.length];
    const ex = q.x - p.x,
      ey = q.y - p.y;
    const denominator = dx * ey - dy * ex;
    if (Math.abs(denominator) < EPS) continue;
    const t = ((p.x - a.x) * ey - (p.y - a.y) * ex) / denominator;
    const u = ((p.x - a.x) * dy - (p.y - a.y) * dx) / denominator;
    if (t > 0 && t < 1 && u >= 0 && u <= 1) ts.push(t);
  }
  ts.sort((a, b) => a - b);
  for (let i = 1; i < ts.length; i++) {
    const t = (ts[i - 1] + ts[i]) / 2;
    if (!pointInPolygon({ x: a.x + dx * t, y: a.y + dy * t }, polygon))
      return false;
  }
  return true;
}
function polygonInside(points: Point[], lasso: Point[]) {
  return points.every((p, i) =>
    segmentInside(p, points[(i + 1) % points.length], lasso),
  );
}
export function footprint(e: CanvasElement): Point[] {
  let points: Point[];
  if (e.type === "shape" && (e.shape === "circle" || e.shape === "star")) {
    const count = e.shape === "star" ? 10 : 96;
    points = Array.from({ length: count }, (_, i) => {
      const angle = (i * 2 * Math.PI) / count - Math.PI / 2;
      const scale = e.shape === "star" && i % 2 ? 0.45 : 1;
      return {
        x: Math.cos(angle) * (e.width / 2 + (e.borderWidth || 0) / 2) * scale,
        y: Math.sin(angle) * (e.height / 2 + (e.borderWidth || 0) / 2) * scale,
      };
    });
  } else {
    const pad = e.type === "shape" ? (e.borderWidth || 0) / 2 : 0;
    const w = e.width / 2 + pad,
      h = e.height / 2 + pad;
    points = [
      { x: -w, y: -h },
      { x: w, y: -h },
      { x: w, y: h },
      { x: -w, y: h },
    ];
  }
  return points.map((p) => localToWorld(p, e));
}
export function fullyContained(e: CanvasElement, lasso: Point[]) {
  if (lasso.length < 3) return false;
  if (e.type !== "stroke") return polygonInside(footprint(e), lasso);
  const points = e.points || [];
  const radius = (e.penWidth || 1) / 2;
  const circles: Point[][] = [];
  for (let i = 0; i < points.length; i += 2)
    circles.push(
      Array.from({ length: 16 }, (_, n) =>
        localToWorld(
          {
            x:
              (points[i] - 0.5) * e.width +
              Math.cos((n * Math.PI) / 8) * radius,
            y:
              (points[i + 1] - 0.5) * e.height +
              Math.sin((n * Math.PI) / 8) * radius,
          },
          e,
        ),
      ),
    );
  return circles.every(
    (circle, i) =>
      polygonInside(circle, lasso) &&
      (!i ||
        circle.every((p, j) => segmentInside(circles[i - 1][j], p, lasso))),
  );
}
export function lassoSelection(elements: CanvasElement[], lasso: Point[]) {
  const enclosed = new Set(
    elements.filter((e) => fullyContained(e, lasso)).map((e) => e.id),
  );
  const incompleteGroups = new Set(
    elements
      .filter((e) => e.groupId && !enclosed.has(e.id))
      .map((e) => e.groupId),
  );
  return elements
    .filter(
      (e) =>
        enclosed.has(e.id) && (!e.groupId || !incompleteGroups.has(e.groupId)),
    )
    .map((e) => e.id);
}
export function selectedWithGroup(elements: CanvasElement[], id: string) {
  const e = elements.find((e) => e.id === id);
  return e?.groupId
    ? elements
        .filter((item) => item.groupId === e.groupId)
        .map((item) => item.id)
    : [id];
}
