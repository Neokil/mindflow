import { useRef, useState } from "react";
import { ErrorMessage, Modal, Spinner } from "../../ui";
import {
  IMPORT_EXTENSIONS,
  importFile,
  importName,
} from "../transfer/importFiles";
import type { ImportResult } from "../transfer/importFiles";
type Item = { file: File; result?: ImportResult; error?: string };
export default function ImportWorkspaceDialog({
  close,
  open,
  workspaceId,
}: {
  close: () => void;
  open: (id: string, pageId?: string) => Promise<void>;
  workspaceId?: string;
}) {
  const [items, setItems] = useState<Item[]>([]),
    [name, setName] = useState("");
  const [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [progress, setProgress] = useState("");
  const stop = useRef(false);
  const completed = items.filter((item) => item.result);
  async function submit() {
    if (!items.length || busy) return;
    setBusy(true);
    setError("");
    stop.current = false;
    try {
      for (const [index, item] of items.entries()) {
        if (stop.current) break;
        if (item.result) continue;
        const update = (value: Partial<Item>) =>
          setItems((entries) =>
            entries.map((entry, i) =>
              i === index ? { ...entry, ...value } : entry,
            ),
          );
        update({ error: undefined });
        try {
          const result = await importFile(
            item.file,
            items.length === 1 ? name : importName(item.file),
            workspaceId,
            (message) =>
              setProgress(
                `${index + 1}/${items.length}: ${item.file.name} — ${message}`,
              ),
          );
          update({ result });
        } catch (err) {
          update({ error: (err as Error).message });
        }
      }
    } finally {
      setBusy(false);
      setProgress("");
    }
  }
  async function openResult(result: ImportResult) {
    setBusy(true);
    setError("");
    try {
      await open(result.id, result.pageId);
    } catch (error) {
      setError((error as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal
      title={workspaceId ? "Import pages" : "Import workspace"}
      close={() => {
        if (!busy) close();
      }}
    >
      <p className="muted">
        {workspaceId
          ? "Append pages from mindflow, Goodnotes, or PDF files. Existing pages stay unchanged."
          : "Import mindflow, Butterfly, Goodnotes, or PDF files as new private workspaces. Select multiple files to migrate a collection."}{" "}
        Each file is independent; failed files do not undo successful imports.
        PDFs become page images that you can annotate.
      </p>
      {completed.length === 0 && (
        <form
          onSubmit={(event) => {
            event.preventDefault();
            void submit();
          }}
        >
          <label>
            Workspace exports
            <input
              aria-label="Workspace export"
              type="file"
              accept={
                workspaceId ? ".mindflow,.pdf,.goodnotes" : IMPORT_EXTENSIONS
              }
              multiple
              required
              disabled={busy}
              onChange={(event) => {
                const files = [...(event.target.files || [])];
                if (!files.length) return;
                if (
                  files.length > 20 ||
                  files.reduce((sum, file) => sum + file.size, 0) >
                    512 * 1024 * 1024
                ) {
                  setError("Choose up to 20 files totaling at most 512 MiB.");
                  return;
                }
                setItems(files.map((file) => ({ file })));
                setName(importName(files[0]));
                setError("");
              }}
            />
          </label>
          {!workspaceId && items.length <= 1 && (
            <label>
              Workspace name
              <input
                aria-label="Imported workspace name"
                value={name}
                maxLength={200}
                required
                disabled={busy}
                onChange={(event) => setName(event.target.value)}
              />
            </label>
          )}
          <div className="form-actions">
            <button
              className="secondary"
              type="button"
              disabled={busy}
              onClick={close}
            >
              Cancel
            </button>
            <button className="primary" disabled={busy || !items.length}>
              {busy ? (
                <>
                  <Spinner /> Importing…
                </>
              ) : (
                "Import"
              )}
            </button>
          </div>
        </form>
      )}
      {items.length > 0 && (
        <ul className="import-results">
          {items.map((item, index) => (
            <li key={index}>
              <strong>{item.file.name}</strong>
              {item.result ? (
                <>
                  <p role="status">
                    Imported {item.result.pageCount} pages and{" "}
                    {item.result.elementCount} elements{" "}
                    {workspaceId
                      ? "into this workspace."
                      : "into a private workspace."}
                  </p>
                  {item.result.warnings.length > 0 && (
                    <>
                      <h3>Conversion notes</h3>
                      <ul className="import-notes">
                        {item.result.warnings.map((note) => (
                          <li key={note}>{note}</li>
                        ))}
                      </ul>
                    </>
                  )}
                  <button
                    className="secondary"
                    disabled={busy}
                    onClick={() => void openResult(item.result!)}
                  >
                    {workspaceId ? "Open imported page" : "Open workspace"}
                  </button>
                </>
              ) : item.error ? (
                <p className="error" role="alert">
                  {item.error}
                </p>
              ) : (
                <p className="muted">Waiting to import</p>
              )}
            </li>
          ))}
        </ul>
      )}
      {busy && (
        <>
          <p role="status">{progress}</p>
          <button
            className="secondary"
            onClick={() => {
              stop.current = true;
            }}
          >
            Stop after current file
          </button>
        </>
      )}
      {!busy && completed.length > 0 && (
        <div className="form-actions">
          <button className="secondary" onClick={close}>
            Done
          </button>
          {completed.length < items.length && (
            <button className="primary" onClick={() => void submit()}>
              Import remaining files
            </button>
          )}
        </div>
      )}
      <ErrorMessage message={error} />
    </Modal>
  );
}
