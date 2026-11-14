import { useEffect, useState } from "react";
import { Trash2 } from "lucide-react";
import { api } from "../../api";
import type { CanvasElement } from "../../model";
import { PageSession } from "../../session";
import { ErrorMessage, Modal, Spinner } from "../../ui";
import { instantiateTemplate } from "./templates";

type Summary = { id: string; name: string; elementCount: number };
export default function TemplateLibrary({
  session,
  selection,
  creating,
  insert,
  close,
}: {
  session: PageSession;
  selection: CanvasElement[];
  creating: boolean;
  insert: (elements: CanvasElement[]) => void;
  close: () => void;
}) {
  const [items, setItems] = useState<Summary[]>(),
    [name, setName] = useState(""),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const path = `/workspaces/${session.workspaceId}/templates`;
  useEffect(() => {
    let cancelled = false;
    api<Summary[]>(path)
      .then((items) => {
        if (!cancelled) setItems(items);
      })
      .catch((error) => {
        if (!cancelled) setError(error.message);
      });
    return () => {
      cancelled = true;
    };
  }, [path]);
  async function save() {
    setBusy(true);
    setError("");
    try {
      const snapshot = await session.prepare({
        ...session.doc,
        elements: selection,
      });
      await api(path, "POST", {
        name,
        elements: instantiateTemplate(snapshot.elements),
      });
      close();
    } catch (error) {
      setError((error as Error).message);
    } finally {
      setBusy(false);
    }
  }
  async function place(item: Summary) {
    setBusy(true);
    setError("");
    try {
      const snapshot = await api<{ elements: CanvasElement[] }>(
        `${path}/${item.id}`,
      );
      if (!session.editable)
        throw new Error("Take editor control before inserting a template.");
      insert(instantiateTemplate(snapshot.elements));
      close();
    } catch (error) {
      setError((error as Error).message);
    } finally {
      setBusy(false);
    }
  }
  async function remove(item: Summary) {
    if (
      !confirm(
        `Delete template “${item.name}”? Already inserted elements will remain unchanged.`,
      )
    )
      return;
    setBusy(true);
    setError("");
    try {
      await api(`${path}/${item.id}`, "DELETE");
      setItems((items) => items?.filter((entry) => entry.id !== item.id));
    } catch (error) {
      setError((error as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal
      title={creating ? "Save as template" : "Workspace templates"}
      close={() => {
        if (!busy) close();
      }}
    >
      {creating ? (
        <form
          onSubmit={(event) => {
            event.preventDefault();
            void save();
          }}
        >
          <p className="muted">
            Save {selection.length} selected elements for reuse on any page in
            this workspace. Inserted copies can be edited individually.
          </p>
          <label>
            Template name
            <input
              aria-label="Template name"
              required
              maxLength={200}
              autoFocus
              disabled={busy}
              value={name}
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
            <button
              className="primary"
              disabled={
                busy ||
                !name.trim() ||
                !selection.length ||
                selection.length > 500 ||
                !session.canWrite
              }
            >
              {busy ? <Spinner /> : "Save template"}
            </button>
          </div>
        </form>
      ) : (
        <>
          <p className="muted">
            Choose a template, then click on the canvas to place an editable
            copy. Templates are shared with this workspace.
          </p>
          {!session.editable && (
            <p className="muted">Take editor control to insert a template.</p>
          )}
          {!items && !error && <Spinner />}
          {items?.length === 0 && (
            <p>
              No templates yet. Select elements and choose “Save as template”.
            </p>
          )}
          <ul className="template-list">
            {items?.map((item) => (
              <li key={item.id}>
                <div>
                  <strong>{item.name}</strong>
                  <span className="muted">{item.elementCount} elements</span>
                </div>
                <button
                  className="secondary compact"
                  aria-label={`Insert ${item.name}`}
                  disabled={busy || !session.editable}
                  onClick={() => void place(item)}
                  data-tooltip={`Insert ${item.name}`}
                >
                  Insert
                </button>
                {session.canWrite && (
                  <button
                    className="icon-button danger"
                    aria-label={`Delete template ${item.name}`}
                    disabled={busy}
                    onClick={() => void remove(item)}
                    data-tooltip={`Delete template ${item.name}`}
                  >
                    <Trash2 size={16} />
                  </button>
                )}
              </li>
            ))}
          </ul>
        </>
      )}
      <ErrorMessage message={error} />
    </Modal>
  );
}
