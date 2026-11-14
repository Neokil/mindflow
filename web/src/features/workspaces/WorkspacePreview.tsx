import { Layers3 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { api } from "../../api";
import type { Workspace } from "../../model";
import type { PageData } from "../../session";

// Only one offscreen canvas renders at a time, even when many cards become visible.
let previewQueue: Promise<unknown> = Promise.resolve();
function enqueuePreview(task: () => Promise<void>) {
  const pending = previewQueue.then(task);
  previewQueue = pending.catch(() => {});
  return pending;
}

export default function WorkspacePreview({
  workspace,
  background,
}: {
  workspace: Workspace;
  background: string;
}) {
  const host = useRef<HTMLSpanElement>(null);
  const [visible, setVisible] = useState(false);
  const [preview, setPreview] = useState<{
    url: string;
    background: string;
    empty: boolean;
  }>();
  const [failed, setFailed] = useState(false);
  const { firstPageId, previewRevision } = workspace;
  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          setVisible(true);
          observer.disconnect();
        }
      },
      { rootMargin: "200px" },
    );
    if (host.current) observer.observe(host.current);
    return () => observer.disconnect();
  }, []);
  useEffect(() => {
    let canceled = false;
    let url: string | undefined;
    if (!visible || !firstPageId || !background) return;
    void enqueuePreview(async () => {
      if (canceled) return;
      setPreview(undefined);
      setFailed(false);
      const page = await api<PageData>(`/pages/${firstPageId}/document`);
      if (canceled) return;
      const { renderPage } = await import("../transfer/renderPage");
      if (canceled) return;
      const blob = await renderPage(page.document, {
        resolution: 512,
        background: true,
        grid: true,
        backgroundColor: background,
      });
      if (canceled) return;
      url = URL.createObjectURL(blob);
      setPreview({
        url,
        background: page.document.settings.backgroundColor || background,
        empty: !page.document.elements.length,
      });
    }).catch(() => {
      if (!canceled) setFailed(true);
    });
    return () => {
      canceled = true;
      if (url) URL.revokeObjectURL(url);
    };
  }, [visible, firstPageId, previewRevision, background]);
  return (
    <span
      ref={host}
      className={`workspace-preview-content ${workspace.pageCount > 1 ? "has-pages" : ""}`}
    >
      {workspace.pageCount > 1 && (
        <span className="workspace-page-stack" aria-hidden="true">
          <span />
          <span />
        </span>
      )}
      <span
        className="workspace-preview-sheet"
        style={{ backgroundColor: preview?.background || background }}
      >
        {preview ? (
          <img
            className="workspace-thumbnail"
            src={preview.url}
            alt={`Preview of the first page of ${workspace.name}`}
          />
        ) : (
          <span className="workspace-preview-status">
            {!firstPageId
              ? "No pages"
              : failed
                ? "Preview unavailable"
                : "Loading preview…"}
          </span>
        )}
      </span>
      {preview?.empty && (
        <span className="workspace-preview-empty">Blank first page</span>
      )}
      {workspace.pageCount > 1 && (
        <span className="workspace-page-count">
          <Layers3 size={13} />
          {workspace.pageCount} pages
        </span>
      )}
    </span>
  );
}
