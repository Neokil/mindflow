import { api, APIError } from "./api";
import type { CanvasDocument, WorkspacePermission } from "./model";
import { clone, emptyDocument, uid } from "./model";
export type ControlRequest = {
  requesterId: string;
  avatarVersion?: string;
  id: string;
  requesterEmail: string;
  requesterDisplayName: string;
  deadline: number;
  status: string;
  incoming: boolean;
  serverNow: number;
};
export type Lease = {
  request?: ControlRequest;
  permission?: WorkspacePermission;
  editable: boolean;
  epoch: number;
  expiresAt: number;
  revision: number;
};
export type PageData = {
  document: CanvasDocument;
  revision: number;
  name: string;
  workspaceId: string;
};
export type LocalImage = {
  file: File;
  url: string;
  width: number;
  height: number;
};
export const localImages = new Map<string, LocalImage>();
export type Transport = <T = unknown>(
  path: string,
  method?: string,
  body?: unknown,
) => Promise<T>;
type SaveRequest = {
  baseRevision: number;
  epoch: number;
  clientId: string;
  mutationId: string;
  document: CanvasDocument;
};
export class PageSession {
  readonly clientId = uid();
  control?: ControlRequest;
  controlDeadline = 0;
  private dismissedControl = "";
  camera?: { x: number; y: number; zoom: number };
  doc = emptyDocument();
  revision = 0;
  epoch = 0;
  editable = false;
  permission: WorkspacePermission = "owner";
  loaded = false;
  status = "Loading";
  error = "";
  name = "";
  workspaceId = "";
  generation = 0;
  acknowledged = 0;
  undoStack: CanvasDocument[] = [];
  redoStack: CanvasDocument[] = [];
  recovery?: CanvasDocument;
  private listeners = new Set<() => void>();
  private timer?: ReturnType<typeof setTimeout>;
  private poll?: ReturnType<typeof setInterval>;
  private renewal = 0;
  private busy = false;
  private syncing = false;
  private disposed = false;
  private pending?: { request: SaveRequest; generation: number };
  private dirtySince = 0;
  constructor(
    public id: string,
    private request: Transport = api,
    private onAuthenticationRequired: () => void = () => {},
    private onSaved: () => void = () => {},
  ) {}
  get canWrite() {
    return this.permission === "owner" || this.permission === "full";
  }
  get canComment() {
    return this.canWrite || this.permission === "comment";
  }
  get dirty() {
    return this.generation !== this.acknowledged;
  }
  get unsaved() {
    return this.dirty || !!this.recovery;
  }
  subscribe = (fn: () => void) => {
    this.listeners.add(fn);
    return () => {
      this.listeners.delete(fn);
    };
  };
  notify = () => {
    for (const fn of this.listeners) fn();
  };
  private authFailure(e: unknown) {
    if (e instanceof APIError && e.status === 401)
      this.onAuthenticationRequired();
  }
  async open() {
    try {
      const lease = await this.request<Lease>(
        `/pages/${this.id}/lease`,
        "POST",
        { clientId: this.clientId, action: "acquire" },
      );
      const data = await this.request<PageData>(`/pages/${this.id}/document`);
      if (this.disposed) return;
      this.doc = data.document;
      this.revision = data.revision;
      this.name = data.name;
      this.workspaceId = data.workspaceId;
      this.permission = lease.permission || "owner";
      this.epoch = lease.epoch;
      this.editable = lease.editable;
      this.receiveControl(lease);
      this.renewal = Date.now();
      this.loaded = true;
      this.status = "Saved";
      this.poll = setInterval(() => void this.sync(), 1000);
    } catch (e) {
      this.status = "Load failed";
      this.error = (e as Error).message;
      this.authFailure(e);
    }
    this.notify();
  }
  edit(next: CanvasDocument, record = true) {
    if (!this.editable || !this.loaded) return;
    if (next.elements.length > 10000) {
      this.error = "A page can contain at most 10,000 elements.";
      this.notify();
      return;
    }
    if (JSON.stringify(next) === JSON.stringify(this.doc)) return;
    if (record) {
      this.undoStack.push(this.doc);
      if (this.undoStack.length > 100) this.undoStack.shift();
      this.redoStack = [];
    }
    this.doc = next;
    this.generation++;
    if (!this.dirtySince) this.dirtySince = Date.now();
    this.status = navigator.onLine === false ? "Disconnected" : "Unsaved";
    this.error = "";
    this.notify();
    this.schedule();
  }
  undo() {
    if (!this.editable || !this.undoStack.length) return;
    const prev = this.undoStack.pop()!;
    this.redoStack.push(this.doc);
    this.edit(prev, false);
  }
  redo() {
    if (!this.editable || !this.redoStack.length) return;
    const next = this.redoStack.pop()!;
    this.undoStack.push(this.doc);
    this.edit(next, false);
  }
  private schedule(delay = 750) {
    clearTimeout(this.timer);
    this.timer = setTimeout(
      () => void this.flush(),
      Math.min(
        delay,
        Math.max(0, 5000 - (Date.now() - (this.dirtySince || Date.now()))),
      ),
    );
  }
  private replaceAsset(oldId: string, newId: string) {
    const replace = (doc: CanvasDocument) => ({
      ...doc,
      elements: doc.elements.map((e) =>
        e.assetId === oldId ? { ...e, assetId: newId } : e,
      ),
    });
    this.doc = replace(this.doc);
    this.undoStack = this.undoStack.map(replace);
    this.redoStack = this.redoStack.map(replace);
    if (this.recovery) this.recovery = replace(this.recovery);
    const local = localImages.get(oldId);
    if (local) localImages.set(newId, local);
  }
  async prepare(document: CanvasDocument, workspaceId = this.workspaceId) {
    const result = clone(document);
    const ids = new Set(
      result.elements
        .filter((e) => e.type === "image" && e.assetId?.startsWith("local:"))
        .map((e) => e.assetId!),
    );
    for (const id of ids) {
      const image = localImages.get(id);
      if (!image)
        throw new Error(
          "An image is no longer available. Reattach it before saving.",
        );
      const form = new FormData();
      if (workspaceId) form.append("workspaceId", workspaceId);
      form.append("file", image.file);
      const uploaded = await this.request<{ id: string }>(
        "/assets",
        "POST",
        form,
      );
      this.replaceAsset(id, uploaded.id);
      result.elements = result.elements.map((e) =>
        e.assetId === id ? { ...e, assetId: uploaded.id } : e,
      );
    }
    return result;
  }
  async savePendingChanges() {
    const deadline = Date.now() + 30000;
    while (this.busy && Date.now() < deadline)
      await new Promise((resolve) => setTimeout(resolve, 50));
    if (!this.busy && this.dirty) await this.flush();
    if (this.unsaved || this.busy)
      throw new Error(
        "Save or recover your unsaved changes before continuing. Check your connection and editor control.",
      );
  }
  async flush() {
    if (this.busy || !this.dirty || !this.editable || this.disposed) return;
    this.busy = true;
    clearTimeout(this.timer);
    this.status = "Saving";
    this.notify();
    try {
      if (!this.pending) {
        const generation = this.generation;
        const snapshot = clone(this.doc);
        const document = await this.prepare(snapshot);
        if (!this.editable || this.disposed) return;
        this.pending = {
          generation,
          request: {
            baseRevision: this.revision,
            epoch: this.epoch,
            clientId: this.clientId,
            mutationId: uid(),
            document,
          },
        };
      }
      const pending = this.pending;
      const result = await this.request<{ revision: number }>(
        `/pages/${this.id}/document`,
        "PUT",
        pending.request,
      );
      this.revision = result.revision;
      this.onSaved();
      this.acknowledged = pending.generation;
      this.pending = undefined;
      this.dirtySince = this.dirty ? Date.now() : 0;
      this.status = this.dirty ? "Unsaved" : "Saved";
      this.error = "";
    } catch (e) {
      this.authFailure(e);
      if (e instanceof APIError && e.status === 409) {
        const restored =
          e.code === "lease_lost" && (await this.restoreExpiredControl());
        if (restored === false) await this.loseControl();
        else if (restored === undefined) {
          this.status = "Disconnected";
          this.error =
            "Could not verify editor control. Your changes remain in this tab.";
        }
      } else {
        this.status = e instanceof APIError ? "Save failed" : "Disconnected";
        this.error = (e as Error).message;
        // A corrected edit must replace a rejected snapshot, while uncertain
        // network outcomes retain their exact mutation identity for retry.
        if (
          e instanceof APIError &&
          e.status >= 400 &&
          e.status < 500 &&
          e.status !== 401
        ) {
          this.pending = undefined;
          if (e.code === "asset_missing") {
            for (const id of new Set(
              this.doc.elements
                .filter((el) => el.type === "image")
                .map((el) => el.assetId!),
            )) {
              const local = localImages.get(id);
              if (local && !id.startsWith("local:")) {
                const fresh = `local:${uid()}`;
                localImages.set(fresh, local);
                this.replaceAsset(id, fresh);
              }
            }
          }
        }
      }
    } finally {
      this.busy = false;
      this.notify();
      if (this.dirty && this.editable && !this.disposed) {
        clearTimeout(this.timer);
        this.timer = setTimeout(
          () => void this.flush(),
          this.status === "Unsaved" ? 750 : 5000,
        );
      }
    }
  }
  private async restoreExpiredControl() {
    const pending = this.pending;
    if (!pending) return false;
    try {
      let lease = await this.request<Lease>(
        `/pages/${this.id}/lease?clientId=${this.clientId}`,
      );
      if (lease.revision !== pending.request.baseRevision) return false;
      if (!lease.editable) {
        if (
          lease.epoch !== pending.request.epoch ||
          lease.expiresAt > Math.floor(Date.now() / 1000)
        )
          return false;
        lease = await this.request<Lease>(`/pages/${this.id}/lease`, "POST", {
          clientId: this.clientId,
          action: "acquire",
        });
      }
      if (!lease.editable || lease.revision !== pending.request.baseRevision)
        return false;
      this.epoch = lease.epoch;
      this.renewal = Date.now();
      this.pending = undefined;
      this.status = "Unsaved";
      this.error = "";
      return true;
    } catch (error) {
      this.authFailure(error);
      return undefined;
    }
  }
  private async loseControl() {
    if (this.dirty && !this.recovery) this.recovery = clone(this.doc);
    this.editable = false;
    this.pending = undefined;
    this.generation = 0;
    this.acknowledged = 0;
    this.undoStack = [];
    this.redoStack = [];
    this.status = "Control lost";
    try {
      const data = await this.request<PageData>(`/pages/${this.id}/document`);
      this.doc = data.document;
      this.revision = data.revision;
    } catch (e) {
      this.error = (e as Error).message;
      this.authFailure(e);
    }
    this.notify();
  }
  async sync() {
    if (!this.loaded || this.disposed || this.syncing || this.busy) return;
    this.syncing = true;
    try {
      // Resolve a lost acknowledgment before judging a changed revision/lease.
      if (this.pending && this.editable) {
        await this.flush();
        if (!this.editable) return;
      }
      let lease = await this.request<Lease>(
        `/pages/${this.id}/lease?clientId=${this.clientId}`,
      );
      this.receiveControl(lease);
      this.permission = lease.permission || "owner";
      if (!this.editable && lease.editable) {
        const data = await this.request<PageData>(`/pages/${this.id}/document`);
        this.doc = data.document;
        this.revision = data.revision;
        this.epoch = lease.epoch;
        this.editable = true;
        this.renewal = Date.now();
        this.status = "Saved";
        this.error = "";
      }
      if (this.editable) {
        if (!lease.editable) {
          if (
            lease.epoch === this.epoch &&
            lease.expiresAt <= Math.floor(Date.now() / 1000) &&
            lease.revision === this.revision
          )
            lease = await this.request<Lease>(
              `/pages/${this.id}/lease`,
              "POST",
              { clientId: this.clientId, action: "acquire" },
            );
          if (!lease.editable) {
            await this.loseControl();
            return;
          }
          this.epoch = lease.epoch;
        }
        if (lease.revision !== this.revision && !this.pending) {
          if (this.dirty) {
            await this.loseControl();
            return;
          }
          const data = await this.request<PageData>(
            `/pages/${this.id}/document`,
          );
          this.doc = data.document;
          this.revision = data.revision;
          this.undoStack = [];
          this.redoStack = [];
        }
        if (Date.now() - this.renewal > 15000) {
          lease = await this.request<Lease>(`/pages/${this.id}/lease`, "POST", {
            clientId: this.clientId,
            epoch: this.epoch,
            action: "renew",
          });
          this.epoch = lease.epoch;
          this.renewal = Date.now();
        }
        if (this.dirty) void this.flush();
        else {
          this.status = "Saved";
          this.error = "";
        }
      } else if (lease.revision !== this.revision) {
        const data = await this.request<PageData>(`/pages/${this.id}/document`);
        this.doc = data.document;
        this.revision = data.revision;
        this.name = data.name;
      }
    } catch (e) {
      this.authFailure(e);
      if (e instanceof APIError && e.status === 409) await this.loseControl();
      else if (e instanceof APIError && e.status === 404) {
        if (this.dirty) await this.loseControl();
        this.editable = false;
        this.loaded = false;
        this.doc = emptyDocument();
        this.error =
          "This page was deleted or workspace access was removed. Local unsaved work can be recovered into another workspace.";
        this.status = "Page unavailable";
      } else {
        this.status = "Disconnected";
        this.error = (e as Error).message;
      }
    } finally {
      this.syncing = false;
      this.notify();
    }
  }
  private receiveControl(lease: Lease) {
    this.control =
      lease.request?.id === this.dismissedControl ? undefined : lease.request;
    if (this.control)
      this.controlDeadline =
        Date.now() + this.control.deadline - this.control.serverNow;
  }
  async decideControl(action: "accept" | "deny" | "cancel") {
    if (!this.control) return;
    try {
      const lease = await this.request<Lease>(
        `/pages/${this.id}/lease`,
        "POST",
        {
          clientId: this.clientId,
          epoch: this.epoch,
          action,
          requestId: this.control.id,
        },
      );
      this.receiveControl(lease);
      await this.sync();
    } catch (error) {
      this.error = (error as Error).message;
      this.authFailure(error);
    }
    this.notify();
  }
  dismissControl() {
    this.dismissedControl = this.control?.id || "";
    this.control = undefined;
    this.notify();
  }
  async takeover(force = false) {
    if (this.busy || this.recovery || !this.canWrite) return;
    try {
      if (this.dirty) this.recovery = clone(this.doc);
      const lease = await this.request<Lease>(
        `/pages/${this.id}/lease`,
        "POST",
        { clientId: this.clientId, action: force ? "force" : "takeover" },
      );
      const data = await this.request<PageData>(`/pages/${this.id}/document`);
      this.doc = data.document;
      this.revision = data.revision;
      this.epoch = lease.epoch;
      this.permission = lease.permission || "owner";
      this.editable = lease.editable;
      this.receiveControl(lease);
      this.renewal = Date.now();
      this.generation = 0;
      this.acknowledged = 0;
      this.pending = undefined;
      this.undoStack = [];
      this.redoStack = [];
      this.status = "Saved";
      this.error = "";
    } catch (e) {
      this.authFailure(e);
      this.error = (e as Error).message;
    }
    this.notify();
  }
  async recover(workspaceId: string) {
    if (!this.recovery) throw new Error("No local recovery copy is available.");
    const document = await this.prepare(this.recovery, workspaceId);
    const result = await this.request<{ id: string }>(
      "/pages/recovery-copy",
      "POST",
      { workspaceId, name: `${this.name || "Untitled"} — recovered`, document },
    );
    this.recovery = undefined;
    this.notify();
    return result.id;
  }
  discardRecovery() {
    this.recovery = undefined;
    this.notify();
  }
  dispose() {
    this.disposed = true;
    clearTimeout(this.timer);
    clearInterval(this.poll);
    if (!this.unsaved)
      void this.request(`/pages/${this.id}/lease`, "POST", {
        clientId: this.clientId,
        epoch: this.epoch,
        action: "release",
      }).catch(() => {});
  }
}
