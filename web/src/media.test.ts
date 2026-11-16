import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { api } from "./api";
import {
  assetImageURL,
  clearMedia,
  getAssetMetadata,
  getDecodedImage,
  invalidateMedia,
  peekAssetImage,
  peekAssetMetadata,
} from "./media";
vi.mock("./api", () => ({ api: vi.fn() }));
class TestImage {
  naturalWidth = 100;
  naturalHeight = 100;
  onload?: () => void;
  onerror?: () => void;
  set src(url: string) {
    this.naturalWidth = this.naturalHeight = Number(
      /bucket=(\d+)/.exec(url)?.[1] || 1000,
    );
    queueMicrotask(() => this.onload?.());
  }
  decode() {
    return Promise.resolve();
  }
}
beforeEach(() => {
  clearMedia();
  vi.stubGlobal("Image", TestImage);
  vi.mocked(api).mockReset();
});
afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});
it("reuses decoded images synchronously, selecting a suitable size or a temporary smaller preview", async () => {
  const small = await getDecodedImage("/api/v1/assets/a?bucket=128");
  const large = await getDecodedImage("/api/v1/assets/a?bucket=512");
  expect(peekAssetImage("a", 100, 100)).toBe(small);
  expect(peekAssetImage("a", 200, 200)).toBe(large);
  expect(peekAssetImage("a", 1000, 1000)).toBe(large);
  expect(peekAssetImage("another", 100, 100)).toBeUndefined();
  expect(await getDecodedImage("/api/v1/assets/a?bucket=512")).toBe(large);
  clearMedia();
  expect(peekAssetImage("a", 100, 100)).toBeUndefined();
});
it("keeps recently displayed images when the bounded decoded cache fills", async () => {
  const first = await getDecodedImage("/api/v1/assets/recent?bucket=128");
  for (let i = 0; i < 63; i++)
    await getDecodedImage(`/api/v1/assets/item-${i}?bucket=128`);
  expect(peekAssetImage("recent", 100, 100)).toBe(first);
  await getDecodedImage("/api/v1/assets/new?bucket=128");
  expect(peekAssetImage("recent", 100, 100)).toBe(first);
  expect(peekAssetImage("item-0", 100, 100)).toBeUndefined();
});
it("retains known image URLs while stale metadata refreshes, and invalidates them on media events", async () => {
  const clock = vi.spyOn(Date, "now").mockReturnValue(0);
  const meta = { width: 1000, height: 1000, variants: [128, 512] };
  vi.mocked(api).mockResolvedValueOnce(meta);
  await getAssetMetadata("a");
  clock.mockReturnValue(6000);
  let resolve!: (value: unknown) => void;
  vi.mocked(api).mockImplementationOnce(
    () =>
      new Promise((done) => {
        resolve = done;
      }) as ReturnType<typeof api>,
  );
  const refreshing = getAssetMetadata("a");
  expect(peekAssetMetadata("a")).toEqual(meta);
  expect(assetImageURL("a", meta, 100, 100)).toBe(
    "/api/v1/assets/a?bucket=128",
  );
  resolve({ ...meta, variants: [100, 128, 512] });
  await refreshing;
  expect(peekAssetMetadata("a")?.variants).toContain(100);
  invalidateMedia("a");
  expect(peekAssetMetadata("a")).toBeUndefined();
});
