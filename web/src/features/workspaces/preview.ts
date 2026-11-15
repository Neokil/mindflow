import { api, APIError } from "../../api";
import type { PageData } from "../../session";

type PreviewState = {
  pageId: string;
  revision: number;
  needed: boolean;
  canWrite: boolean;
};

// Serialize offscreen renders and coalesce saves without losing the latest one.
let queue: Promise<void> = Promise.resolve();
const pending = new Map<string, boolean>();
export function refreshWorkspacePreview(workspaceId: string) {
  if (pending.has(workspaceId)) {
    pending.set(workspaceId, true);
    return;
  }
  pending.set(workspaceId, false);
  queue = queue.then(async () => {
    try {
      do {
        pending.set(workspaceId, false);
        try {
          await updatePreview(workspaceId);
        } catch (error) {
          if (error instanceof APIError && error.status === 409) {
            await updatePreview(workspaceId);
          } else throw error;
        }
      } while (pending.get(workspaceId));
    } catch {
      // A derived thumbnail must never turn a successful document save into a failure.
      // Opening or saving the workspace again retries a missing/stale preview.
    } finally {
      pending.delete(workspaceId);
    }
  });
}

async function updatePreview(workspaceId: string) {
  const state = await api<PreviewState>(`/workspaces/${workspaceId}/preview`);
  if (!state.needed || !state.canWrite || !state.pageId) return;
  const page = await api<PageData>(`/pages/${state.pageId}/document`);
  const { renderPage } = await import("../transfer/renderPage");
  const color = page.document.settings.backgroundColor;
  const blob = await renderPage(page.document, {
    resolution: 512,
    background: !!color,
    grid: true,
    backgroundColor: color || "#ffffff",
  });
  const form = new FormData();
  form.set("pageId", state.pageId);
  form.set("revision", String(page.revision));
  form.set("file", blob, "preview.png");
  await api(`/workspaces/${workspaceId}/preview`, "POST", form);
}
