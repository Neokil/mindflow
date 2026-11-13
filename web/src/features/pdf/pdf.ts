import type { PDFDocumentProxy } from "pdfjs-dist";
export const MAX_PDF_BYTES = 50 * 1024 * 1024;
export const MAX_PDF_PAGES = 100;
const TIMEOUT = 30000;
export async function openPDF(
  data: Uint8Array,
  signal: AbortSignal,
): Promise<PDFDocumentProxy> {
  const pdfjs = await import("pdfjs-dist/legacy/build/pdf.mjs");
  const worker = await import("pdfjs-dist/legacy/build/pdf.worker.min.mjs?url");
  if (signal.aborted) throw new Error("PDF loading cancelled.");
  pdfjs.GlobalWorkerOptions.workerSrc = worker.default;
  const task = pdfjs.getDocument({
    data,
    disableWorkerRendering: true,
    enableXfa: false,
    useWasm: false,
    disableFontFace: true,
    useSystemFonts: true,
    maxImageSize: 16000000,
    canvasMaxAreaInBytes: 32 * 1024 * 1024,
    cMapUrl: "/pdfjs/cmaps/",
    cMapPacked: true,
    standardFontDataUrl: "/pdfjs/standard_fonts/",
    iccUrl: "/pdfjs/iccs/",
    stopAtErrors: true,
  });
  let timeout = false;
  const stop = () => {
    void task.destroy();
  };
  signal.addEventListener("abort", stop, { once: true });
  const timer = setTimeout(() => {
    timeout = true;
    stop();
  }, TIMEOUT);
  try {
    const doc = await task.promise;
    if (doc.numPages > MAX_PDF_PAGES) {
      await doc.loadingTask.destroy();
      throw new Error("PDFs can contain at most 100 pages.");
    }
    return doc;
  } catch (err) {
    if (timeout) throw new Error("PDF loading timed out.", { cause: err });
    if (err instanceof Error && err.name === "PasswordException")
      throw new Error(
        "Password-protected PDFs are not supported. Export an unlocked copy first.",
        { cause: err },
      );
    throw err;
  } finally {
    clearTimeout(timer);
    signal.removeEventListener("abort", stop);
  }
}
export async function renderPDFPage(
  pdf: PDFDocumentProxy,
  pageNumber: number,
  signal: AbortSignal,
  edge = 2400,
) {
  const page = await pdf.getPage(pageNumber);
  const base = page.getViewport({ scale: 1 });
  if (
    !Number.isFinite(base.width + base.height) ||
    base.width <= 0 ||
    base.height <= 0 ||
    Math.max(base.width, base.height) > 1000000
  )
    throw new Error("PDF page dimensions are unsupported.");
  const viewport = page.getViewport({
    scale: Math.min(2, edge / Math.max(base.width, base.height)),
  });
  const canvas = document.createElement("canvas");
  canvas.width = Math.max(1, Math.ceil(viewport.width));
  canvas.height = Math.max(1, Math.ceil(viewport.height));
  const task = page.render({
    canvas,
    viewport,
    background: "#ffffff",
    annotationMode: 1,
  });
  const cancel = () => task.cancel();
  signal.addEventListener("abort", cancel, { once: true });
  const timer = setTimeout(cancel, TIMEOUT);
  try {
    if (signal.aborted) throw new Error("PDF rendering cancelled.");
    await task.promise;
    const blob = await new Promise<Blob>((resolve, reject) =>
      canvas.toBlob(
        (value) =>
          value
            ? resolve(value)
            : reject(new Error("Could not create PDF page image.")),
        "image/png",
      ),
    );
    const width = Math.min(800, base.width);
    return { blob, width, height: (base.height * width) / base.width };
  } finally {
    clearTimeout(timer);
    signal.removeEventListener("abort", cancel);
    canvas.width = 0;
    canvas.height = 0;
    page.cleanup();
  }
}
