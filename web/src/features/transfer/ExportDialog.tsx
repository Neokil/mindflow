import { useEffect, useState } from "react";
import { api, apiFile } from "../../api";
import type { Page, Workspace } from "../../model";
import { ErrorMessage, Modal, Spinner } from "../../ui";
import { safeFilename } from "./exportGeometry";
export default function ExportDialog({
  workspace,
  pages: initialPages,
  pageId,
  beforeExport,
  close,
}: {
  workspace: Workspace;
  pages?: Page[];
  pageId?: string;
  beforeExport?: (pageIds: string[]) => Promise<void>;
  close: () => void;
}) {
  const [pages, setPages] = useState(initialPages);
  const [scope, setScope] = useState(pageId ? "selected" : "all");
  const [selected, setSelected] = useState<string[]>(pageId ? [pageId] : []);
  const [format, setFormat] = useState("mindflow");
  const [templates, setTemplates] = useState(!pageId);
  const [resolution, setResolution] = useState(2048);
  const [background, setBackground] = useState(true),
    [grid, setGrid] = useState(false);
  const [error, setError] = useState(""),
    [progress, setProgress] = useState("");
  const [ready, setReady] = useState<{ blob: Blob; name: string }>();
  useEffect(() => {
    if (!initialPages)
      void api<Page[]>(`/workspaces/${workspace.id}/pages`)
        .then(setPages)
        .catch((e) => setError(e.message));
  }, [initialPages, workspace.id]);
  async function submit() {
    setError("");
    setReady(undefined);
    setProgress("Saving changes…");
    try {
      const selection = {
        pageIds: scope === "all" ? [] : selected,
        includeTemplates: templates,
      };
      await beforeExport?.(selection.pageIds);
      setProgress("Preparing export…");
      let file: { blob: Blob; name: string };
      if (format === "mindflow") {
        file = {
          blob: await apiFile(`/workspaces/${workspace.id}/export`, selection),
          name: `${safeFilename(workspace.name)}.mindflow`,
        };
      } else {
        const { visualExport } = await import("./exportFiles");
        const snapshot = await api<import("./exportFiles").Snapshot>(
          `/workspaces/${workspace.id}/export/snapshot`,
          "POST",
          selection,
        );
        file = await visualExport(
          snapshot,
          format as "pdf" | "png",
          {
            resolution,
            background,
            grid,
            backgroundColor:
              getComputedStyle(document.documentElement)
                .getPropertyValue("--bg")
                .trim() || "#ffffff",
          },
          setProgress,
        );
      }
      setReady(file);
      const { downloadFile } = await import("./exportFiles");
      downloadFile(file);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setProgress("");
    }
  }
  return (
    <Modal
      title="Export"
      close={() => {
        if (!progress) close();
      }}
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          void submit();
        }}
      >
        <p className="muted">
          Export “{workspace.name}”. Native files keep elements editable. PDF
          and PNG are flattened copies fitted to all elements, with a margin.
        </p>
        <fieldset disabled={!!progress} className="export-fields">
          <label>
            Pages
            <select
              aria-label="Export pages"
              value={scope}
              onChange={(e) => {
                setScope(e.target.value);
                setTemplates(e.target.value === "all");
                setReady(undefined);
              }}
            >
              <option value="all">Whole workspace</option>
              <option value="selected">Selected pages</option>
            </select>
          </label>
          {scope === "selected" && (
            <div className="export-page-list">
              {pages?.map((p) => (
                <label key={p.id} className="export-check">
                  <input
                    type="checkbox"
                    checked={selected.includes(p.id)}
                    onChange={(e) => {
                      setSelected(
                        e.target.checked
                          ? [...selected, p.id]
                          : selected.filter((id) => id !== p.id),
                      );
                      setReady(undefined);
                    }}
                  />
                  {p.name}
                </label>
              ))}
            </div>
          )}
          <label>
            Format
            <select
              aria-label="Export format"
              value={format}
              onChange={(e) => {
                setFormat(e.target.value);
                setReady(undefined);
              }}
            >
              <option value="mindflow">mindflow (.mindflow)</option>
              <option value="pdf">PDF (.pdf)</option>
              <option value="png">PNG images (.png / .zip)</option>
            </select>
          </label>
          {format === "mindflow" ? (
            <>
              <label className="export-check">
                <input
                  type="checkbox"
                  checked={templates}
                  onChange={(e) => {
                    setTemplates(e.target.checked);
                    setReady(undefined);
                  }}
                />
                Include reusable templates
              </label>
              <p className="muted">
                Includes original media. Sharing permissions, comments and user
                details stay in this workspace.
              </p>
            </>
          ) : (
            <>
              <label>
                Longest image edge
                <select
                  aria-label="Export resolution"
                  value={resolution}
                  onChange={(e) => {
                    setResolution(Number(e.target.value));
                    setReady(undefined);
                  }}
                >
                  <option value={1024}>1,024 pixels</option>
                  <option value={2048}>2,048 pixels</option>
                  <option value={4096}>4,096 pixels</option>
                </select>
              </label>
              <label className="export-check">
                <input
                  type="checkbox"
                  checked={background}
                  onChange={(e) => {
                    setBackground(e.target.checked);
                    setReady(undefined);
                  }}
                />
                Include background
              </label>
              <label className="export-check">
                <input
                  type="checkbox"
                  checked={grid}
                  onChange={(e) => {
                    setGrid(e.target.checked);
                    setReady(undefined);
                  }}
                />
                Include dot grid
              </label>
              <p className="muted">
                One PDF page or PNG per canvas page, in tab order. Multiple PNGs
                download together as a ZIP. Embedded PDFs appear as their
                preview. Fine grids are omitted at small scales.
              </p>
            </>
          )}
        </fieldset>
        <ErrorMessage message={error} />
        {progress && (
          <p role="status">
            <Spinner /> {progress}
          </p>
        )}
        {ready && (
          <p role="status">
            Export ready.{" "}
            <button
              type="button"
              className="text-button"
              onClick={async () =>
                (await import("./exportFiles")).downloadFile(ready)
              }
            >
              Download again
            </button>
          </p>
        )}
        <div className="form-actions">
          <button
            type="button"
            className="secondary"
            disabled={!!progress}
            onClick={close}
          >
            Close
          </button>
          <button
            className="primary"
            disabled={
              !!progress || !pages || (scope === "selected" && !selected.length)
            }
          >
            Export
          </button>
        </div>
      </form>
    </Modal>
  );
}
