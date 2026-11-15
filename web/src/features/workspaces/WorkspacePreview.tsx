import { Layers3 } from "lucide-react";
import type { Workspace } from "../../model";

export default function WorkspacePreview({
  workspace,
  background,
}: {
  workspace: Workspace;
  background: string;
}) {
  return (
    <span
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
        style={{ backgroundColor: background }}
      >
        {workspace.previewId ? (
          <img
            className="workspace-thumbnail"
            loading="lazy"
            src={`/api/v1/workspaces/${workspace.id}/previews/${workspace.previewId}`}
            alt={`Preview of the first page of ${workspace.name}`}
          />
        ) : (
          <span className="workspace-preview-status">
            {workspace.firstPageId
              ? "Preview available after opening the workspace"
              : "No pages"}
          </span>
        )}
      </span>
      {workspace.previewEmpty && (
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
