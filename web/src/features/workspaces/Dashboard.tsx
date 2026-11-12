import {
  ArrowUpRight,
  Copy,
  Layers3,
  LockKeyhole,
  MoreHorizontal,
  Pencil,
  Plus,
  Trash2,
} from "lucide-react";
import { useEffect, useState } from "react";
import type { ServerEvent } from "../../api";
import { api } from "../../api";
import type { Workspace } from "../../model";
import { Empty, ErrorMessage, Spinner } from "../../ui";
import NameDialog from "./NameDialog";
export default function Dashboard({
  open,
  serverEvent,
}: {
  open: (w: Workspace) => void;
  serverEvent: ServerEvent;
}) {
  const [items, setItems] = useState<Workspace[]>([]),
    [loading, setLoading] = useState(true),
    [error, setError] = useState("");
  const [dialog, setDialog] = useState<{
    title: string;
    initial?: string;
    action: (name: string) => Promise<void>;
  }>();
  const [menu, setMenu] = useState<string>();
  async function load() {
    setItems(await api("/workspaces"));
  }
  useEffect(() => {
    void load()
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);
  useEffect(() => {
    if (serverEvent.type === "workspace" || serverEvent.type === "connected")
      void load().catch(() => {});
  }, [serverEvent]);
  function create() {
    setDialog({
      title: "A new workspace",
      action: async (name) => {
        const result = await api<{ id: string }>("/workspaces", "POST", {
          name,
        });
        const all = await api<Workspace[]>("/workspaces");
        setItems(all);
        open(all.find((w) => w.id === result.id)!);
      },
    });
  }
  async function mutate(fn: () => Promise<unknown>) {
    try {
      await fn();
      await load();
      setMenu(undefined);
    } catch (e) {
      setError((e as Error).message);
    }
  }
  return (
    <main className="workspace-list">
      <div className="workspace-heading">
        <div>
          <span className="eyebrow">YOUR PRIVATE SPACE</span>
          <h1>Workspaces</h1>
          <p className="muted">
            A home for every idea. A little room to let it grow.
          </p>
        </div>
        <button className="primary" onClick={create}>
          <Plus size={17} />
          New workspace
        </button>
      </div>
      <ErrorMessage message={error} />
      {loading ? (
        <Spinner />
      ) : items.length === 0 ? (
        <Empty
          icon={<Layers3 size={27} strokeWidth={1.4} />}
          title="Make room for an idea."
        >
          Create your first workspace to collect inspiration,
          <br />
          connect thoughts, and see what takes shape.
          <br />
          <button
            className="primary"
            style={{ marginTop: 25 }}
            onClick={create}
          >
            <Plus size={16} />
            Create a workspace
          </button>
        </Empty>
      ) : (
        <>
          <div className="list-label">
            ALL WORKSPACES <span>{items.length}</span>
          </div>
          <div className="workspace-grid">
            {items.map((w, i) => (
              <article className="workspace-card" key={w.id}>
                <button
                  className={`workspace-preview preview-${i % 4}`}
                  onClick={() => open(w)}
                  aria-label={`Open ${w.name}`}
                >
                  <div className="preview-stack">
                    <span />
                    <span />
                    <span />
                  </div>
                  <span className="preview-open">
                    <ArrowUpRight size={18} />
                  </span>
                </button>
                <div className="workspace-card-footer">
                  <button className="workspace-title" onClick={() => open(w)}>
                    <strong>{w.name}</strong>
                    <span>
                      {w.pageCount} {w.pageCount === 1 ? "page" : "pages"}
                      {w.shared ? " · Shared" : ""}
                      <span className="info-dot">·</span>
                      {new Date(w.updatedAt * 1000).toLocaleDateString(
                        undefined,
                        { month: "short", day: "numeric" },
                      )}
                    </span>
                  </button>
                  <div className="menu-anchor">
                    <button
                      className="icon-button"
                      aria-label={`Options for ${w.name}`}
                      hidden={!!w.role && w.role !== "owner"}
                      onClick={() => setMenu(menu === w.id ? undefined : w.id)}
                    >
                      <MoreHorizontal size={19} />
                    </button>
                    {menu === w.id && (
                      <>
                        <button
                          className="menu-dismiss"
                          aria-label="Close workspace menu"
                          onClick={() => setMenu(undefined)}
                        />
                        <div className="dropdown">
                          <button
                            onClick={() => {
                              setMenu(undefined);
                              setDialog({
                                title: "Rename workspace",
                                initial: w.name,
                                action: async (name) => {
                                  await api(`/workspaces/${w.id}`, "PATCH", {
                                    name,
                                    revision: w.revision,
                                  });
                                  await load();
                                },
                              });
                            }}
                          >
                            <Pencil size={14} />
                            Rename
                          </button>
                          <button
                            onClick={() =>
                              mutate(() =>
                                api(
                                  `/workspaces/${w.id}/duplicate`,
                                  "POST",
                                  {},
                                ),
                              )
                            }
                          >
                            <Copy size={14} />
                            Duplicate
                          </button>
                          <button
                            className="danger"
                            onClick={() => {
                              if (
                                confirm(`Delete “${w.name}” and all its pages?`)
                              )
                                void mutate(() =>
                                  api(`/workspaces/${w.id}`, "DELETE", {
                                    revision: w.revision,
                                    confirm: true,
                                  }),
                                );
                            }}
                          >
                            <Trash2 size={14} />
                            Delete
                          </button>
                        </div>
                      </>
                    )}
                  </div>
                </div>
              </article>
            ))}
            <button className="workspace-card new-card" onClick={create}>
              <span>
                <Plus size={24} />
              </span>
              Start something new
            </button>
          </div>
        </>
      )}
      <div className="workspace-bottom">
        <LockKeyhole size={12} />
        Only you can see your workspaces.
      </div>
      {dialog && <NameDialog {...dialog} close={() => setDialog(undefined)} />}
    </main>
  );
}
