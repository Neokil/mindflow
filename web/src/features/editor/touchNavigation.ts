import { worldPoint, zoomAt } from "../../geometry";
import type { Point } from "../../model";
import type { Camera } from "./types";

export function touchCamera(
  camera: Camera,
  start: [Point, Point],
  current: [Point, Point],
): Camera {
  const middle = (points: [Point, Point]) => ({
    x: (points[0].x + points[1].x) / 2,
    y: (points[0].y + points[1].y) / 2,
  });
  const distance = (points: [Point, Point]) =>
    Math.hypot(points[0].x - points[1].x, points[0].y - points[1].y);
  const origin = middle(start),
    target = middle(current);
  const world = worldPoint(origin, camera);
  const zoom = zoomAt(
    origin,
    camera,
    distance(current) / Math.max(1, distance(start)),
  ).zoom;
  return { x: target.x - world.x * zoom, y: target.y - world.y * zoom, zoom };
}
