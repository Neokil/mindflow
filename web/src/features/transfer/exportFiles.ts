import type { CanvasDocument } from "../../model";
import { exportSize, safeFilename } from "./exportGeometry";
import { renderPage, type RenderOptions } from "./renderPage";
export type Snapshot = {
  name: string;
  pages: { name: string; document: CanvasDocument }[];
};
export type ExportFile = { blob: Blob; name: string };
export async function visualExport(
  snapshot: Snapshot,
  format: "pdf" | "png",
  options: RenderOptions,
  progress: (text: string) => void,
): Promise<ExportFile> {
  if (!snapshot.pages.length)
    throw new Error("Add a page before exporting a PDF or image.");
  const pixels = snapshot.pages.reduce((sum, p) => {
    const size = exportSize(p.document, options.resolution);
    return sum + size.width * size.height;
  }, 0);
  if (pixels > 128 * 1024 * 1024)
    throw new Error(
      "This export is too large. Choose fewer pages or a lower resolution.",
    );
  const name = safeFilename(snapshot.name);
  const pdf =
    format === "pdf"
      ? await (await import("pdf-lib")).PDFDocument.create()
      : undefined;
  const files: Record<string, Uint8Array> = {};
  let bytes = 0;
  for (const [index, page] of snapshot.pages.entries()) {
    progress(`Rendering page ${index + 1} of ${snapshot.pages.length}…`);
    const blob = await renderPage(page.document, options);
    bytes += blob.size;
    if (bytes > 128 * 1024 * 1024)
      throw new Error("This export exceeds 128 MiB. Export fewer pages.");
    if (format === "png" && snapshot.pages.length === 1)
      return { blob, name: `${safeFilename(page.name)}.png` };
    const data = new Uint8Array(await blob.arrayBuffer());
    if (pdf) {
      const image = await pdf.embedPng(data),
        bounds = exportSize(page.document, options.resolution).bounds;
      const factor = Math.min(
        0.75,
        14400 / Math.max(bounds.width, bounds.height),
      );
      const width = bounds.width * factor,
        height = bounds.height * factor;
      pdf
        .addPage([width, height])
        .drawImage(image, { x: 0, y: 0, width, height });
    } else
      files[
        `${String(index + 1).padStart(3, "0")}-${safeFilename(page.name)}.png`
      ] = data;
    // Let the browser paint progress between pages.
    await new Promise((resolve) => setTimeout(resolve, 0));
  }
  if (pdf)
    return {
      blob: new Blob([new Uint8Array(await pdf.save())], {
        type: "application/pdf",
      }),
      name: `${name}.pdf`,
    };
  const { zipSync } = await import("fflate");
  return {
    blob: new Blob([new Uint8Array(zipSync(files, { level: 0 }))], {
      type: "application/zip",
    }),
    name: `${name}-images.zip`,
  };
}
export function downloadFile(file: ExportFile) {
  const url = URL.createObjectURL(file.blob),
    anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = file.name;
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
  setTimeout(() => URL.revokeObjectURL(url), 60000);
}
