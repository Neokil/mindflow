import useCanvasInput from "./useCanvasInput";
import type { CanvasInteraction } from "./useCanvasInput";
import ColorInput from "../../ColorInput";
import useCanvasAccent from "./useCanvasAccent";
import TemplateLibrary from "./TemplateLibrary";
import { instantiateTemplate } from "./templates";
import Konva from "konva";
import {
  Copy,
  BookmarkPlus,
  LockKeyhole,
  LockKeyholeOpen,
  SlidersHorizontal,
  Trash2,
  X,
} from "lucide-react";
import type { CSSProperties, SetStateAction } from "react";
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import {
  fitCamera,
  lassoSelection,
  selectedWithGroup,
  worldPoint,
  zoomAt,
} from "../../geometry";
import type { CanvasElement, Point } from "../../model";
import {
  canvasElementDefaults,
  clone,
  duplicateElements,
  uid,
} from "../../model";
import { PageSession, localImages } from "../../session";
import ImageCropDialog from "./ImageCropDialog";
import CommentsLayer from "./CommentsLayer";
import CanvasSettings from "./CanvasSettings";
import { retainCheckedLines } from "./textLists";
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
  const accent = useCanvasAccent();
  const interaction = useRef<CanvasInteraction>(null);
  const host = useRef<HTMLDivElement>(null),
    stage = useRef<Konva.Stage>(null),
    preview = useRef<Konva.Line>(null),
    fileInput = useRef<HTMLInputElement>(null);
  const [size, setSize] = useState({ width: 0, height: 0 });
  const [camera, setCamera] = useState<Camera>(
    session.camera || { x: 500, y: 350, zoom: 1 },
  );
  const [tool, setTool] = useState<Tool>("select"),
    [selection, updateSelection] = useState<string[]>([]),
    [canvasSettingsOpen, setCanvasSettingsOpen] = useState(false),
    [objectSettingsAnchor, setObjectSettingsAnchor] = useState<Point>(),
    [help, setHelp] = useState(false);
  const [markerColor, setMarkerColor] = useState("#a4d65e");
  const [markerWidth, setMarkerWidth] = useState(24);
  const [penColor, setPenColor] = useState(accent),
    [pencilOnly, setPencilOnly] = useState(false),
    [penWidth, setPenWidth] = useState(3),
    [pending, setPending] = useState<CanvasElement[]>([]),
    [cursor, setCursor] = useState<Point>({ x: 0, y: 0 }),
    [error, setError] = useState("");
  function setSelection(value: SetStateAction<string[]>) {
    setObjectSettingsAnchor(undefined);
    updateSelection(value);
  }
  const [editing, setEditing] = useState<string>(),
    [textValue, setTextValue] = useState("");
  const [templatesOpen, setTemplatesOpen] = useState<"save" | "browse">();
  const [templatePlacement, setTemplatePlacement] = useState<CanvasElement[]>(
    [],
  );
  const [commentPoint, setCommentPoint] = useState<Point>();
  const [creation, setCreation] = useState<CanvasElement>();
  const [cropping, setCropping] = useState<CanvasElement>();
  const gesture = useRef<
    | {
        kind: "pan" | "lasso" | "pen" | "create" | "comment" | "template";
        element?: CanvasElement;
        highlight?: boolean;
        start: Point;
        camera: Camera;
        points: Point[];
        additive?: boolean;
      }
    | undefined
  >(undefined);

  const textOriginal = useRef<CanvasElement | undefined>(undefined);
  useEffect(() => {
    session.camera = camera;
  }, [camera, session]);
  const doc = session.doc;
  const defaults = canvasElementDefaults(doc.settings);
  const selected = doc.elements.filter((e) => selection.includes(e.id));
  const selectionLocked = selected.some((element) => element.locked);
  useLayoutEffect(() => {
    const observer = new ResizeObserver(([entry]) => {
      const { width, height } = entry.contentRect;
      setSize({ width, height });
    });
    observer.observe(host.current!);
    return () => observer.disconnect();
  }, []);
  const cameraInitialized = useRef(Boolean(session.camera));
  useLayoutEffect(() => {
    if (!cameraInitialized.current && size.width > 0 && size.height > 0) {
      cameraInitialized.current = true;
      setCamera(fitCamera(doc.elements, size));
    }
  }, [size, doc.elements]);
  useEffect(() => {
    if (!session.editable) {
      setPending([]);
      setTemplatePlacement([]);
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
    if (selectionLocked) return;
    editElements(
      session.doc.elements.map((e) =>
        selection.includes(e.id) ? { ...e, ...values } : e,
      ),
    );
  }
  function remove() {
    if (selectionLocked) return;
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
  function toggleLock() {
    if (editing) finishText();
    const groups = new Set(
      selected.map((element) => element.groupId).filter(Boolean),
    );
    editElements(
      session.doc.elements.map((element) =>
        groups.has(element.groupId)
          ? { ...element, locked: !selectionLocked }
          : element,
      ),
    );
  }
  function ungroup() {
    patch({ groupId: undefined });
  }
  function reorder(direction: number) {
    if (selectionLocked) return;
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
    if (!session.editable || e.locked) return;
    session.undoStack.push(clone(session.doc));
    session.redoStack = [];
    setEditing(e.id);
    setTextValue(e.text || "");
    textOriginal.current = clone(e);
    setSelection([e.id]);
    setTool("select");
  }
  function finishText(cancel = false) {
    if (!editing) return;
    const original = textOriginal.current;
    session.edit(
      {
        ...session.doc,
        elements: session.doc.elements.map((e) =>
          e.id === editing
            ? cancel && original
              ? original
              : { ...e, text: textValue }
            : e,
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
            color: defaults.defaultTextColor,
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
            fill: defaults.defaultFill,
            border: defaults.defaultBorder,
            borderWidth: 2,
          });
    gesture.current = { kind: "create", start: p, camera, points: [], element };
    setCreation({ ...element, ...creationBounds(p, p, false, doc.settings) });
  }
  async function filesAdded(files: File[]) {
    if (!session.editable) return;
    setTemplatePlacement([]);
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
    textColor: defaults.defaultTextColor,
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
      setTemplatePlacement([]);
      setTemplatesOpen(undefined);
      setSelection([]);
      gesture.current = undefined;
      setCreation(undefined);
      preview.current?.points([]);
      setTool("select");
    },
  });

  function pointerDown(e: Konva.KonvaEventObject<PointerEvent>) {
    if (e.evt.button !== 0 && e.evt.button !== 1) return;
    const p = pointer();
    const world = worldPoint(p, camera);
    const target = e.target;
    if (
      tool === "pan" ||
      ((tool === "pen" || tool === "highlighter") &&
        pencilOnly &&
        e.evt.pointerType === "touch") ||
      e.evt.button === 1 ||
      e.evt.metaKey ||
      e.evt.ctrlKey
    ) {
      e.evt.preventDefault();
      gesture.current = { kind: "pan", start: p, camera, points: [] };
      return;
    }
    if (tool === "comment" && session.canComment) {
      gesture.current = { kind: "comment", start: world, camera, points: [] };
      return;
    }
    if (editing) {
      finishText();
      return;
    }
    if (templatePlacement.length && session.editable) {
      setCursor(world);
      gesture.current = { kind: "template", start: world, camera, points: [] };
      return;
    }
    if (pending.length && session.editable) {
      beginCreation(world);
      return;
    }
    if (tool !== "select" && session.editable) {
      if (tool === "pen" || tool === "highlighter") {
        gesture.current = {
          kind: "pen",
          highlight: tool === "highlighter",
          start: world,
          camera,
          points: [world],
        };
        preview.current?.setAttrs({
          points: [world.x, world.y, world.x + 0.01, world.y + 0.01],
          stroke: tool === "highlighter" ? markerColor : penColor,
          strokeWidth: tool === "highlighter" ? markerWidth : penWidth,
          opacity: tool === "highlighter" ? 0.35 : 1,
          globalCompositeOperation:
            tool === "highlighter" ? "multiply" : "source-over",
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
        stroke: accent,
        opacity: 1,
        globalCompositeOperation: "source-over",
        strokeWidth: 1 / camera.zoom,
        fill: `${accent}18`,
        closed: true,
      });
    }
  }
  function pointerMove(event: Konva.KonvaEventObject<PointerEvent>) {
    const p = pointer(),
      world = worldPoint(p, camera);
    if (pending.length || templatePlacement.length) setCursor(world);
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
    if (g.kind === "comment" || g.kind === "template") return;
    const last = g.points[g.points.length - 1];
    if (Math.hypot(world.x - last.x, world.y - last.y) * camera.zoom < 2)
      return;
    if (g.points.length >= 20000) return;
    g.points.push(world);
    preview.current?.points(g.points.flatMap((p) => [p.x, p.y]));
    preview.current?.getLayer()?.batchDraw();
  }
  function pointerUp(event: PointerEvent) {
    const g = gesture.current;
    gesture.current = undefined;
    if (!g) return;
    preview.current?.points([]);
    if (g.kind === "comment" && session.canComment) {
      setCommentPoint(worldPoint(pointer(), camera));
      setTool("select");
      return;
    }
    if (g.kind === "template" && session.editable) {
      const elements = instantiateTemplate(
        templatePlacement,
        worldPoint(pointer(), camera),
      );
      editElements([...session.doc.elements, ...elements]);
      setSelection(elements.map((element) => element.id));
      setTemplatePlacement([]);
      return;
    }
    if (g.kind === "pan") {
      const p = pointer();
      setCamera({
        ...g.camera,
        x: g.camera.x + p.x - g.start.x,
        y: g.camera.y + p.y - g.start.y,
      });
      return;
    }
    if (g.kind === "create" && session.editable && g.element) {
      const distance = Math.hypot(
        pointer().x - (g.start.x * camera.zoom + camera.x),
        pointer().y - (g.start.y * camera.zoom + camera.y),
      );
      const element =
        distance < 3
          ? { ...g.element, x: g.start.x, y: g.start.y }
          : {
              ...g.element,
              ...creationBounds(
                g.start,
                worldPoint(pointer(), camera),
                event.shiftKey,
                doc.settings,
              ),
            };
      editElements([...session.doc.elements, element]);
      setCreation(undefined);
      setSelection([element.id]);
      if (pending.length) setPending(pending.slice(1));
      setTool("select");
      if (element.type === "text") startText(element);
      return;
    }
    if ((g.kind === "lasso" || g.kind === "pen") && g.points.length < 20000) {
      const last = g.points[g.points.length - 1];
      const release = worldPoint(pointer(), camera);
      if (last.x !== release.x || last.y !== release.y) g.points.push(release);
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
        color: g.highlight ? markerColor : penColor,
        penWidth: g.highlight ? markerWidth : penWidth,
        highlight: g.highlight || undefined,
      };
      editElements([...session.doc.elements, stroke]);
      setSelection([]);
    }
  }
  const cancelGesture = useCallback(() => {
    gesture.current = undefined;
    setCreation(undefined);
    preview.current?.points([]);
    stage.current?.position({ x: camera.x, y: camera.y });
    if (host.current)
      host.current.style.backgroundPosition = `${camera.x}px ${camera.y}px`;
  }, [camera]);
  const input = useCanvasInput({
    stage,
    scene: interaction,
    camera,
    onCamera: setCamera,
    start: pointerDown,
    move: pointerMove,
    end: pointerUp,
    cancel: cancelGesture,
  });
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
    setCamera(fitCamera(doc.elements, size));
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
        {...input.capture}
        style={{
          ...backgroundStyle,
          backgroundColor: doc.settings.backgroundColor,
          cursor:
            pending.length || templatePlacement.length
              ? "copy"
              : tool === "pan"
                ? "grab"
                : tool === "pen" || tool === "highlighter" || tool === "comment"
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
          interactionRef={interaction}
          coarse={input.coarse}
          navigating={input.navigating}
          allowPointer={input.allowed}
          accent={accent}
          document={doc}
          camera={camera}
          size={size}
          selection={selection}
          editable={session.editable}
          tool={tool}
          editing={editing}
          placement={creation ? undefined : pending[0]}
          templatePlacement={templatePlacement}
          creation={creation}
          cursor={cursor}
          refresh={refresh}
          stageRef={stage}
          previewRef={preview}
          isPanning={() => gesture.current?.kind === "pan"}
          onPointerDown={input.down}
          onPointerMove={input.move}
          onSelectElement={selectElement}
          onSelectionChange={setSelection}
          onObjectSettings={(id, position) => {
            setSelection(selectedWithGroup(session.doc.elements, id));
            setObjectSettingsAnchor(position);
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
                    el.id === editing
                      ? {
                          ...el,
                          text: value,
                          checked:
                            el.listStyle === "check"
                              ? retainCheckedLines(
                                  el.text || "",
                                  value,
                                  el.checked,
                                )
                              : undefined,
                        }
                      : el,
                  ),
                },
                false,
              );
            }}
            onListStyle={(listStyle) =>
              session.edit(
                {
                  ...session.doc,
                  elements: session.doc.elements.map((el) =>
                    el.id === editing
                      ? {
                          ...el,
                          listStyle,
                          checked:
                            listStyle === "check" ? el.checked : undefined,
                          ...(listStyle
                            ? {
                                align: "left" as const,
                                verticalAlign: "top" as const,
                              }
                            : {}),
                        }
                      : el,
                  ),
                },
                false,
              )
            }
            onFinish={finishText}
          />
        )}
      </div>
      <EditorToolbar
        tool={tool}
        placing={pending.length > 0 || templatePlacement.length > 0}
        editable={session.editable}
        canComment={session.canComment}
        canUndo={session.undoStack.length > 0}
        canRedo={session.redoStack.length > 0}
        onTool={(tool) => {
          setTool(tool);
          setPending([]);
          setTemplatePlacement([]);
        }}
        onTemplates={() => setTemplatesOpen("browse")}
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
      {(pending.length > 0 ||
        templatePlacement.length > 0 ||
        tool !== "select") && (
        <div className="tool-hint">
          {templatePlacement.length
            ? "Click to place template."
            : pending.length
              ? `Drag to size image${pending.length > 1 ? ` (${pending.length} remaining)` : ""}`
              : tool === "pan"
                ? "Drag to pan. Use two fingers to pan and pinch to zoom."
                : tool === "pen" || tool === "highlighter"
                  ? "Draw freely. Select a stroke to move or delete it."
                  : tool === "text"
                    ? "Drag to size a text box."
                    : tool === "comment"
                      ? "Click to leave a comment."
                      : `Drag to size a ${tool}. Hold Shift for a square.`}
          <kbd>esc</kbd>
        </div>
      )}
      {(tool === "pen" || tool === "highlighter") && session.editable && (
        <div className="pen-options">
          <label>
            Color
            <ColorInput
              label="Pen color"
              value={tool === "highlighter" ? markerColor : penColor}
              onChange={tool === "highlighter" ? setMarkerColor : setPenColor}
            />
          </label>
          <label className="check-label">
            <input
              type="checkbox"
              aria-label="Pencil only"
              checked={pencilOnly}
              onChange={(event) => setPencilOnly(event.target.checked)}
            />
            Pencil only
          </label>
          <label>
            Width
            <input
              aria-label="Pen thickness"
              type="range"
              min="1"
              max="40"
              value={tool === "highlighter" ? markerWidth : penWidth}
              onChange={(e) =>
                (tool === "highlighter" ? setMarkerWidth : setPenWidth)(
                  +e.target.value,
                )
              }
            />
            <span>{tool === "highlighter" ? markerWidth : penWidth}px</span>
          </label>
        </div>
      )}
      <button
        className={`inspector-toggle icon-button ${canvasSettingsOpen ? "active" : ""}`}
        title="Canvas settings"
        aria-label="Canvas settings"
        onClick={() => setCanvasSettingsOpen(!canvasSettingsOpen)}
      >
        <SlidersHorizontal size={19} />
      </button>
      {canvasSettingsOpen && (
        <CanvasSettings
          settings={doc.settings}
          editable={session.editable}
          onSettings={(settings) => session.edit({ ...session.doc, settings })}
          onClose={() => setCanvasSettingsOpen(false)}
        />
      )}
      {objectSettingsAnchor && selected.length > 0 && (
        <SelectionInspector
          anchor={objectSettingsAnchor}
          selected={selected}
          editable={session.editable && !selectionLocked}
          canUnlock={session.editable && selectionLocked}
          onLock={toggleLock}
          onSaveTemplate={() => setTemplatesOpen("save")}
          canSaveTemplate={session.canWrite}
          defaults={defaults}
          onPatch={patch}
          onDuplicate={duplicate}
          onGroup={group}
          onUngroup={ungroup}
          onReorder={reorder}
          onDelete={remove}
          onEditText={startText}
          onCrop={setCropping}
          onClose={() => setObjectSettingsAnchor(undefined)}
        />
      )}
      {templatesOpen && (
        <TemplateLibrary
          session={session}
          selection={selected}
          creating={templatesOpen === "save"}
          close={() => setTemplatesOpen(undefined)}
          insert={(elements) => {
            setPending([]);
            setSelection([]);
            setTool("select");
            setTemplatePlacement(elements);
            setCursor(worldPoint(pointer(), camera));
          }}
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
      {selected.length > 0 && !editing && (
        <div className="selection-bar">
          <span>{selected.length} selected</span>
          <button
            className="icon-button"
            aria-label="Selection properties"
            onClick={(event) => {
              const box = event.currentTarget.getBoundingClientRect();
              setObjectSettingsAnchor({ x: box.left, y: box.top });
            }}
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
            className="icon-button"
            aria-label="Save as template"
            title="Save as template"
            disabled={!session.canWrite || selected.length > 500}
            onClick={() => setTemplatesOpen("save")}
          >
            <BookmarkPlus size={16} />
          </button>
          {selected.some((element) => element.groupId) && (
            <button
              className="icon-button"
              aria-label={selectionLocked ? "Unlock groups" : "Lock groups"}
              title={selectionLocked ? "Unlock groups" : "Lock groups"}
              disabled={!session.editable}
              onClick={toggleLock}
            >
              {selectionLocked ? (
                <LockKeyholeOpen size={16} />
              ) : (
                <LockKeyhole size={16} />
              )}
            </button>
          )}
          <button
            className="icon-button danger"
            aria-label="Delete selection"
            disabled={!session.editable || selectionLocked}
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
