import { afterEach, expect, it, vi } from "vitest";
import type { CanvasElement } from "./model";
import {
  duplicateElements,
  uid,
  wholeFontSize,
  validClipboardElements,
} from "./model";

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

it("accepts portable pressure ink and rejects mismatched or invalid pressure samples", () => {
  const stroke = {
    id: "ink",
    type: "stroke",
    x: 0,
    y: 0,
    width: 100,
    height: 100,
    rotation: 0,
    points: [0, 0, 1, 1],
    pressures: [0.2, 0.9],
  };
  expect(validClipboardElements([stroke])).toBe(true);
  for (const pressures of [[0.5], [0, 2], [NaN, 1], [0, Infinity]])
    expect(validClipboardElements([{ ...stroke, pressures }])).toBe(false);
  expect(validClipboardElements([{ ...stroke, highlight: true }])).toBe(false);
  expect(validClipboardElements([{ ...stroke, type: "text" }])).toBe(false);
});

it("duplicates internal element links and discards foreign clipboard targets", () => {
  const source: CanvasElement[] = [
    {
      id: "source",
      type: "shape",
      shape: "rectangle",
      x: 0,
      y: 0,
      width: 100,
      height: 100,
      rotation: 0,
      link: { pageId: "first", elementId: "target" },
    },
    {
      id: "target",
      type: "text",
      x: 200,
      y: 0,
      width: 100,
      height: 100,
      rotation: 0,
      link: { pageId: "other", elementId: "outside" },
    },
  ];
  const copied = duplicateElements(source, 20, "first", "second", false);
  expect(copied[0].link).toEqual({ pageId: "second", elementId: copied[1].id });
  expect(copied[1].link).toBeUndefined();
  expect(source[0].link?.elementId).toBe("target");
});
