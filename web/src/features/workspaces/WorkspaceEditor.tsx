import TrashDialog from "./TrashDialog";
import ExportDialog from "../transfer/ExportDialog";
import ImportWorkspaceDialog from "./ImportWorkspaceDialog";
import type { CanvasFullscreen } from "../../useCanvasFullscreen";
import ControlRequestDialog from "../editor/ControlRequestDialog";
import {
  ArrowLeft,
  Download,
  Upload,
  Check,
  ChevronDown,
  ChevronUp,
  CloudOff,
  Copy,
  FolderOpen,
  GripVertical,
  LockKeyhole,
  MoreHorizontal,
  Pencil,
  Plus,
  Trash2,
  X,
} from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import type { ServerEvent } from "../../api";
import { api } from "../../api";
import type { Page, Workspace } from "../../model";
import { PageSession } from "../../session";
import { Empty, ErrorMessage, Modal, Spinner } from "../../ui";
import SharingDialog from "./SharingDialog";
import Canvas from "../editor/Canvas";
import NameDialog from "./NameDialog";
export default function WorkspaceEditor({
  workspace: initial,
  headerCollapsed,
  onToggleHeader,
  fullscreen,
  activePageId,
  onPageChange: setPageId,
  back,
  dirtyChanged,
  onAuthenticationRequired,
  serverEvent,
}: {
  workspace: Workspace;
  headerCollapsed: boolean;
  onToggleHeader: () => void;
  fullscreen: CanvasFullscreen;
  activePageId?: string;
  onPageChange: (pageId?: string, replace?: boolean) => void;
  back: () => void;
  dirtyChanged: (dirty: boolean) => void;
  onAuthenticationRequired: () => void;
  serverEvent: ServerEvent;
}) {
  const [workspace, setWorkspace] = useState(initial),
    [pages, setPages] = useState<Page[]>([]),
    [loading, setLoading] = useState(true),
    [error, setError] = useState(""),
    [dialog, setDialog] = useState<{
      title: string;
      initial?: string;
      action: (name: string) => Promise<void>;
    }>(),
    [menu, setMenu] = useState<string>();
  const [exporting, setExporting] = useState<{ pageId?: string }>();
  const [trashOpen, setTrashOpen] = useState(false);
  const [importing, setImporting] = useState(false);
  const [sharing, setSharing] = useState(false);
  const canManage = !workspace.role || workspace.role === "owner";
  const canWrite = canManage || workspace.role === "full";
  const [refresh, setRefresh] = useState(0),
    [, render] = useState(0),
    [recovering, setRecovering] = useState(false);
  const [recoveryTarget, setRecoveryTarget] = useState("");
  const [recoveryWorkspaces, setRecoveryWorkspaces] = useState<Workspace[]>();
  const sessions = useRef(new Map<string, PageSession>()),
    unsubscribers = useRef(new Map<string, () => void>());
  const dragTab = useRef<string | undefined>(undefined);
  const workspaceId = workspace.id;
  const pageId = activePageId;
  const session = pageId ? sessions.current.get(pageId) : undefined;
  useEffect(() => {
    if (!pageId && pages[0]) setPageId(pages[0].id, true);
  }, [pageId, pages, setPageId]);
  const load = useCallback(async () => {
    const list = await api<Page[]>(`/workspaces/${workspaceId}/pages`);
    setPages(list);
    const all = await api<Workspace[]>("/workspaces");
    const current = all.find((w) => w.id === workspaceId);
    if (current) setWorkspace(current);
  }, [workspaceId]);
  useEffect(() => {
    const openSessions = sessions.current;
    const subscriptions = unsubscribers.current;
    void load()
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
    return () => {
      for (const unsub of subscriptions.values()) unsub();
      for (const s of openSessions.values()) s.dispose();
      openSessions.clear();
      dirtyChanged(false);
    };
  }, [load, dirtyChanged]);
  useEffect(() => {
    if (
      !pageId ||
      !pages.some((page) => page.id === pageId) ||
      sessions.current.has(pageId)
    )
      return;
    const s = new PageSession(pageId, api, onAuthenticationRequired);
    sessions.current.set(pageId, s);
    unsubscribers.current.set(
      pageId,
      s.subscribe(() => {
        render((n) => n + 1);
        dirtyChanged([...sessions.current.values()].some((s) => s.unsaved));
      }),
    );
    void s.open();
    render((n) => n + 1);
  }, [pageId, pages, dirtyChanged, onAuthenticationRequired]);
  useEffect(() => {
    for (const session of sessions.current.values()) void session.sync();
    if (serverEvent.type === "workspace" || serverEvent.type === "connected")
      void load().catch(() => {});
    if (serverEvent.type === "asset" || serverEvent.type === "comment")
      setRefresh((value) => value + 1);
  }, [serverEvent, load]);
  useEffect(() => {
    const online = () => {
      for (const session of sessions.current.values()) void session.sync();
      void load().catch(() => {});
      setRefresh((value) => value + 1);
    };
    const offline = () => {
      for (const session of sessions.current.values()) {
        session.status = "Disconnected";
        session.notify();
      }
    };
    window.addEventListener("online", online);
    window.addEventListener("offline", offline);
    return () => {
      window.removeEventListener("online", online);
      window.removeEventListener("offline", offline);
    };
  }, [load]);
  useEffect(() => {
    const listener = (e: BeforeUnloadEvent) => {
      if ([...sessions.current.values()].some((s) => s.unsaved)) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", listener);
    return () => window.removeEventListener("beforeunload", listener);
  }, []);
  function leave() {
    if (
      [...sessions.current.values()].some((s) => s.unsaved) &&
      !confirm(
        "There are unsaved changes. Leave this workspace and discard them?",
      )
    )
      return;
    back();
  }
  function newPage() {
    setDialog({
      title: "A new page",
      action: async (name) => {
        const result = await api<{ id: string }>(
          `/workspaces/${workspaceId}/pages`,
          "POST",
          { name },
        );
        await load();
        setPageId(result.id);
      },
    });
  }
  async function mutate(fn: () => Promise<unknown>) {
    setError("");
    try {
      await fn();
      await load();
      setMenu(undefined);
    } catch (e) {
      setError((e as Error).message);
    }
  }
  function pageAction(page: Page, action: string) {
    setMenu(undefined);
    if (action === "rename")
      setDialog({
        title: "Rename page",
        initial: page.name,
        action: async (name) => {
          await api(`/pages/${page.id}`, "PATCH", {
            name,
            revision: page.metadataRevision,
          });
          await load();
          const s = sessions.current.get(page.id);
          if (s) s.name = name;
        },
      });
    if (action === "duplicate")
      void mutate(async () => {
        const s = sessions.current.get(page.id);
        if (s?.dirty) {
          await s.flush();
          if (s.dirty) throw new Error("Save this page before duplicating it.");
        }
        const result = await api<{ id: string }>(
          `/pages/${page.id}/duplicate`,
          "POST",
          {},
        );
        setPageId(result.id);
      });
    if (
      action === "delete" &&
      confirm(
        `Move “${page.name}” to Trash? Saved content can be restored for 30 days. Unsaved changes will be discarded.`,
      )
    )
      void mutate(async () => {
        await api(`/pages/${page.id}`, "DELETE", {
          revision: page.metadataRevision,
          confirm: true,
        });
        const s = sessions.current.get(page.id);
        unsubscribers.current.get(page.id)?.();
        s?.dispose();
        sessions.current.delete(page.id);
        dirtyChanged([...sessions.current.values()].some((s) => s.unsaved));
        if (pageId === page.id)
          setPageId(pages.find((p) => p.id !== page.id)?.id);
      });
  }
  async function moveTab(target: string) {
    const source = dragTab.current;
    if (!source || source === target) return;
    const ids = pages.map((p) => p.id);
    ids.splice(ids.indexOf(source), 1);
    ids.splice(ids.indexOf(target), 0, source);
    await mutate(() =>
      api(`/workspaces/${workspaceId}/page-order`, "PUT", {
        ids,
        revision: workspace.revision,
      }),
    );
  }
  async function recover() {
    if (!session) return;
    setRecovering(true);
    try {
      const available = await api<Workspace[]>("/workspaces");
      const list = available.filter(
        (item) => !item.role || item.role === "owner" || item.role === "full",
      );
      setRecoveryWorkspaces(list);
      setRecoveryTarget(
        list.find((w) => w.id === workspaceId)?.id || list[0]?.id || "",
      );
    } catch (e) {
      setError((e as Error).message);
      setRecovering(false);
    }
  }
  const saveStatus = session?.status || "Loading";
  return (
    <div className="workspace-editor">
      <div className="workspace-navigation">
        <div className="editor-heading" id="workspace-heading">
          <div className="breadcrumb">
            <button
              className="icon-button"
              aria-label="Back to workspaces"
              onClick={leave}
            >
              <ArrowLeft size={18} />
            </button>
            <span className="breadcrumb-parent">Workspaces</span>
            <span className="breadcrumb-slash">/</span>
            <button
              className="workspace-name"
              title={canManage ? "Rename workspace" : "Workspace name"}
              disabled={!canManage}
              onClick={() =>
                setDialog({
                  title: "Rename workspace",
                  initial: workspace.name,
                  action: async (name) => {
                    await api(`/workspaces/${workspaceId}`, "PATCH", {
                      name,
                      revision: workspace.revision,
                    });
                    await load();
                  },
                })
              }
            >
              {workspace.name}
              <ChevronDown size={13} />
            </button>
            <button
              className="private-badge"
              aria-label="Workspace sharing"
              onClick={() => setSharing(true)}
            >
              <LockKeyhole size={10} />
              {workspace.shared ? "Shared" : "Private"}
            </button>
          </div>
          <div className="workspace-transfer-actions">
            {canWrite && (
              <button
                className="icon-button"
                aria-label="Workspace trash"
                title="Workspace trash"
                onClick={() => setTrashOpen(true)}
              >
                <Trash2 size={17} />
              </button>
            )}
            {canWrite && (
              <button
                className="icon-button"
                aria-label="Import pages"
                title="Import pages"
                onClick={() => setImporting(true)}
              >
                <Upload size={17} />
              </button>
            )}
            <button
              className="icon-button"
              aria-label="Export workspace"
              title="Export workspace"
              onClick={() => setExporting({})}
            >
              <Download size={17} />
            </button>
          </div>
          <div
            className={`save-status ${["Disconnected", "Save failed", "Control lost"].includes(saveStatus) ? "warning" : ""}`}
            aria-live="polite"
          >
            {saveStatus === "Saving" ? (
              <Spinner />
            ) : saveStatus === "Saved" ? (
              <Check size={13} />
            ) : saveStatus === "Disconnected" ? (
              <CloudOff size={14} />
            ) : (
              <span className="status-dot" />
            )}
            <span data-testid="save-status">{saveStatus}</span>
          </div>
        </div>
        <div
          className="page-tabs"
          id="workspace-pages"
          role="tablist"
          aria-label="Workspace pages"
        >
          {pages.map((p) => (
            <div
              className={`page-tab-wrap ${p.id === pageId ? "active" : ""}`}
              key={p.id}
              draggable={canWrite}
              onDragStart={() => {
                dragTab.current = p.id;
              }}
              onDragOver={(e) => e.preventDefault()}
              onDrop={(e) => {
                e.preventDefault();
                void moveTab(p.id);
              }}
            >
              <button
                className="page-tab"
                role="tab"
                aria-selected={p.id === pageId}
                onClick={() => setPageId(p.id)}
                onDoubleClick={() => {
                  if (canWrite) pageAction(p, "rename");
                }}
              >
                <GripVertical className="tab-grip" size={11} />
                {p.name}
                {sessions.current.get(p.id)?.unsaved && (
                  <span className="dirty-dot" />
                )}
              </button>
              <div className="menu-anchor">
                <button
                  className="tab-menu icon-button"
                  aria-label={`Page options for ${p.name}`}
                  onClick={() => setMenu(menu === p.id ? undefined : p.id)}
                >
                  <MoreHorizontal size={14} />
                </button>
                {menu === p.id && (
                  <>
                    <button
                      className="menu-dismiss"
                      aria-label="Close page menu"
                      onClick={() => setMenu(undefined)}
                    />
                    <div className="dropdown">
                      <button
                        onClick={() => {
                          setMenu(undefined);
                          setExporting({ pageId: p.id });
                        }}
                      >
                        <Download size={14} />
                        Export page
                      </button>
                      <button
                        disabled={!canWrite}
                        onClick={() => pageAction(p, "rename")}
                      >
                        <Pencil size={14} />
                        Rename
                      </button>
                      <button
                        disabled={!canWrite}
                        onClick={() => pageAction(p, "duplicate")}
                      >
                        <Copy size={14} />
                        Duplicate
                      </button>
                      <button
                        className="danger"
                        disabled={!canWrite}
                        onClick={() => pageAction(p, "delete")}
                      >
                        <Trash2 size={14} />
                        Delete
                      </button>
                    </div>
                  </>
                )}
              </div>
            </div>
          ))}
          <button
            className="icon-button add-page"
            aria-label="Add page"
            disabled={!canWrite}
            onClick={newPage}
          >
            <Plus size={17} />
          </button>
        </div>
        <button
          className="icon-button editor-header-toggle"
          aria-label={
            headerCollapsed
              ? "Show app header and pages"
              : "Hide app header and pages"
          }
          title={
            headerCollapsed
              ? "Show app header and pages"
              : "Hide app header and pages"
          }
          aria-expanded={!headerCollapsed}
          aria-controls="app-header workspace-heading workspace-pages"
          onClick={onToggleHeader}
        >
          {headerCollapsed ? (
            <ChevronDown size={16} />
          ) : (
            <ChevronUp size={16} />
          )}
        </button>
      </div>
      {error && (
        <div className="editor-banner error" role="alert">
          {error}
          <button
            className="icon-button"
            aria-label="Dismiss message"
            onClick={() => setError("")}
          >
            <X size={15} />
          </button>
        </div>
      )}
      {session?.error && (
        <div className="editor-banner warning" role="alert">
          {session.error}
          <button
            className="text-button"
            onClick={() => {
              void session.sync();
              void session.flush();
            }}
          >
            Retry
          </button>
        </div>
      )}
      {session?.control && <ControlRequestDialog session={session} />}
      {[...sessions.current.entries()]
        .filter(([id, s]) => id !== pageId && s.recovery)
        .map(([id, s]) => (
          <div className="editor-banner recovery-banner" key={id}>
            <span>
              Unsaved local work from “{s.name || "another page"}” needs
              recovery.
            </span>
            <button className="secondary compact" onClick={() => setPageId(id)}>
              Review local work
            </button>
          </div>
        ))}
      {session?.recovery && (
        <div className="editor-banner recovery-banner">
          <span>
            Your unsaved work is safe in this tab. Save it as a separate page to
            keep it.
          </span>
          <button className="secondary compact" onClick={recover}>
            Save as a separate page
          </button>
          <button
            className="text-button"
            onClick={() => {
              if (confirm("Discard the unsaved local recovery copy?"))
                session.discardRecovery();
            }}
          >
            Discard
          </button>
        </div>
      )}
      {loading ? (
        <div className="editor-loading">
          <Spinner />
        </div>
      ) : session?.loaded ? (
        <Canvas
          key={pageId}
          session={session}
          fullscreen={fullscreen}
          refresh={refresh}
          onImportedPages={(id) => {
            void load()
              .then(() => setPageId(id))
              .catch((err) => setError(err.message));
          }}
        />
      ) : session?.status === "Page unavailable" ||
        (pageId && !pages.some((page) => page.id === pageId)) ? (
        <Empty icon={<LockKeyhole size={25} />} title="Page unavailable">
          <button className="secondary" onClick={leave}>
            Back to workspaces
          </button>
        </Empty>
      ) : pages.length === 0 ? (
        <Empty icon={<FolderOpen size={25} />} title="A fresh page awaits.">
          <button className="primary" onClick={newPage}>
            <Plus size={16} />
            Add a page
          </button>
        </Empty>
      ) : (
        <div className="editor-loading">
          <Spinner />
          {session?.status === "Load failed" && (
            <button className="secondary" onClick={() => void session.open()}>
              Retry loading
            </button>
          )}
        </div>
      )}
      {trashOpen && (
        <TrashDialog
          workspaceId={workspaceId}
          close={() => setTrashOpen(false)}
          changed={load}
        />
      )}
      {exporting && (
        <ExportDialog
          workspace={workspace}
          pages={pages}
          pageId={exporting.pageId}
          close={() => setExporting(undefined)}
          beforeExport={async (ids) => {
            for (const [id, current] of sessions.current)
              if (!ids.length || ids.includes(id))
                await current.savePendingChanges();
          }}
        />
      )}
      {importing && (
        <ImportWorkspaceDialog
          workspaceId={workspaceId}
          close={() => {
            setImporting(false);
            void load().catch((e) => setError(e.message));
          }}
          open={async (_id, importedPage) => {
            await load();
            setImporting(false);
            if (importedPage) setPageId(importedPage);
          }}
        />
      )}
      {sharing && (
        <SharingDialog
          workspace={workspace}
          close={() => setSharing(false)}
          changed={load}
          beforeTransfer={async () => {
            for (const current of sessions.current.values())
              await current.savePendingChanges();
          }}
        />
      )}
      {dialog && <NameDialog {...dialog} close={() => setDialog(undefined)} />}
      {recovering && recoveryWorkspaces && (
        <Modal
          title="Keep your local work"
          close={() => {
            setRecovering(false);
            setRecoveryWorkspaces(undefined);
          }}
        >
          <p className="muted">
            Create an independent page from your unsaved canvas.
          </p>
          <label>
            Workspace
            <select
              value={recoveryTarget}
              onChange={(e) => setRecoveryTarget(e.target.value)}
            >
              {recoveryWorkspaces.map((w) => (
                <option key={w.id} value={w.id}>
                  {w.name}
                </option>
              ))}
            </select>
          </label>
          <ErrorMessage message={error} />
          <div className="form-actions">
            <button
              className="secondary"
              onClick={async () => {
                const name = prompt(
                  "Name for a new workspace",
                  "Recovered work",
                );
                if (!name) return;
                try {
                  const result = await api<{ id: string }>(
                    "/workspaces",
                    "POST",
                    { name },
                  );
                  setRecoveryTarget(result.id);
                  setRecoveryWorkspaces(await api("/workspaces"));
                } catch (e) {
                  setError((e as Error).message);
                }
              }}
            >
              New workspace
            </button>
            <button
              className="primary"
              disabled={!recoveryTarget}
              onClick={async () => {
                if (!session) return;
                try {
                  const id = await session.recover(recoveryTarget);
                  setRecovering(false);
                  setRecoveryWorkspaces(undefined);
                  if (recoveryTarget === workspaceId) {
                    await load();
                    setPageId(id);
                  } else
                    setError("Recovered page saved in the selected workspace.");
                } catch (e) {
                  setError((e as Error).message);
                }
              }}
            >
              Save recovered page
            </button>
          </div>
        </Modal>
      )}
    </div>
  );
}
