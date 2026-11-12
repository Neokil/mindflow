import { api } from "./api";
export type AssetMetadata = {
  width: number;
  height: number;
  variants: number[];
};
const metadata = new Map<
  string,
  { at: number; promise: Promise<AssetMetadata> }
>();
const decoded = new Map<
  string,
  { promise: Promise<HTMLImageElement>; bytes: number }
>();
let decodedBytes = 0;
export function getAssetMetadata(id: string) {
  const entry = metadata.get(id);
  if (entry && Date.now() - entry.at < 5000) return entry.promise;
  const promise = api<AssetMetadata>(`/assets/${id}/meta`);
  metadata.set(id, { at: Date.now(), promise });
  promise.catch(() => {
    if (metadata.get(id)?.promise === promise) metadata.delete(id);
  });
  return promise;
}
export function invalidateMedia(id: string) {
  metadata.delete(id);
}
export function getDecodedImage(url: string) {
  const cached = decoded.get(url);
  if (cached) return cached.promise;
  const entry = {
    promise: undefined as unknown as Promise<HTMLImageElement>,
    bytes: 0,
  };
  entry.promise = new Promise<HTMLImageElement>((resolve, reject) => {
    const image = new Image();
    image.onload = () => {
      if (decoded.get(url) !== entry) {
        resolve(image);
        return;
      }
      entry.bytes = image.naturalWidth * image.naturalHeight * 4;
      decodedBytes += entry.bytes;
      // Bound retained decoded images, independently from the images currently
      // held by visible renderer nodes. In-flight requests for a URL are shared.
      while (decodedBytes > 64 * 1024 * 1024 || decoded.size > 64) {
        const key = decoded.keys().next().value;
        if (!key) break;
        decodedBytes -= decoded.get(key)!.bytes;
        decoded.delete(key);
      }
      resolve(image);
    };
    image.onerror = () => {
      decoded.delete(url);
      reject(new Error("Image unavailable"));
    };
    image.src = url;
  });
  decoded.set(url, entry);
  return entry.promise;
}
export function clearMedia() {
  metadata.clear();
  decoded.clear();
  decodedBytes = 0;
}
