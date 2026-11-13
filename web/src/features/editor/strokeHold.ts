import type { Point } from "../../model";

export const strokeHoldDelay = 650;
const holdTolerance = 4; // Screen pixels, independent of canvas zoom.

export class StrokeHold {
  private timer?: ReturnType<typeof setTimeout>;
  private anchor?: Point;

  move(point: Point, zoom: number, onHold: () => void) {
    if (
      this.anchor &&
      Math.hypot(point.x - this.anchor.x, point.y - this.anchor.y) * zoom <=
        holdTolerance
    )
      return;
    this.cancel();
    this.anchor = point;
    this.timer = setTimeout(onHold, strokeHoldDelay);
  }

  cancel() {
    clearTimeout(this.timer);
    this.timer = undefined;
    this.anchor = undefined;
  }
}
