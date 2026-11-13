import Konva from "konva";
import { Copy, SlidersHorizontal, Trash2, X } from "lucide-react";
import type { CSSProperties } from "react";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import {
  footprint,
  lassoSelection,
  selectedWithGroup,
  worldPoint,
  zoomAt,
} from "../../geometry";
import type { CanvasElement, Point } from "../../model";
import { clone, duplicateElements, uid } from "../../model";
import { PageSession, localImages } from "../../session";
import ImageCropDialog from "./ImageCropDialog";
import CommentsLayer from "./CommentsLayer";
import CanvasSettings from "./CanvasSettings";
import { creationBounds } from "./creation";
import CanvasControls from "./CanvasControls";
import CanvasScene from "./CanvasScene";
import CanvasTextEditor from "./CanvasTextEditor";
import EditorHelp from "./EditorHelp";
import EditorToolbar from "./EditorToolbar";
import SelectionInspector from "./SelectionInspector";
import type { Camera, Tool } from "./types";
import useEditorShortcuts from "./useEditorShortcuts";

export default function Canvas({
  session,
  refresh = 0,
}: {
  session: PageSession;
  refresh?: number;
}) {
  const host = useRef<HTMLDivElement>(null),
    stage = useRef<Konva.Stage>(null),
    preview = useRef<Konva.Line>(null),
    fileInput = useRef<HTMLInputElement>(null);
  const [size, setSize] = useState({ width: 1000, height: 700 });
  const [camera, setCamera] = useState<Camera>(
    session.camera || { x: 500, y: 350, zoom: 1 },
  );
  const [tool, setTool] = useState<Tool>("select"),
    [selection, setSelection] = useState<string[]>([]),
    [inspector, setInspector] = useState<"canvas" | "object">(),
    [help, setHelp] = useState(false);
  const [color, setColor] = useState("#6c63d9"),
    [fill, setFill] = useState("#e8e4fa"),
    [penWidth, setPenWidth] = useState(3),
    [pending, setPending] = useState<CanvasElement[]>([]),
    [cursor, setCursor] = useState<Point>({ x: 0, y: 0 }),
    [error, setError] = useState("");
  const [editing, setEditing] = useState<string>(),
    [textValue, setTextValue] = useState("");
  const [commentPoint, setCommentPoint] = useState<Point>();
  const [creation, setCreation] = useState<CanvasElement>();
  const [cropping, setCropping] = useState<CanvasElement>();
  const gesture = useRef<
    | {
        kind: "pan" | "lasso" | "pen" | "create";
        element?: CanvasElement;
        start: Point;
        camera: Camera;
        points: Point[];
        additive?: boolean;
      }
    | undefined
  >(undefined);

  const textOriginal = useRef("");
  useEffect(() => {
    session.camera = camera;
  }, [camera, session]);
  const doc = session.doc;
  const selected = doc.elements.filter((e) => selection.includes(e.id));
  useLayoutEffect(() => {
    const observer = new ResizeObserver(([entry]) => {
      const { width, height } = entry.contentRect;
      setSize({ width, height });
    });
    observer.observe(host.current!);
    return () => observer.disconnect();
  }, []);
  useEffect(() => {
    setCamera((c) =>
      c.x === 500 && c.y === 350
        ? { ...c, x: size.width / 2, y: size.height / 2 }
        : c,
    );
  }, [size.width, size.height]);
  useEffect(() => {
    if (!session.editable) {
      setPending([]);
      setCreation(undefined);
      setEditing(undefined);
      setCropping(undefined);
      gesture.current = undefined;
      preview.current?.points([]);
    }
  }, [session.editable]);

  const editElements = (elements: CanvasElement[]) =>
    session.edit({ ...session.doc, elements });
  function patch(values: Partial<CanvasElement>) {
    editElements(
      session.doc.elements.map((e) =>
        selection.includes(e.id) ? { ...e, ...values } : e,
      ),
    );
  }
  function remove() {
    editElements(session.doc.elements.filter((e) => !selection.includes(e.id)));
    setSelection([]);
  }
  function duplicate() {
    const copies = duplicateElements(selected);
    editElements([...session.doc.elements, ...copies]);
    setSelection(copies.map((e) => e.id));
  }
  function group() {
    if (selected.length < 2) return;
    const id = uid();
    patch({ groupId: id });
  }
  function ungroup() {
    patch({ groupId: undefined });
  }
  function reorder(direction: number) {
    const elements = [...session.doc.elements];
    if (direction > 0) {
      for (let i = elements.length - 2; i >= 0; i--)
        if (
          selection.includes(elements[i].id) &&
          !selection.includes(elements[i + 1].id)
        )
          [elements[i], elements[i + 1]] = [elements[i + 1], elements[i]];
    } else {
      for (let i = 1; i < elements.length; i++)
        if (
          selection.includes(elements[i].id) &&
          !selection.includes(elements[i - 1].id)
        )
          [elements[i], elements[i - 1]] = [elements[i - 1], elements[i]];
    }
    editElements(elements);
  }
  function pointer() {
    return (
      stage.current?.getPointerPosition() || {
        x: size.width / 2,
        y: size.height / 2,
      }
    );
  }
  function startText(e: CanvasElement) {
    if (!session.editable) return;
    session.undoStack.push(clone(session.doc));
    session.redoStack = [];
    setEditing(e.id);
    setTextValue(e.text || "");
    textOriginal.current = e.text || "";
    setSelection([e.id]);
    setTool("select");
  }
  function finishText(cancel = false) {
    if (!editing) return;
    const value = cancel ? textOriginal.current : textValue;
    session.edit(
      {
        ...session.doc,
        elements: session.doc.elements.map((e) =>
          e.id === editing ? { ...e, text: value } : e,
        ),
      },
      false,
    );
    setEditing(undefined);
  }
  function beginCreation(p: Point) {
    const element: CanvasElement =
      pending[0] ||
      (tool === "text"
        ? {
            id: uid(),
            type: "text",
            x: p.x,
            y: p.y,
            width: 240,
            height: 100,
            rotation: 0,
            text: "",
            fontSize: 28,
            color,
            align: "center",
            verticalAlign: "middle",
          }
        : {
            id: uid(),
            type: "shape",
            x: p.x,
            y: p.y,
            width: 160,
            height: 120,
            rotation: 0,
            shape: tool as "rectangle",
            fill,
            border: color,
            borderWidth: 2,
          });
    gesture.current = { kind: "create", start: p, camera, points: [], element };
    setCreation({ ...element, x: p.x, y: p.y });
  }
  async function filesAdded(files: File[]) {
    if (!session.editable) return;
    try {
      const elements: CanvasElement[] = [];
      for (const file of files) {
        if (!["image/png", "image/jpeg", "image/webp"].includes(file.type))
          throw new Error("Choose PNG, JPEG, or WebP images.");
        if (file.size > 20 * 1024 * 1024)
          throw new Error("Choose an image up to 20 MB.");
        const url = URL.createObjectURL(file);
        const im = new Image();
        im.src = url;
        await im.decode();
        if (
          im.naturalWidth * im.naturalHeight > 40e6 ||
          Math.max(im.naturalWidth, im.naturalHeight) > 16384
        ) {
          URL.revokeObjectURL(url);
          throw new Error("Image exceeds the supported dimensions.");
        }
        const assetId = `local:${uid()}`;
        localImages.set(assetId, {
          file,
          url,
          width: im.naturalWidth,
          height: im.naturalHeight,
        });
        const scale = Math.min(
          1,
          500 / Math.max(im.naturalWidth, im.naturalHeight),
        );
        elements.push({
          id: uid(),
          type: "image",
          assetId,
          x: 0,
          y: 0,
          width: im.naturalWidth * scale,
          height: im.naturalHeight * scale,
          rotation: 0,
        });
      }
      setPending((previous) => [...previous, ...elements]);
      setTool("select");
      setCursor(worldPoint(pointer(), camera));
      setError("");
    } catch (e) {
      setError((e as Error).message);
    }
  }
  useEditorShortcuts({
    session,
    selected,
    camera,
    color,
    filesAdded,
    pointer,
    editElements,
    setSelection,
    setTool,
    setError,
    duplicate,
    group,
    ungroup,
    remove,
    cancel: () => {
      setPending([]);
      setSelection([]);
      gesture.current = undefined;
      setCreation(undefined);
      preview.current?.points([]);
      setTool("select");
    },
  });

  function mouseDown(e: Konva.KonvaEventObject<MouseEvent>) {
    if (e.evt.button !== 0 && e.evt.button !== 1) return;
    const p = pointer();
    const world = worldPoint(p, camera);
    const target = e.target;
    if (e.evt.button === 1 || e.evt.metaKey || e.evt.ctrlKey) {
      e.evt.preventDefault();
      gesture.current = { kind: "pan", start: p, camera, points: [] };
      return;
    }
    if (tool === "comment" && session.canComment) {
      setCommentPoint(world);
      setTool("select");
      return;
    }
    if (editing) {
      finishText();
      return;
    }
    if (pending.length && session.editable) {
      beginCreation(world);
      return;
    }
    if (tool !== "select" && session.editable) {
      if (tool === "pen") {
        gesture.current = {
          kind: "pen",
          start: world,
          camera,
          points: [world],
        };
        preview.current?.setAttrs({
          points: [world.x, world.y, world.x + 0.01, world.y + 0.01],
          stroke: color,
          strokeWidth: penWidth,
          fill: undefined,
          closed: false,
        });
      } else beginCreation(world);
      return;
    }
    if (target === stage.current) {
      gesture.current = {
        kind: "lasso",
        start: world,
        camera,
        points: [world],
        additive: e.evt.shiftKey,
      };
      if (!e.evt.shiftKey) setSelection([]);
      preview.current?.setAttrs({
        points: [world.x, world.y],
        stroke: "#8a80df",
        strokeWidth: 1 / camera.zoom,
        fill: "#8a80df18",
        closed: true,
      });
    }
  }
  function mouseMove(event: Konva.KonvaEventObject<MouseEvent>) {
    const p = pointer(),
      world = worldPoint(p, camera);
    if (pending.length) setCursor(world);
    const g = gesture.current;
    if (!g) return;
    if (g.kind === "pan") {
      // Camera-only gestures do not need a React render for every pointer sample.
      const position = {
        x: g.camera.x + p.x - g.start.x,
        y: g.camera.y + p.y - g.start.y,
      };
      stage.current?.position(position);
      if (host.current)
        host.current.style.backgroundPosition = `${position.x}px ${position.y}px`;
      return;
    }
    if (g.kind === "create" && g.element) {
      setCreation({
        ...g.element,
        ...creationBounds(g.start, world, event.evt.shiftKey, doc.settings),
      });
      return;
    }
    const last = g.points[g.points.length - 1];
    if (Math.hypot(world.x - last.x, world.y - last.y) * camera.zoom < 2)
      return;
    if (g.points.length >= 20000) return;
    g.points.push(world);
    preview.current?.points(g.points.flatMap((p) => [p.x, p.y]));
    preview.current?.getLayer()?.batchDraw();
  }
  function mouseUp() {
    const g = gesture.current;
    gesture.current = undefined;
    if (!g) return;
    preview.current?.points([]);
    if (g.kind === "pan") {
      const position = stage.current?.position();
      if (position) setCamera({ ...g.camera, ...position });
      return;
    }
    if (g.kind === "create" && session.editable && creation) {
      const distance = Math.hypot(
        pointer().x - (g.start.x * camera.zoom + camera.x),
        pointer().y - (g.start.y * camera.zoom + camera.y),
      );
      const element =
        distance < 3 && g.element
          ? { ...g.element, x: g.start.x, y: g.start.y }
          : creation;
      editElements([...session.doc.elements, element]);
      setCreation(undefined);
      setSelection([element.id]);
      if (pending.length) setPending(pending.slice(1));
      setTool("select");
      if (element.type === "text") startText(element);
      return;
    }
    if (g.kind === "lasso") {
      const ids = lassoSelection(session.doc.elements, g.points);
      setSelection((previous) =>
        g.additive ? [...new Set([...previous, ...ids])] : ids,
      );
    }
    if (g.kind === "pen" && session.editable) {
      const xs = g.points.map((p) => p.x),
        ys = g.points.map((p) => p.y);
      const minX = Math.min(...xs),
        minY = Math.min(...ys);
      const width = Math.max(0.1, Math.max(...xs) - minX),
        height = Math.max(0.1, Math.max(...ys) - minY);
      if (g.points.length === 1)
        g.points.push({ x: minX + 0.1, y: minY + 0.1 });
      const stroke: CanvasElement = {
        id: uid(),
        type: "stroke",
        x: minX + width / 2,
        y: minY + height / 2,
        width,
        height,
        rotation: 0,
        points: g.points.flatMap((p) => [
          (p.x - minX) / width,
          (p.y - minY) / height,
        ]),
        color,
        penWidth,
      };
      editElements([...session.doc.elements, stroke]);
      setSelection([]);
    }
  }
  function selectElement(id: string, shift = false) {
    const ids = selectedWithGroup(session.doc.elements, id);
    if (shift)
      setSelection((previous) =>
        ids.every((id) => previous.includes(id))
          ? previous.filter((id) => !ids.includes(id))
          : [...new Set([...previous, ...ids])],
      );
    else if (!selection.includes(id)) setSelection(ids);
  }
  function fit() {
    if (!doc.elements.length) {
      setCamera({ x: size.width / 2, y: size.height / 2, zoom: 1 });
      return;
    }
    const points = doc.elements.flatMap(footprint);
    const minX = Math.min(...points.map((p) => p.x)),
      maxX = Math.max(...points.map((p) => p.x)),
      minY = Math.min(...points.map((p) => p.y)),
      maxY = Math.max(...points.map((p) => p.y));
    const zoom = Math.min(
      2,
      Math.max(
        0.05,
        Math.min(
          (size.width - 200) / Math.max(1, maxX - minX),
          (size.height - 180) / Math.max(1, maxY - minY),
        ),
      ),
    );
    setCamera({
      x: size.width / 2 - ((minX + maxX) / 2) * zoom,
      y: size.height / 2 - ((minY + maxY) / 2) * zoom,
      zoom,
    });
  }
  const rawGrid = doc.settings.gridSize * camera.zoom;
  const grid =
    rawGrid >= 6 ? rawGrid : rawGrid * 2 ** Math.ceil(Math.log2(6 / rawGrid));
  const backgroundStyle: CSSProperties =
    doc.settings.background === "dots"
      ? {
          backgroundImage:
            "radial-gradient(var(--grid-dot) .8px, transparent .8px)",
          backgroundSize: `${grid}px ${grid}px`,
          backgroundPosition: `${camera.x}px ${camera.y}px`,
        }
      : {};
  const editingElement = doc.elements.find((e) => e.id === editing);
  return (
    <div className="canvas-shell">
      <div
        className="canvas-host"
        ref={host}
        style={{
          ...backgroundStyle,
          cursor: pending.length
            ? "copy"
            : tool === "pen" || tool === "comment"
              ? "crosshair"
              : tool === "text"
                ? "text"
                : "default",
        }}
        onDragOver={(e) => {
          e.preventDefault();
        }}
        onDrop={(e) => {
          e.preventDefault();
          void filesAdded([...e.dataTransfer.files]);
        }}
        onAuxClick={(event) => event.preventDefault()}
        onContextMenu={(event) => event.preventDefault()}
        data-testid="canvas"
      >
        <CanvasScene
          document={doc}
          camera={camera}
          size={size}
          selection={selection}
          editable={session.editable}
          tool={tool}
          editing={editing}
          placement={creation ? undefined : pending[0]}
          creation={creation}
          cursor={cursor}
          refresh={refresh}
          stageRef={stage}
          previewRef={preview}
          isPanning={() => gesture.current?.kind === "pan"}
          onPointerDown={mouseDown}
          onPointerMove={mouseMove}
          onPointerUp={mouseUp}
          onSelectElement={selectElement}
          onSelectionChange={setSelection}
          onObjectSettings={(id) => {
            setSelection(selectedWithGroup(session.doc.elements, id));
            setInspector("object");
          }}
          onTextEdit={startText}
          onElementsChange={editElements}
          onZoom={(delta, mode) =>
            setCamera(
              zoomAt(
                pointer(),
                camera,
                Math.exp(
                  -delta *
                    (mode === 1 ? 16 : mode === 2 ? size.height : 1) *
                    0.0015,
                ),
              ),
            )
          }
        />
        <CommentsLayer
          pageId={session.id}
          camera={camera}
          size={size}
          canComment={session.canComment}
          refresh={refresh}
          draftPoint={commentPoint}
          closeDraft={() => setCommentPoint(undefined)}
        />
        {editingElement && (
          <CanvasTextEditor
            element={editingElement}
            camera={camera}
            value={textValue}
            onChange={(value) => {
              setTextValue(value);
              session.edit(
                {
                  ...session.doc,
                  elements: session.doc.elements.map((el) =>
                    el.id === editing ? { ...el, text: value } : el,
                  ),
                },
                false,
              );
            }}
            onFinish={finishText}
          />
        )}
      </div>
      <EditorToolbar
        tool={tool}
        placing={pending.length > 0}
        editable={session.editable}
        canComment={session.canComment}
        canUndo={session.undoStack.length > 0}
        canRedo={session.redoStack.length > 0}
        onTool={(tool) => {
          setTool(tool);
          setPending([]);
        }}
        onImages={() => fileInput.current?.click()}
        onUndo={() => session.undo()}
        onRedo={() => session.redo()}
      />
      <input
        hidden
        ref={fileInput}
        type="file"
        accept="image/png,image/jpeg,image/webp"
        multiple
        onChange={(e) => {
          void filesAdded([...(e.target.files || [])]);
          e.target.value = "";
        }}
      />
      {(pending.length > 0 || tool !== "select") && (
        <div className="tool-hint">
          {pending.length
            ? `Drag to size image${pending.length > 1 ? ` (${pending.length} remaining)` : ""}`
            : tool === "pen"
              ? "Draw freely. Select a stroke to move or delete it."
              : tool === "text"
                ? "Drag to size a text box."
                : tool === "comment"
                  ? "Click to leave a comment."
                  : `Drag to size a ${tool}. Hold Shift for a square.`}
          <kbd>esc</kbd>
        </div>
      )}
      {tool === "pen" && session.editable && (
        <div className="pen-options">
          <label>
            Color
            <input
              aria-label="Pen color"
              type="color"
              value={color}
              onChange={(e) => setColor(e.target.value)}
            />
          </label>
          <label>
            Width
            <input
              aria-label="Pen thickness"
              type="range"
              min="1"
              max="40"
              value={penWidth}
              onChange={(e) => setPenWidth(+e.target.value)}
            />
            <span>{penWidth}px</span>
          </label>
        </div>
      )}
      <button
        className={`inspector-toggle icon-button ${inspector ? "active" : ""}`}
        title="Canvas settings"
        aria-label="Canvas settings"
        onClick={() =>
          setInspector(inspector === "canvas" ? undefined : "canvas")
        }
      >
        <SlidersHorizontal size={19} />
      </button>
      {inspector === "canvas" && (
        <CanvasSettings
          settings={doc.settings}
          editable={session.editable}
          onSettings={(settings) => session.edit({ ...session.doc, settings })}
          onClose={() => setInspector(undefined)}
        />
      )}
      {inspector === "object" && selected.length > 0 && (
        <SelectionInspector
          selected={selected}
          editable={session.editable}
          color={color}
          fill={fill}
          onColor={setColor}
          onFill={setFill}
          onPatch={patch}
          onDuplicate={duplicate}
          onGroup={group}
          onUngroup={ungroup}
          onReorder={reorder}
          onDelete={remove}
          onEditText={startText}
          onCrop={setCropping}
          onClose={() => setInspector(undefined)}
        />
      )}
      {cropping && session.editable && (
        <ImageCropDialog
          element={cropping}
          onClose={() => setCropping(undefined)}
          onApply={(values) => {
            editElements(
              session.doc.elements.map((element) =>
                element.id === cropping.id
                  ? { ...element, ...values }
                  : element,
              ),
            );
            setCropping(undefined);
          }}
        />
      )}
      {selected.length > 0 && inspector !== "object" && (
        <div className="selection-bar">
          <span>{selected.length} selected</span>
          <button
            className="icon-button"
            aria-label="Selection properties"
            onClick={() => setInspector("object")}
          >
            <SlidersHorizontal size={16} />
          </button>
          <button
            className="icon-button"
            aria-label="Duplicate selection"
            disabled={!session.editable}
            onClick={duplicate}
          >
            <Copy size={16} />
          </button>
          <button
            className="icon-button danger"
            aria-label="Delete selection"
            disabled={!session.editable}
            onClick={remove}
          >
            <Trash2 size={16} />
          </button>
        </div>
      )}
      <CanvasControls
        count={doc.elements.length}
        editable={session.editable}
        snap={doc.settings.snap}
        zoom={camera.zoom}
        onHelp={() => setHelp(true)}
        onSnap={() =>
          session.edit({
            ...session.doc,
            settings: {
              ...session.doc.settings,
              snap: !session.doc.settings.snap,
            },
          })
        }
        onZoom={(factor) =>
          setCamera(
            zoomAt({ x: size.width / 2, y: size.height / 2 }, camera, factor),
          )
        }
        onFit={fit}
      />
      {!doc.elements.length &&
        !pending.length &&
        tool === "select" &&
        session.loaded && (
          <div className="canvas-empty">
            <span className="empty-orbit">✳</span>
            <h2>Every idea starts somewhere.</h2>
            <p>
              Drop an image, write a thought, or draw a little.
              <br />
              There’s room for all of it.
            </p>
          </div>
        )}
      {error && (
        <div className="canvas-toast" role="alert">
          {error}
          <button
            className="icon-button"
            aria-label="Dismiss error"
            onClick={() => setError("")}
          >
            <X size={15} />
          </button>
        </div>
      )}
      {help && <EditorHelp close={() => setHelp(false)} />}
    </div>
  );
}
