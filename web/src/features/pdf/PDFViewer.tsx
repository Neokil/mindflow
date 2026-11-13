import { useEffect, useState } from "react";
import type { PDFDocumentProxy } from "pdfjs-dist";
import { Modal, ErrorMessage, Spinner } from "../../ui";
import { openPDF, renderPDFPage } from "./pdf";
export default function PDFViewer({
  assetId,
  close,
}: {
  assetId: string;
  close: () => void;
}) {
  const [pdf, setPDF] = useState<PDFDocumentProxy>();
  const [page, setPage] = useState(1),
    [image, setImage] = useState(""),
    [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController();
    let document: PDFDocumentProxy | undefined;
    void (async () => {
      try {
        const response = await fetch(`/api/v1/assets/${assetId}`, {
          signal: controller.signal,
        });
        if (!response.ok)
          throw new Error("PDF is unavailable or access was removed.");
        document = await openPDF(
          new Uint8Array(await response.arrayBuffer()),
          controller.signal,
        );
        if (controller.signal.aborted) await document.loadingTask.destroy();
        else setPDF(document);
      } catch (err) {
        if (!controller.signal.aborted) setError((err as Error).message);
      }
    })();
    return () => {
      controller.abort();
      void document?.loadingTask.destroy();
    };
  }, [assetId]);
  useEffect(() => {
    if (!pdf) return;
    const controller = new AbortController();
    let url = "";
    setImage("");
    setError("");
    void renderPDFPage(pdf, page, controller.signal)
      .then(({ blob }) => {
        if (!controller.signal.aborted) {
          url = URL.createObjectURL(blob);
          setImage(url);
        }
      })
      .catch((err) => {
        if (!controller.signal.aborted) setError(err.message);
      });
    return () => {
      controller.abort();
      if (url) URL.revokeObjectURL(url);
    };
  }, [pdf, page]);
  return (
    <Modal title="PDF viewer" close={close} wide>
      <p className="muted">
        Read-only PDF. Use an image import to annotate PDF pages.
      </p>
      <div className="pdf-navigation">
        <button
          className="secondary"
          disabled={!pdf || page === 1}
          onClick={() => setPage(page - 1)}
        >
          Previous page
        </button>
        <span aria-live="polite">
          {pdf ? `Page ${page} of ${pdf.numPages}` : "Loading PDF…"}
        </span>
        <button
          className="secondary"
          disabled={!pdf || page === pdf.numPages}
          onClick={() => setPage(page + 1)}
        >
          Next page
        </button>
      </div>
      <ErrorMessage message={error} />
      {!image && !error && <Spinner />}
      {image && (
        <img className="pdf-page" src={image} alt={`PDF page ${page}`} />
      )}
    </Modal>
  );
}
