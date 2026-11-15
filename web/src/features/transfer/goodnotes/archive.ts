import { unzipSync, zipSync } from "fflate";
import type { CanvasDocument } from "../../../model";
import { invalid, MAX_EXPANDED } from "./wire";
export type PortablePage = {
  id: string;
  name: string;
  document: CanvasDocument;
};
export type PortableAsset = {
  id: string;
  mime: string;
  data: Uint8Array;
  pdfPage?: number;
  background?: boolean;
};
export type Notebook = {
  name: string;
  pages: PortablePage[];
  assets: PortableAsset[];
  warnings: string[];
};
export function readZip(data: Uint8Array): Record<string, Uint8Array> {
  if (data.length > 64 * 1024 * 1024) invalid();
  let expanded = 0,
    count = 0;
  const names = new Set<string>();
  return unzipSync(data, {
    filter: (entry) => {
      if (
        ++count > 5000 ||
        names.has(entry.name) ||
        entry.name.startsWith("/") ||
        entry.name.split("/").some((p) => p === "..")
      )
        invalid();
      names.add(entry.name);
      expanded += entry.originalSize;
      if (entry.originalSize > 64 * 1024 * 1024 || expanded > MAX_EXPANDED)
        invalid();
      return !entry.name.endsWith("/");
    },
  });
}
export function writeZip(files: Record<string, Uint8Array>) {
  if (
    Object.values(files).reduce((n, data) => n + data.length, 0) > MAX_EXPANDED
  )
    invalid();
  const output = zipSync(files, { level: 6 });
  if (output.length > 64 * 1024 * 1024) invalid();
  return output;
}
export function portableFile(notebook: Notebook): Blob {
  const manifest = {
    format: "mindflow",
    version: 1,
    kind: "workspace",
    name: notebook.name,
    pages: notebook.pages,
    templates: [],
    assets: notebook.assets.map((asset) => ({
      id: asset.id,
      mime: asset.mime,
    })),
  };
  const metadata = new TextEncoder().encode(JSON.stringify(manifest));
  if (metadata.length > 32 * 1024 * 1024)
    throw new Error(
      "Converted notebook exceeds 32 MiB of canvas data. Split the source notebook into smaller files.",
    );
  const files: Record<string, Uint8Array> = { "manifest.json": metadata };
  for (const asset of notebook.assets)
    files[`assets/${asset.id}.bin`] = asset.data;
  return new Blob([new Uint8Array(writeZip(files))], {
    type: "application/vnd.mindflow+zip",
  });
}
