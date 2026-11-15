import { apiFile } from "../../../api";
import { clone, type CanvasDocument } from "../../../model";
import { openPDF, renderPDFPage } from "../../pdf/pdf";
import { safeFilename } from "../exportGeometry";
import { portableFile, readZip, type Notebook } from "./archive";
import { fromNative, replaceWithImage } from "./write";
async function convert<T>(message: unknown): Promise<T> {
  const worker = new Worker(new URL("./worker.ts", import.meta.url), {
    type: "module",
  });
  try {
    return await new Promise<T>((resolve, reject) => {
      const timer = setTimeout(
        () =>
          reject(
            new Error("Goodnotes conversion timed out. Try a smaller file."),
          ),
        120000,
      );
      const done = (value: T | undefined, error?: string) => {
        clearTimeout(timer);
        if (error) reject(new Error(error));
        else resolve(value!);
      };
      worker.onmessage = (event) => done(event.data.result, event.data.error);
      worker.onerror = () =>
        done(undefined, "Goodnotes conversion failed. Try a smaller file.");
      const payload = message as { data?: Uint8Array; notebook?: Notebook };
      const buffers = new Set<ArrayBuffer>();
      if (payload.data?.buffer instanceof ArrayBuffer)
        buffers.add(payload.data.buffer);
      for (const asset of payload.notebook?.assets || [])
        if (asset.data.buffer instanceof ArrayBuffer)
          buffers.add(asset.data.buffer);
      worker.postMessage(message, [...buffers]);
    });
  } finally {
    worker.terminate();
  }
}
export async function importGoodnotes(
  file: File,
  name: string,
  progress: (message: string) => void,
): Promise<{ blob: Blob; warnings: string[] }> {
  progress("Reading Goodnotes notebook…");
  const notebook = await convert<Notebook>({
    mode: "read",
    data: new Uint8Array(await file.arrayBuffer()),
  });
  notebook.name = name;
  let expanded = 0;
  for (const asset of notebook.assets) {
    if (asset.mime === "application/pdf") {
      progress("Rendering embedded Goodnotes PDF page…");
      const signal = new AbortController().signal,
        pdf = await openPDF(asset.data, signal);
      try {
        const pageNumber = asset.pdfPage || 1;
        const viewport = (await pdf.getPage(pageNumber)).getViewport({
          scale: 1,
        });
        const rendered = await renderPDFPage(pdf, pageNumber, signal);
        asset.data = new Uint8Array(await rendered.blob.arrayBuffer());
        asset.mime = "image/png";
        if (asset.background)
          for (const page of notebook.pages)
            for (const element of page.document.elements)
              if (element.assetId === asset.id) {
                // Native Goodnotes placement and PDF dimensions both use points.
                element.width = viewport.width;
                element.height = viewport.height;
                element.x = element.width / 2;
                element.y = element.height / 2;
              }
        notebook.warnings.push(
          "Embedded PDF pages are imported as images for annotation; keep the original Goodnotes file for embedded PDF data.",
        );
      } finally {
        await pdf.loadingTask.destroy();
      }
    }
    expanded += asset.data.length;
    if (expanded > 100 * 1024 * 1024)
      throw new Error(
        "Converted Goodnotes media exceeds 100 MiB. Split the notebook before importing.",
      );
  }
  return {
    blob: portableFile(notebook),
    warnings: [...new Set(notebook.warnings)],
  };
}
export async function exportGoodnotes(
  workspaceId: string,
  selection: { pageIds: string[] },
  progress: (message: string) => void,
) {
  const raw = await apiFile(`/workspaces/${workspaceId}/export`, {
    ...selection,
    includeTemplates: false,
  });
  const files = readZip(new Uint8Array(await raw.arrayBuffer()));
  const notebook = fromNative(files);
  const notes = new Set<string>();
  let generatedBytes = 0;
  const deadline = Date.now() + 120000;
  const retain = (asset: Notebook["assets"][number]) => {
    generatedBytes += asset.data.length;
    if (
      notebook.assets.length >= 1000 ||
      generatedBytes > 100 * 1024 * 1024 ||
      Date.now() > deadline
    )
      throw new Error(
        "Goodnotes conversion exceeds its media or time limit. Export fewer pages.",
      );
    notebook.assets.push(asset);
  };
  for (const page of notebook.pages)
    for (let index = 0; index < page.document.elements.length; index++) {
      if (Date.now() > deadline)
        throw new Error("Goodnotes conversion timed out. Export fewer pages.");
      const element = page.document.elements[index];
      if (element.type === "pdf") {
        const id = `preview-${notebook.assets.length}`;
        retain({
          id,
          mime: "image/png",
          data: files[`assets/${element.assetId}.png`],
        });
        page.document.elements[index] = replaceWithImage(element, id);
        notes.add(
          "Embedded PDFs become first-page preview images. Export the PDF separately to retain all pages.",
        );
      } else if (
        element.type === "shape" ||
        (element.type === "text" && element.rotation !== 0)
      ) {
        progress(
          "Converting unsupported Goodnotes objects to individual images…",
        );
        const document: CanvasDocument = {
          ...page.document,
          settings: { ...page.document.settings },
          elements: [clone(element)],
        };
        const region = (await import("../exportGeometry")).exportBounds(
          document,
        );
        const blob = await (
          await import("../renderPage")
        ).renderPage(document, {
          region,
          resolution: 2048,
          background: false,
          grid: false,
          backgroundColor: "transparent",
        });
        const id = `raster-${notebook.assets.length}`;
        retain({
          id,
          mime: "image/png",
          data: new Uint8Array(await blob.arrayBuffer()),
        });
        page.document.elements[index] = {
          ...replaceWithImage(element, id),
          x: region.x + region.width / 2,
          y: region.y + region.height / 2,
          width: region.width,
          height: region.height,
          rotation: 0,
        };
        notes.add(
          "Shapes and rotated text become individual images; other text, ink, and images remain editable.",
        );
      }
    }
  progress("Writing native Goodnotes notebook…");
  const result = await convert<{ data: Uint8Array; warnings: string[] }>({
    mode: "write",
    notebook,
  });
  return {
    blob: new Blob([new Uint8Array(result.data)], { type: "application/zip" }),
    name: `${safeFilename(notebook.name)}.goodnotes`,
    warnings: [...notes, ...result.warnings],
  };
}
