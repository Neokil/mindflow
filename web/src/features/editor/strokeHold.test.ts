import { afterEach, expect, it, vi } from "vitest";
import { StrokeHold, strokeHoldDelay } from "./strokeHold";

afterEach(() => vi.useRealTimers());

it("tolerates hand jitter in screen pixels while waiting for a hold", () => {
  vi.useFakeTimers();
  const hold = new StrokeHold();
  const straighten = vi.fn();
  hold.move({ x: 100, y: 20 }, 2, straighten);
  vi.advanceTimersByTime(500);
  hold.move({ x: 101, y: 21 }, 2, straighten);
  vi.advanceTimersByTime(strokeHoldDelay - 500);
  expect(straighten).toHaveBeenCalledTimes(1);
  hold.cancel();
});

it("restarts on movement and cancels work before a new gesture", () => {
  vi.useFakeTimers();
  const hold = new StrokeHold();
  const abandoned = vi.fn();
  const next = vi.fn();
  hold.move({ x: 10, y: 20 }, 1, abandoned);
  vi.advanceTimersByTime(strokeHoldDelay - 1);
  hold.move({ x: 15, y: 20 }, 1, abandoned);
  vi.advanceTimersByTime(strokeHoldDelay - 1);
  expect(abandoned).not.toHaveBeenCalled();
  hold.cancel();
  hold.move({ x: 30, y: 40 }, 1, next);
  vi.advanceTimersByTime(strokeHoldDelay);
  expect(abandoned).not.toHaveBeenCalled();
  expect(next).toHaveBeenCalledTimes(1);
  hold.cancel();
});
