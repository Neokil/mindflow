import { useEffect, useState } from "react";
import { Group, Layer, Rect, Stage } from "react-konva";
import { api } from "../../api";
import { fitCamera } from "../../geometry";
import type {
  CanvasDocument,
  CanvasElement,
  ElementLink,
  Page,
} from "../../model";
import { ErrorMessage, Modal, Spinner } from "../../ui";
import ElementContent from "./CanvasElement";

export default function ElementLinkDialog({
  workspaceId,
  pageId,
  document,
  selected,
  close,
  apply,
}: {
  workspaceId: string;
  pageId: string;
  document: CanvasDocument;
  selected: CanvasElement[];
  close: () => void;
  apply: (link: ElementLink) => void;
}) {
  const [pages, setPages] = useState<Page[]>([]),
    [targetPage, setTargetPage] = useState(selected[0]?.link?.pageId || pageId),
    [target, setTarget] = useState(selected[0]?.link?.elementId || ""),
    [elements, setElements] = useState<CanvasElement[]>(),
    [error, setError] = useState("");
  useEffect(() => {
    void api<Page[]>(`/workspaces/${workspaceId}/pages`)
      .then((items) => {
        setPages(items);
        setTargetPage((current) =>
          items.some((item) => item.id === current) ? current : pageId,
        );
      })
      .catch((e) => setError(e.message));
  }, [workspaceId, pageId]);
  useEffect(() => {
    let cancelled = false;
    setElements(undefined);
    setError("");
    if (targetPage === pageId) {
      setElements(document.elements);
      return;
    }
    void api<{ document: CanvasDocument }>(`/pages/${targetPage}/document`)
      .then((data) => {
        if (!cancelled) setElements(data.document.elements);
      })
      .catch((e) => {
        if (!cancelled) setError(e.message);
      });
    return () => {
      cancelled = true;
    };
  }, [targetPage, pageId, document]);
  const width = Math.min(560, window.innerWidth - 96),
    height = 240;
  const camera = fitCamera(elements || [], { width, height });
  const available = elements?.filter(
    (element) =>
      targetPage !== pageId ||
      !selected.some((source) => source.id === element.id),
  );
  const valid = available?.some((element) => element.id === target);
  return (
    <Modal title="Element link" close={close}>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          if (valid) apply({ pageId: targetPage, elementId: target });
        }}
      >
        <p className="muted">
          Choose a page, then click a target in its preview or select it below.
          Clicking the linked element will focus its target. Shift-click selects
          it for editing.
        </p>
        <label>
          Target page
          <select
            aria-label="Target page"
            value={targetPage}
            onChange={(e) => {
              setTargetPage(e.target.value);
              setTarget("");
            }}
          >
            {pages.map((page) => (
              <option key={page.id} value={page.id}>
                {page.name}
              </option>
            ))}
          </select>
        </label>
        <ErrorMessage message={error} />
        {!elements && !error ? <Spinner /> : null}
        {elements && (
          <>
            <div className="link-preview" aria-label="Target page preview">
              <Stage
                width={width}
                height={height}
                x={camera.x}
                y={camera.y}
                scaleX={camera.zoom}
                scaleY={camera.zoom}
              >
                <Layer>
                  {elements.map((element) => (
                    <Group
                      key={element.id}
                      x={element.x}
                      y={element.y}
                      rotation={(element.rotation * 180) / Math.PI}
                      opacity={available?.includes(element) ? 1 : 0.3}
                      onPointerClick={() => {
                        if (available?.includes(element)) setTarget(element.id);
                      }}
                    >
                      <ElementContent
                        e={element}
                        zoom={camera.zoom}
                        refresh={0}
                      />
                      {target === element.id && (
                        <Rect
                          listening={false}
                          x={-element.width / 2 - 4 / camera.zoom}
                          y={-element.height / 2 - 4 / camera.zoom}
                          width={element.width + 8 / camera.zoom}
                          height={element.height + 8 / camera.zoom}
                          stroke="#4caf50"
                          strokeWidth={2 / camera.zoom}
                        />
                      )}
                    </Group>
                  ))}
                </Layer>
              </Stage>
            </div>
            <label>
              Target element
              <select
                aria-label="Target element"
                value={valid ? target : ""}
                onChange={(e) => setTarget(e.target.value)}
              >
                <option value="">Choose an element</option>
                {available?.map((element, index) => (
                  <option key={element.id} value={element.id}>
                    {index + 1}.{" "}
                    {element.text?.trim().slice(0, 60) ||
                      element.shape ||
                      element.type}{" "}
                    {element.id.slice(0, 6)}
                  </option>
                ))}
              </select>
            </label>
            {!available?.length && (
              <p className="muted">
                No available target elements on this page.
              </p>
            )}
          </>
        )}
        <div className="form-actions">
          <button className="secondary" type="button" onClick={close}>
            Cancel
          </button>
          <button className="primary" disabled={!valid}>
            Save link
          </button>
        </div>
      </form>
    </Modal>
  );
}
