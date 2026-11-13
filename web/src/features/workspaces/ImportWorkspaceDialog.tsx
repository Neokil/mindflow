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
}: {
  close: () => void;
  open: (id: string) => Promise<void>;
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
        await api<Imported>("/workspaces/import/butterfly", "POST", form),
      );
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal
      title="Import workspace"
      close={() => {
        if (!busy) close();
      }}
    >
      {result ? (
        <>
          <p role="status">
            Imported {result.pageCount} pages and {result.elementCount} elements
            into a private workspace.
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
                  await open(result.id);
                } catch (err) {
                  setError((err as Error).message);
                } finally {
                  setBusy(false);
                }
              }}
            >
              Open workspace
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
            Import a Butterfly .bfly export as a new private workspace. The
            original file stays unchanged.
          </p>
          <label>
            Butterfly export (.bfly)
            <input
              aria-label="Butterfly export"
              type="file"
              accept=".bfly,application/zip"
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
                if (selected) setName(selected.name.replace(/\.bfly$/i, ""));
              }}
            />
          </label>
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
