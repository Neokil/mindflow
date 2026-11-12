import { afterEach, expect, it, vi } from "vitest";
import { uid, wholeFontSize, validClipboardElements } from "./model";

afterEach(() => vi.unstubAllGlobals());

it("creates a UUID v4 with only getRandomValues available", () => {
  const getRandomValues = vi.fn((bytes: Uint8Array) => bytes.fill(255));
  vi.stubGlobal("crypto", { getRandomValues });
  expect(uid()).toBe("ffffffff-ffff-4fff-bfff-ffffffffffff");
  expect(getRandomValues).toHaveBeenCalledOnce();
});

it("rounds font sizes and enforces their supported range", () => {
  expect(wholeFontSize(28.4)).toBe(28);
  expect(wholeFontSize(28.6)).toBe(29);
  expect(wholeFontSize(0)).toBe(1);
  expect(wholeFontSize(2500)).toBe(2000);
});

it("rejects clipboard image crops outside the original image", () => {
  const image = {
    id: "image",
    type: "image",
    x: 0,
    y: 0,
    width: 100,
    height: 100,
    rotation: 0,
    crop: { x: 0.2, y: 0.1, width: 0.5, height: 0.6 },
  };
  expect(validClipboardElements([image])).toBe(true);
  expect(
    validClipboardElements([{ ...image, crop: { ...image.crop, width: 0.9 } }]),
  ).toBe(false);
  expect(
    validClipboardElements([{ ...image, crop: { ...image.crop, height: 0 } }]),
  ).toBe(false);
});
