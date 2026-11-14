import type { CanvasElement, Point } from "../../model";
import type { InkSample } from "../../strokeGeometry";
export type Stabilization = "off" | "light" | "strong";
const smoothingDistance = { off: 0, light: 2, strong: 6 };
const maxSamples = 20000;
function validPressure(value: number) {
  return Number.isFinite(value) ? Math.max(0, Math.min(1, value)) : 0.5;
}
export class InkStroke {
  samples: InkSample[];
  straight = false;
  private profile?: { fraction: number; pressure: number }[];
  constructor(
    readonly start: Point,
    readonly pressureEnabled: boolean,
    readonly stabilization: Stabilization,
    readonly zoom: number,
    pressure: number,
  ) {
    this.samples = [{ ...start, pressure: validPressure(pressure) }];
  }
  add(point: Point, pressure: number) {
    const value = validPressure(pressure);
    if (this.straight) {
      this.updateLine(point, value);
      return;
    }
    const last = this.samples[this.samples.length - 1];
    const distance = Math.hypot(point.x - last.x, point.y - last.y) * this.zoom;
    const smoothing = smoothingDistance[this.stabilization];
    const alpha = smoothing ? 1 - Math.exp(-distance / smoothing) : 1;
    const next = {
      x: last.x + (point.x - last.x) * alpha,
      y: last.y + (point.y - last.y) * alpha,
      pressure: value,
    };
    if (Math.hypot(next.x - last.x, next.y - last.y) * this.zoom < 0.5) {
      if (this.pressureEnabled) last.pressure = value;
      return;
    }
    if (this.samples.length < maxSamples) this.samples.push(next);
  }
  straighten(end: Point) {
    // Use the physical pointer position, not the stabilized (slightly delayed) tip.
    if (Math.hypot(end.x - this.start.x, end.y - this.start.y) * this.zoom < 12)
      return;
    this.finish(end);
    const distances = [0];
    for (let i = 1; i < this.samples.length; i++)
      distances.push(
        distances[i - 1] +
          Math.hypot(
            this.samples[i].x - this.samples[i - 1].x,
            this.samples[i].y - this.samples[i - 1].y,
          ),
      );
    const total = distances[distances.length - 1] || 1;
    const pressures = this.samples.map((p) => p.pressure);
    const samples =
      this.pressureEnabled &&
      Math.max(...pressures) - Math.min(...pressures) > 0.02
        ? this.samples.map((p, i) => ({
            fraction: distances[i] / total,
            pressure: p.pressure,
          }))
        : [
            { fraction: 0, pressure: pressures[0] },
            { fraction: 1, pressure: pressures[pressures.length - 1] },
          ];
    this.profile = samples;
    this.straight = true;
    this.updateLine(end, pressures[pressures.length - 1]);
  }
  finish(point: Point) {
    const last = this.samples[this.samples.length - 1];
    // pointerup reports zero pressure; preserve the last in-contact pressure.
    if (this.straight) {
      this.updateLine(point, last.pressure);
      return;
    }
    const sample = { ...point, pressure: last.pressure };
    if (Math.hypot(point.x - last.x, point.y - last.y) > 1e-8) {
      if (this.samples.length < maxSamples) this.samples.push(sample);
      else this.samples[this.samples.length - 1] = sample;
    }
  }
  element(
    id: string,
    color: string,
    width: number,
    highlight: boolean,
  ): CanvasElement {
    const samples = [...this.samples];
    if (samples.length === 1)
      samples.push({
        ...samples[0],
        x: samples[0].x + 0.1,
        y: samples[0].y + 0.1,
      });
    const minX = Math.min(...samples.map((p) => p.x)),
      minY = Math.min(...samples.map((p) => p.y));
    const bounds = {
      width: Math.max(0.1, Math.max(...samples.map((p) => p.x)) - minX),
      height: Math.max(0.1, Math.max(...samples.map((p) => p.y)) - minY),
    };
    return {
      id,
      type: "stroke",
      x: minX + bounds.width / 2,
      y: minY + bounds.height / 2,
      ...bounds,
      rotation: 0,
      points: samples.flatMap((p) => [
        (p.x - minX) / bounds.width,
        (p.y - minY) / bounds.height,
      ]),
      pressures: this.pressureEnabled
        ? samples.map((p) => p.pressure)
        : undefined,
      color,
      penWidth: width,
      highlight: highlight || undefined,
    };
  }
  private updateLine(end: Point, pressure: number) {
    if (!this.profile) return;
    this.profile[this.profile.length - 1].pressure = pressure;
    this.samples = this.profile.map((p) => ({
      x: this.start.x + (end.x - this.start.x) * p.fraction,
      y: this.start.y + (end.y - this.start.y) * p.fraction,
      pressure: p.pressure,
    }));
  }
}
