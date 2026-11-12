import { afterEach, expect, it, vi } from "vitest";
import { uid } from "./model";

afterEach(() => vi.unstubAllGlobals());

it("creates a UUID v4 with only getRandomValues available", () => {
  const getRandomValues = vi.fn((bytes: Uint8Array) => bytes.fill(255));
  vi.stubGlobal("crypto", { getRandomValues });
  expect(uid()).toBe("ffffffff-ffff-4fff-bfff-ffffffffffff");
  expect(getRandomValues).toHaveBeenCalledOnce();
});
