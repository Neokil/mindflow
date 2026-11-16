import { api } from "./api";
export type AssetMetadata = {
  width: number;
  height: number;
  variants: number[];
};
const metadata = new Map<
  string,
  { at: number; promise: Promise<AssetMetadata>; value?: AssetMetadata }
>();
const decoded = new Map<
  string,
  {
    promise: Promise<HTMLImageElement>;
    bytes: number;
    image?: HTMLImageElement;
  }
>();
let decodedBytes = 0;
export function getAssetMetadata(id: string) {
  const entry = metadata.get(id);
  if (entry && Date.now() - entry.at < 5000) return entry.promise;
  const promise = api<AssetMetadata>(`/assets/${id}/meta`);
  const next = { at: Date.now(), promise, value: entry?.value };
  metadata.set(id, next);
  void promise.then(
    (value) => {
      next.value = value;
    },
    () => {},
  );
  promise.catch(() => {
    if (metadata.get(id)?.promise === promise) metadata.delete(id);
  });
  return promise;
}
// Last-known immutable dimensions/variant URLs remain useful while metadata refreshes.
export function peekAssetMetadata(id: string) {
  return metadata.get(id)?.value;
}

export function assetImageURL(
  id: string,
  meta: AssetMetadata,
  width: number,
  height: number,
) {
  const need =
    Math.max(width / meta.width, height / meta.height) *
    Math.max(meta.width, meta.height);
  const bucket = meta.variants.find((value) => value >= need);
  return `/api/v1/assets/${id}${bucket ? `?bucket=${bucket}` : ""}`;
}

// Reuse a decoded representation on the very first frame after a page switch.
// Prefer the smallest adequate version, or the largest available while upgrading.
export function peekAssetImage(id: string, width: number, height: number) {
  const prefix = `/api/v1/assets/${id}`;
  let selected:
    { url: string; image: HTMLImageElement; scale: number } | undefined;
  for (const [url, entry] of decoded) {
    if ((url !== prefix && !url.startsWith(`${prefix}?`)) || !entry.image)
      continue;
    const image = entry.image;
    const scale = Math.min(
      image.naturalWidth / width,
      image.naturalHeight / height,
    );
    if (
      !selected ||
      (scale >= 1 && (selected.scale < 1 || scale < selected.scale)) ||
      (scale < 1 && selected.scale < scale)
    ) {
      selected = { url, image, scale };
    }
  }
  if (selected) {
    const entry = decoded.get(selected.url)!;
    decoded.delete(selected.url);
    decoded.set(selected.url, entry);
  }
  return selected?.image;
}

export function invalidateMedia(id: string) {
  metadata.delete(id);
}
export function getDecodedImage(url: string) {
  const cached = decoded.get(url);
  if (cached) {
    decoded.delete(url);
    decoded.set(url, cached);
    return cached.promise;
  }
  const entry = {
    promise: undefined as unknown as Promise<HTMLImageElement>,
    bytes: 0,
    image: undefined as HTMLImageElement | undefined,
  };
  entry.promise = new Promise<HTMLImageElement>((resolve, reject) => {
    const image = new Image();
    const loaded = () => {
      if (decoded.get(url) !== entry) {
        resolve(image);
        return;
      }
      entry.image = image;
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
    image.onload = () => {
      void image.decode().then(loaded, failed);
    };
    const failed = () => {
      if (decoded.get(url) === entry) decoded.delete(url);
      reject(new Error("Image unavailable"));
    };
    image.onerror = failed;
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
