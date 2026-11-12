import { afterEach, expect, it, vi } from "vitest";
import { uid, wholeFontSize } from "./model";

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
