import { optimisePageImages } from "../workspaces/optimiseImages";
import { refreshWorkspacePreview } from "../workspaces/preview";
import { useEffect, useRef, useState } from "react";
import type { PDFDocumentProxy } from "pdfjs-dist";
import { api } from "../../api";
import { MAX_ELEMENTS, uid } from "../../model";
import type { CanvasElement } from "../../model";
import { ErrorMessage, Modal, Spinner } from "../../ui";
import { MAX_PDF_BYTES, openPDF, renderPDFPage } from "./pdf";
type Mode = "pages" | "current" | "viewer";
export default function PDFImportDialog({
  workspaceId,
  elementCount,
  pageId,
  saveChanges,
  onElements,
  onPages,
  close,
}: {
  workspaceId: string;
  elementCount: number;
  pageId: string;
  saveChanges: () => Promise<void>;
  onElements: (elements: CanvasElement[]) => void;
  onPages: (id: string) => void;
  close: () => void;
}) {
  const [imported, setImported] = useState<string[]>();
  const [createdPages, setCreatedPages] = useState(false);
  const [file, setFile] = useState<File>();
  const [mode, setMode] = useState<Mode>("pages");
  const [busy, setBusy] = useState(false),
    [progress, setProgress] = useState(""),
    [error, setError] = useState("");
  const abort = useRef<AbortController | undefined>(undefined);
  useEffect(() => () => abort.current?.abort(), []);
  async function run() {
    if (!file || busy) return;
    setBusy(true);
    setError("");
    const controller = new AbortController();
    abort.current = controller;
    let pdf: PDFDocumentProxy | undefined;
    try {
      if (file.size > MAX_PDF_BYTES)
        throw new Error("Choose a PDF up to 50 MiB.");
      setProgress("Reading PDF…");
      pdf = await openPDF(
        new Uint8Array(await file.arrayBuffer()),
        controller.signal,
      );
      if (
        mode !== "pages" &&
        elementCount + (mode === "viewer" ? 1 : pdf.numPages) > MAX_ELEMENTS
      )
        throw new Error(
          "This import would exceed the page's 10,000 element limit.",
        );
      if (mode === "viewer") {
        const preview = await renderPDFPage(pdf, 1, controller.signal, 512);
        const form = new FormData();
        form.append("file", file);
        form.append("preview", preview.blob, "preview.png");
        const asset = await api<{ id: string }>(
          `/workspaces/${workspaceId}/pdf-assets`,
          "POST",
          form,
        );
        if (controller.signal.aborted) return;
        onElements([
          {
            id: uid(),
            type: "pdf",
            assetId: asset.id,
            x: 0,
            y: 0,
            width: preview.width,
            height: preview.height,
            rotation: 0,
          },
        ]);
      } else {
        const elements: CanvasElement[] = [],
          pages: {
            name: string;
            assetId: string;
            width: number;
            height: number;
          }[] = [];
        let top = 0;
        for (let index = 1; index <= pdf.numPages; index++) {
          setProgress(`Converting page ${index} of ${pdf.numPages}…`);
          const image = await renderPDFPage(pdf, index, controller.signal);
          const form = new FormData();
          form.append("file", image.blob, `page-${index}.png`);
          form.append("workspaceId", workspaceId);
          const asset = await api<{ id: string }>("/assets", "POST", form);
          if (controller.signal.aborted) return;
          pages.push({
            name: `${file.name.replace(/\.pdf$/i, "").slice(0, 140)} · ${index}`,
            assetId: asset.id,
            width: image.width,
            height: image.height,
          });
          elements.push({
            id: uid(),
            type: "image",
            assetId: asset.id,
            x: image.width / 2,
            y: top + image.height / 2,
            width: image.width,
            height: image.height,
            rotation: 0,
          });
          top += image.height + 40;
        }
        if (mode === "pages") {
          const result = await api<{ ids: string[] }>(
            `/workspaces/${workspaceId}/pdf-pages`,
            "POST",
            { pages },
          );
          setCreatedPages(true);
          setImported(result.ids);
          refreshWorkspacePreview(workspaceId);
        } else {
          onElements(elements);
          setImported([pageId]);
        }
      }
      if (mode === "viewer") close();
    } catch (err) {
      if (!controller.signal.aborted)
        setError((err as Error).message || "The PDF could not be imported.");
    } finally {
      await pdf?.loadingTask.destroy();
      if (!controller.signal.aborted) {
        setBusy(false);
        setProgress("");
      }
    }
  }
  if (imported) {
    const finish = () => {
      close();
      if (createdPages) onPages(imported[0]);
    };
    return (
      <Modal
        title="Optimise imported images?"
        close={() => {
          if (!busy) finish();
        }}
      >
        <p>
          Create smaller display versions that fit this layout. Original images
          are preserved.
        </p>
        {progress && <p role="status">{progress}</p>}
        <ErrorMessage message={error} />
        <div className="form-actions">
          <button className="secondary" disabled={busy} onClick={finish}>
            Not now
          </button>
          <button
            className="primary"
            disabled={busy}
            onClick={() => {
              setBusy(true);
              setError("");
              void (async () => {
                try {
                  if (!createdPages) await saveChanges();
                  for (const [index, id] of imported.entries()) {
                    setProgress(
                      `Optimising page ${index + 1}/${imported.length}…`,
                    );
                    await optimisePageImages(id);
                  }
                  refreshWorkspacePreview(workspaceId);
                  finish();
                } catch (error) {
                  setError((error as Error).message);
                } finally {
                  setBusy(false);
                }
              })();
            }}
          >
            {busy ? "Optimising…" : "Optimise images"}
          </button>
        </div>
      </Modal>
    );
  }
  return (
    <Modal
      title="Import PDF"
      close={() => {
        abort.current?.abort();
        close();
      }}
    >
      <label>
        PDF file
        <input
          type="file"
          aria-label="PDF file"
          accept="application/pdf,.pdf"
          disabled={busy}
          onChange={(event) => setFile(event.target.files?.[0])}
        />
      </label>
      <label>
        Placement
        <select
          aria-label="PDF placement"
          value={mode}
          disabled={busy}
          onChange={(event) => setMode(event.target.value as Mode)}
        >
          <option value="pages">One mindflow page per PDF page</option>
          <option value="current">All pages as images on this canvas</option>
          <option value="viewer">Embedded PDF viewer</option>
        </select>
      </label>
      <p className="muted">
        {mode === "viewer"
          ? "Keeps the original PDF as one canvas object. Open its read-only viewer to browse pages. PDF pages cannot be annotated in this mode."
          : "Pages are converted to images that you can move, resize and annotate with the pen or highlighter."}
      </p>
      <p className="muted">
        Up to 50 MiB and 100 pages. Password-protected PDFs are not supported.
      </p>
      {busy && (
        <p role="status">
          <Spinner /> {progress}
        </p>
      )}
      <ErrorMessage message={error} />
      <div className="form-actions">
        <button
          className="secondary"
          onClick={() => {
            abort.current?.abort();
            close();
          }}
        >
          Cancel
        </button>
        <button
          className="primary"
          disabled={!file || busy}
          onClick={() => void run()}
        >
          Import PDF
        </button>
      </div>
    </Modal>
  );
}
