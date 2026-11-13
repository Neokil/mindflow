import { useState } from "react";
import { api } from "../../api";
import { ErrorMessage, Modal, Spinner } from "../../ui";

type Imported = {
  id: string;
  pageId: string;
  pageCount: number;
  elementCount: number;
  warnings: string[];
};
export default function ImportWorkspaceDialog({
  close,
  open,
  workspaceId,
}: {
  close: () => void;
  open: (id: string, pageId?: string) => Promise<void>;
  workspaceId?: string;
}) {
  const [file, setFile] = useState<File>(),
    [name, setName] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [result, setResult] = useState<Imported>();
  async function submit() {
    if (!file) return;
    setBusy(true);
    setError("");
    try {
      const form = new FormData();
      form.append("file", file);
      form.append("name", name);
      setResult(
        await api<Imported>(
          workspaceId
            ? `/workspaces/${workspaceId}/import/mindflow`
            : `/workspaces/import/${file.name.toLowerCase().endsWith(".bfly") ? "butterfly" : "mindflow"}`,
          "POST",
          form,
        ),
      );
    } catch (err) {
      setError((err as Error).message);
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
      {result ? (
        <>
          <p role="status">
            Imported {result.pageCount} pages and {result.elementCount} elements{" "}
            {workspaceId ? "into this workspace." : "into a private workspace."}
          </p>
          {result.warnings.length > 0 && (
            <>
              <h3>Conversion notes</h3>
              <ul className="import-notes">
                {result.warnings.map((note) => (
                  <li key={note}>{note}</li>
                ))}
              </ul>
            </>
          )}
          <div className="form-actions">
            <button className="secondary" disabled={busy} onClick={close}>
              Done
            </button>
            <button
              className="primary"
              disabled={busy}
              onClick={async () => {
                setBusy(true);
                try {
                  await open(result.id, result.pageId);
                } catch (err) {
                  setError((err as Error).message);
                } finally {
                  setBusy(false);
                }
              }}
            >
              {workspaceId ? "Open imported page" : "Open workspace"}
            </button>
          </div>
        </>
      ) : (
        <form
          onSubmit={(event) => {
            event.preventDefault();
            void submit();
          }}
        >
          <p className="muted">
            {workspaceId
              ? "Append pages from a .mindflow file. Existing pages stay unchanged."
              : "Import a mindflow .mindflow or Butterfly .bfly export as a new private workspace."}
          </p>
          <label>
            {workspaceId
              ? "mindflow export (.mindflow)"
              : "Workspace export (.mindflow or .bfly)"}
            <input
              aria-label="Workspace export"
              type="file"
              accept={workspaceId ? ".mindflow" : ".mindflow,.bfly"}
              required
              disabled={busy}
              onChange={(event) => {
                const selected = event.target.files?.[0];
                if (selected && selected.size > 64 * 1024 * 1024) {
                  setError("Choose an export up to 64 MiB.");
                  setFile(undefined);
                  return;
                }
                setError("");
                setFile(selected);
                if (selected)
                  setName(selected.name.replace(/\.(bfly|mindflow)$/i, ""));
              }}
            />
          </label>
          {!workspaceId && (
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
            <button className="primary" disabled={busy || !file}>
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
      <ErrorMessage message={error} />
    </Modal>
  );
}
