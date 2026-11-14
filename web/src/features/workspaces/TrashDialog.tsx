import { useEffect, useState } from "react";
import { Folder, File, RotateCcw } from "lucide-react";
import { api } from "../../api";
import { Modal, ErrorMessage, Spinner } from "../../ui";
type Item = {
  kind: "workspace" | "page";
  id: string;
  name: string;
  workspaceId: string;
  workspaceName: string;
  deletedAt: number;
  purgeAt: number;
};
export default function TrashDialog({
  close,
  changed,
  workspaceId,
}: {
  close: () => void;
  changed: () => Promise<void>;
  workspaceId?: string;
}) {
  const [items, setItems] = useState<Item[]>(),
    [busy, setBusy] = useState<string>(),
    [error, setError] = useState("");
  const load = async () => setItems(await api<Item[]>("/trash"));
  useEffect(() => {
    void api<Item[]>("/trash")
      .then(setItems)
      .catch((e) => setError(e.message));
  }, []);
  async function restore(item: Item) {
    setBusy(item.id);
    setError("");
    try {
      await api(`/trash/${item.kind}s/${item.id}/restore`, "POST", {});
      await load();
      await changed();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(undefined);
    }
  }
  const visible = items?.filter(
    (item) => !workspaceId || item.workspaceId === workspaceId,
  );
  return (
    <Modal
      title="Trash"
      close={() => {
        if (!busy) close();
      }}
    >
      <p className="muted">
        Deleted pages and workspaces can be restored for 30 days. After that
        they are permanently deleted.
      </p>
      <ErrorMessage message={error} />
      {!items ? (
        <Spinner />
      ) : !visible?.length ? (
        <p className="muted">Your trash is empty.</p>
      ) : (
        <div className="trash-list">
          {visible.map((item) => (
            <div className="trash-item" key={`${item.kind}:${item.id}`}>
              <span aria-hidden="true">
                {item.kind === "workspace" ? (
                  <Folder size={18} />
                ) : (
                  <File size={18} />
                )}
              </span>
              <div>
                <strong>{item.name}</strong>
                <small>
                  {item.kind === "page"
                    ? `Page in ${item.workspaceName}`
                    : "Workspace"}
                </small>
                <small>
                  Deleted permanently on{" "}
                  {new Date(item.purgeAt * 1000).toLocaleDateString()}
                </small>
              </div>
              <button
                className="secondary"
                disabled={!!busy}
                aria-label={`Restore ${item.name}`}
                onClick={() => void restore(item)}
              >
                {busy === item.id ? <Spinner /> : <RotateCcw size={14} />}
                Restore
              </button>
            </div>
          ))}
        </div>
      )}
    </Modal>
  );
}
