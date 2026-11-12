import { memo, useEffect, useLayoutEffect, useRef, useState } from "react";
import type { CSSProperties } from "react";
import Konva from "konva";
import {
  Stage,
  Layer,
  Group,
  Rect,
  Ellipse,
  Line,
  Text,
  Transformer,
  Image as KImage,
} from "react-konva";
import {
  MousePointer2,
  Square,
  Circle,
  Star,
  Type,
  Pencil,
  ImagePlus,
  Undo2,
  Redo2,
  ZoomIn,
  ZoomOut,
  Scan,
  Grid2X2,
  SlidersHorizontal,
  Copy,
  Trash2,
  Group as GroupIcon,
  Ungroup,
  ArrowUp,
  ArrowDown,
  X,
  Keyboard,
  Check,
} from "lucide-react";
import { api } from "./api";
import { getAssetMetadata, getDecodedImage, invalidateMedia } from "./media";
import { PageSession, localImages } from "./session";
import { clone, duplicateElements, uid, validClipboardElements } from "./model";
import type { CanvasElement, Point } from "./model";
import {
  footprint,
  lassoSelection,
  selectedWithGroup,
  snap,
  worldPoint,
  zoomAt,
} from "./geometry";
import { Modal } from "./ui";

type Tool = "select" | "rectangle" | "circle" | "star" | "text" | "pen";
type Camera = { x: number; y: number; zoom: number };
const radiansToDegrees = (r: number) => (r * 180) / Math.PI;
function ImageContent({
  element: e,
  zoom,
  refresh,
}: {
  element: CanvasElement;
  zoom: number;
  refresh: number;
}) {
  const [image, setImage] = useState<HTMLImageElement>();
  const [failed, setFailed] = useState(false);
  useEffect(() => {
    let canceled = false;
    let retry: ReturnType<typeof setTimeout>;
    const local = localImages.get(e.assetId!);
    const load = (url: string) => {
      getDecodedImage(url)
        .then((im) => {
          if (!canceled) {
            setImage(im);
            setFailed(false);
          }
        })
        .catch(() => {
          if (!canceled) setFailed(true);
        });
    };
    if (e.assetId?.startsWith("local:")) {
      if (local) load(local.url);
      return () => {
        canceled = true;
      };
    }
    if (local && !image) load(local.url);
    const run = async () => {
      try {
        const meta = await getAssetMetadata(e.assetId!);
        if (canceled) return;
        const ratio = devicePixelRatio || 1;
        const need =
          Math.max(
            (e.width * zoom * ratio) / meta.width,
            (e.height * zoom * ratio) / meta.height,
          ) * Math.max(meta.width, meta.height);
        const bucket = meta.variants.find((b) => b >= need);
        load(`/api/v1/assets/${e.assetId}${bucket ? `?bucket=${bucket}` : ""}`);
        if (!bucket && need < Math.max(meta.width, meta.height) * 0.8) {
          await api(`/assets/${e.assetId}/variant`, "POST", {
            width: e.width * zoom * ratio,
            height: e.height * zoom * ratio,
          });
          retry = setTimeout(() => {
            invalidateMedia(e.assetId!);
            void run();
          }, 2500);
        }
      } catch {
        if (!image && !local) setFailed(true);
      }
    };
    const delay = setTimeout(run, 300);
    return () => {
      canceled = true;
      clearTimeout(delay);
      clearTimeout(retry);
    };
    // Re-evaluate image resolution only when the actual display requirement changes.
  }, [
    e.assetId,
    Math.round((e.width * zoom) / 64),
    Math.round((e.height * zoom) / 64),
    refresh,
  ]);
  return image ? (
    <KImage
      image={image}
      x={-e.width / 2}
      y={-e.height / 2}
      width={e.width}
      height={e.height}
    />
  ) : (
    <>
      <Rect
        x={-e.width / 2}
        y={-e.height / 2}
        width={e.width}
        height={e.height}
        fill="#ecebf2"
        cornerRadius={3}
      />
      <Text
        text={failed ? "Image unavailable" : "Loading image…"}
        x={-e.width / 2 + 8}
        y={-8}
        width={e.width - 16}
        fontSize={12}
        align="center"
        fill="#88848e"
      />
    </>
  );
}
const ElementContent = memo(function ElementContent({
  e,
  zoom,
  refresh,
}: {
  e: CanvasElement;
  zoom: number;
  refresh: number;
}) {
  if (e.type === "image")
    return <ImageContent element={e} zoom={zoom} refresh={refresh} />;
  if (e.type === "text")
    return (
      <Text
        x={-e.width / 2}
        y={-e.height / 2}
        width={e.width}
        height={e.height}
        text={e.text || ""}
        fontSize={e.fontSize || 24}
        fontFamily="Arial"
        fontStyle={
          `${e.bold ? "bold" : ""} ${e.italic ? "italic" : ""}`.trim() ||
          "normal"
        }
        align={e.align || "left"}
        fill={e.color || "#28282d"}
        lineHeight={1.3}
        wrap="word"
      />
    );
  if (e.type === "stroke")
    return (
      <Line
        points={(e.points || []).map(
          (p, i) => (p - 0.5) * (i % 2 ? e.height : e.width),
        )}
        stroke={e.color || "#28282d"}
        strokeWidth={e.penWidth || 3}
        lineCap="round"
        lineJoin="round"
        hitStrokeWidth={Math.max(10 / zoom, e.penWidth || 3)}
      />
    );
  const props = {
    fill: e.fill === "transparent" ? undefined : e.fill,
    stroke: e.border,
    strokeWidth: e.borderWidth || 0,
  };
  if (e.shape === "circle")
    return <Ellipse radiusX={e.width / 2} radiusY={e.height / 2} {...props} />;
  if (e.shape === "star")
    return (
      <Line
        closed
        points={Array.from({ length: 10 }, (_, i) => {
          const angle = (i * Math.PI) / 5 - Math.PI / 2,
            factor = i % 2 ? 0.45 : 1;
          return [
            ((Math.cos(angle) * e.width) / 2) * factor,
            ((Math.sin(angle) * e.height) / 2) * factor,
          ];
        }).flat()}
        {...props}
      />
    );
  return (
    <Rect
      x={-e.width / 2}
      y={-e.height / 2}
      width={e.width}
      height={e.height}
      {...props}
    />
  );
});
export default function Canvas({
  session,
  refresh = 0,
}: {
  session: PageSession;
  refresh?: number;
}) {
  const host = useRef<HTMLDivElement>(null),
    stage = useRef<Konva.Stage>(null),
    transformer = useRef<Konva.Transformer>(null),
    preview = useRef<Konva.Line>(null),
    fileInput = useRef<HTMLInputElement>(null),
    nodes = useRef(new Map<string, Konva.Group>());
  const [size, setSize] = useState({ width: 1000, height: 700 });
  const [camera, setCamera] = useState<Camera>(
    session.camera || { x: 500, y: 350, zoom: 1 },
  );
  const [tool, setTool] = useState<Tool>("select"),
    [selection, setSelection] = useState<string[]>([]),
    [inspector, setInspector] = useState(false),
    [help, setHelp] = useState(false);
  const [color, setColor] = useState("#6c63d9"),
    [fill, setFill] = useState("#e8e4fa"),
    [penWidth, setPenWidth] = useState(3),
    [pending, setPending] = useState<CanvasElement[]>([]),
    [cursor, setCursor] = useState<Point>({ x: 0, y: 0 }),
    [error, setError] = useState("");
  const [editing, setEditing] = useState<string>(),
    [textValue, setTextValue] = useState("");
  const textarea = useRef<HTMLTextAreaElement>(null);
  const gesture = useRef<
    | {
        kind: "pan" | "lasso" | "pen";
        start: Point;
        camera: Camera;
        points: Point[];
        additive?: boolean;
      }
    | undefined
  >(undefined);
  const drag = useRef<
    { source: string; start: Point; positions: Map<string, Point> } | undefined
  >(undefined);
  const clipboard = useRef<CanvasElement[]>([]);
  const textOriginal = useRef("");
  const transformAnchor = useRef("");
  useEffect(() => {
    session.camera = camera;
  }, [camera, session]);
  const doc = session.doc;
  const selected = doc.elements.filter((e) => selection.includes(e.id));
  const first = selected[0];
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
    const tr = transformer.current;
    if (!tr) return;
    tr.nodes(
      selection
        .map((id) => nodes.current.get(id))
        .filter((n): n is Konva.Group => !!n),
    );
    tr.getLayer()?.batchDraw();
  }, [selection, doc, camera.zoom, editing, session.editable]);
  useEffect(() => {
    if (!session.editable) {
      setPending([]);
      setEditing(undefined);
      gesture.current = undefined;
      preview.current?.points([]);
    }
  }, [session.editable]);
  useEffect(() => {
    if (!editing) return;
    // Let the initiating canvas pointer event finish before taking DOM focus.
    const timer = setTimeout(() => {
      textarea.current?.focus();
      textarea.current?.select();
    }, 0);
    return () => clearTimeout(timer);
  }, [editing]);
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
  function createAt(p: Point) {
    const position = doc.settings.snap
      ? {
          x: snap(p.x, doc.settings.gridSize),
          y: snap(p.y, doc.settings.gridSize),
        }
      : p;
    if (pending.length) {
      const [item, ...rest] = pending;
      const added = { ...item, ...position };
      editElements([...session.doc.elements, added]);
      setPending(rest);
      setSelection([added.id]);
      return;
    }
    const base = {
      id: uid(),
      ...position,
      width: 160,
      height: 120,
      rotation: 0,
    };
    if (tool === "text") {
      const text: CanvasElement = {
        ...base,
        type: "text",
        width: 240,
        height: 100,
        text: "",
        fontSize: 28,
        color,
        align: "left",
      };
      editElements([...session.doc.elements, text]);
      startText(text);
    } else if (["rectangle", "circle", "star"].includes(tool)) {
      const shape: CanvasElement = {
        ...base,
        type: "shape",
        shape: tool as "rectangle",
        fill,
        border: color,
        borderWidth: 2,
      };
      editElements([...session.doc.elements, shape]);
      setSelection([shape.id]);
      setTool("select");
    }
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
  useEffect(() => {
    const onPaste = (event: ClipboardEvent) => {
      if (
        (event.target as HTMLElement)?.closest(
          "input,textarea,dialog,[contenteditable=true]",
        )
      )
        return;
      if (!session.editable) return;
      const directFiles = [...(event.clipboardData?.files || [])];
      const files = directFiles.length
        ? directFiles
        : [...(event.clipboardData?.items || [])]
            .filter((i) => i.kind === "file")
            .map((i) => i.getAsFile())
            .filter((f): f is File => !!f);
      if (files.length) {
        event.preventDefault();
        void filesAdded(files);
        return;
      }
      const raw =
        event.clipboardData?.getData("application/x-mindflow") ||
        event.clipboardData?.getData("text/plain");
      if (raw?.startsWith('{"mindflow":')) {
        try {
          const parsed = JSON.parse(raw);
          if (
            parsed.mindflow === 1 &&
            validClipboardElements(parsed.elements)
          ) {
            const copied = duplicateElements(parsed.elements);
            editElements([...session.doc.elements, ...copied]);
            setSelection(copied.map((e) => e.id));
            event.preventDefault();
          }
        } catch {
          setError("Clipboard data could not be pasted.");
        }
      } else if (raw) {
        const p = worldPoint(pointer(), camera);
        const text: CanvasElement = {
          id: uid(),
          type: "text",
          ...p,
          width: 260,
          height: Math.max(100, raw.split("\n").length * 32),
          rotation: 0,
          text: raw.slice(0, 100000),
          fontSize: 24,
          color,
          align: "left",
        };
        editElements([...session.doc.elements, text]);
        setSelection([text.id]);
        event.preventDefault();
      }
    };
    const onCopy = (event: ClipboardEvent) => {
      if (
        (event.target as HTMLElement)?.closest(
          "input,textarea,dialog,[contenteditable=true]",
        ) ||
        !selected.length
      )
        return;
      clipboard.current = clone(selected);
      event.preventDefault();
      const value = JSON.stringify({ mindflow: 1, elements: selected });
      event.clipboardData?.setData("application/x-mindflow", value);
      event.clipboardData?.setData("text/plain", value);
    };
    const onKey = (e: KeyboardEvent) => {
      if (
        (e.target as HTMLElement)?.closest(
          "input,textarea,select,[contenteditable=true],dialog",
        )
      )
        return;
      const command = e.metaKey || e.ctrlKey;
      if (e.key === "Escape") {
        setPending([]);
        setSelection([]);
        gesture.current = undefined;
        preview.current?.points([]);
        setTool("select");
        return;
      }
      if (command && e.key.toLowerCase() === "a") {
        e.preventDefault();
        setSelection(session.doc.elements.map((e) => e.id));
        return;
      }
      if (!session.editable) return;
      if (command && e.key.toLowerCase() === "z") {
        e.preventDefault();
        e.shiftKey ? session.redo() : session.undo();
      } else if (command && e.key.toLowerCase() === "d") {
        e.preventDefault();
        duplicate();
      } else if (command && e.key.toLowerCase() === "g") {
        e.preventDefault();
        e.shiftKey ? ungroup() : group();
      } else if (e.key === "Delete" || e.key === "Backspace") {
        e.preventDefault();
        remove();
      } else if (!command && e.key.toLowerCase() === "v") setTool("select");
      else if (!command && e.key.toLowerCase() === "p") setTool("pen");
      else if (!command && e.key.toLowerCase() === "t") setTool("text");
    };
    window.addEventListener("paste", onPaste);
    window.addEventListener("copy", onCopy);
    window.addEventListener("keydown", onKey);
    return () => {
      window.removeEventListener("paste", onPaste);
      window.removeEventListener("copy", onCopy);
      window.removeEventListener("keydown", onKey);
    };
  });
  function mouseDown(e: Konva.KonvaEventObject<MouseEvent>) {
    if (e.evt.button !== 0) return;
    const p = pointer();
    const world = worldPoint(p, camera);
    const target = e.target;
    if (e.evt.metaKey || e.evt.ctrlKey) {
      e.evt.preventDefault();
      gesture.current = { kind: "pan", start: p, camera, points: [] };
      return;
    }
    if (editing) {
      finishText();
      return;
    }
    if (pending.length && session.editable) {
      createAt(world);
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
      } else createAt(world);
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
  function mouseMove() {
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
  function commitTransform() {
    const anchor = transformAnchor.current;
    const corner =
      anchor.includes("left") || anchor.includes("right")
        ? anchor.includes("top") || anchor.includes("bottom")
        : false;
    const elements = session.doc.elements.map((e) => {
      if (!selection.includes(e.id)) return e;
      const node = nodes.current.get(e.id);
      if (!node) return e;
      const sx = Math.abs(node.scaleX()),
        sy = Math.abs(node.scaleY());
      const strokePadding = e.type === "stroke" ? e.penWidth || 3 : 0;
      const width = Math.max(
          0.1,
          (e.width + strokePadding) * sx - strokePadding,
        ),
        height = Math.max(0.1, (e.height + strokePadding) * sy - strokePadding);
      const result = {
        ...e,
        x: node.x(),
        y: node.y(),
        width,
        height,
        rotation: (node.rotation() * Math.PI) / 180,
        ...(e.type === "text" && corner
          ? { fontSize: Math.max(1, Math.min(2000, (e.fontSize || 24) * sx)) }
          : {}),
      };
      node.scale({ x: 1, y: 1 });
      return result;
    });
    editElements(elements);
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
  const tools = [
    { id: "select", icon: MousePointer2, label: "Select (V)" },
    { id: "rectangle", icon: Square, label: "Rectangle" },
    { id: "circle", icon: Circle, label: "Circle" },
    { id: "star", icon: Star, label: "Star" },
    { id: "text", icon: Type, label: "Text (T)" },
    { id: "pen", icon: Pencil, label: "Pen (P)" },
  ] as const;
  return (
    <div className="canvas-shell">
      <div
        className="canvas-host"
        ref={host}
        style={{
          ...backgroundStyle,
          cursor: pending.length
            ? "copy"
            : tool === "pen"
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
        onContextMenu={(event) => event.preventDefault()}
        data-testid="canvas"
      >
        <Stage
          ref={stage}
          width={size.width}
          height={size.height}
          x={camera.x}
          y={camera.y}
          scaleX={camera.zoom}
          scaleY={camera.zoom}
          onMouseDown={mouseDown}
          onMouseMove={mouseMove}
          onMouseUp={mouseUp}
          onMouseLeave={() => {
            if (gesture.current) mouseUp();
          }}
          onWheel={(e) => {
            e.evt.preventDefault();
            const next = zoomAt(
              pointer(),
              camera,
              Math.exp(
                -e.evt.deltaY *
                  (e.evt.deltaMode === 1
                    ? 16
                    : e.evt.deltaMode === 2
                      ? size.height
                      : 1) *
                  0.0015,
              ),
            );
            setCamera(next);
          }}
        >
          <Layer>
            {doc.elements.map((e) => {
              const screenX = e.x * camera.zoom + camera.x,
                screenY = e.y * camera.zoom + camera.y,
                r = (Math.hypot(e.width, e.height) * camera.zoom) / 2 + 100;
              if (
                !selection.includes(e.id) &&
                (screenX + r < -size.width ||
                  screenY + r < -size.height ||
                  screenX - r > size.width * 2 ||
                  screenY - r > size.height * 2)
              )
                return null;
              return (
                <Group
                  key={e.id}
                  id={e.id}
                  ref={(node) => {
                    if (node) nodes.current.set(e.id, node);
                    else nodes.current.delete(e.id);
                  }}
                  x={e.x}
                  y={e.y}
                  rotation={radiansToDegrees(e.rotation)}
                  visible={editing !== e.id}
                  draggable={
                    session.editable &&
                    tool === "select" &&
                    !pending.length &&
                    !editing
                  }
                  onMouseDown={(event) => {
                    if (
                      event.evt.metaKey ||
                      event.evt.ctrlKey ||
                      tool !== "select" ||
                      pending.length
                    )
                      return;
                    selectElement(e.id, event.evt.shiftKey);
                  }}
                  onDblClick={() => {
                    if (e.type === "text") startText(e);
                  }}
                  onDragStart={(event) => {
                    if (gesture.current?.kind === "pan") {
                      event.target.stopDrag();
                      return;
                    }
                    const ids = selection.includes(e.id)
                      ? selection
                      : selectedWithGroup(doc.elements, e.id);
                    setSelection(ids);
                    drag.current = {
                      source: e.id,
                      start: { x: event.target.x(), y: event.target.y() },
                      positions: new Map(
                        doc.elements
                          .filter((el) => ids.includes(el.id))
                          .map((el) => [el.id, { x: el.x, y: el.y }]),
                      ),
                    };
                  }}
                  onDragMove={(event) => {
                    const g = drag.current;
                    if (!g) return;
                    const original = g.positions.get(e.id)!;
                    let x = event.target.x(),
                      y = event.target.y();
                    if (doc.settings.snap) {
                      x = snap(x, doc.settings.gridSize);
                      y = snap(y, doc.settings.gridSize);
                    }
                    const dx = x - original.x,
                      dy = y - original.y;
                    for (const [id, p] of g.positions)
                      nodes.current
                        .get(id)
                        ?.position({ x: p.x + dx, y: p.y + dy });
                  }}
                  onDragEnd={() => {
                    const g = drag.current;
                    if (!g) return;
                    drag.current = undefined;
                    editElements(
                      session.doc.elements.map((el) => {
                        const node = nodes.current.get(el.id);
                        return g.positions.has(el.id) && node
                          ? { ...el, x: node.x(), y: node.y() }
                          : el;
                      }),
                    );
                  }}
                >
                  <ElementContent e={e} zoom={camera.zoom} refresh={refresh} />
                </Group>
              );
            })}
          </Layer>
          <Layer>
            <Line
              ref={preview}
              listening={false}
              lineCap="round"
              lineJoin="round"
            />
            {pending[0] && (
              <Group x={cursor.x} y={cursor.y} opacity={0.65} listening={false}>
                <ElementContent
                  e={pending[0]}
                  zoom={camera.zoom}
                  refresh={refresh}
                />
              </Group>
            )}
            <Transformer
              ref={transformer}
              visible={session.editable && !editing && !pending.length}
              rotateEnabled
              flipEnabled={false}
              padding={4}
              anchorSize={8}
              anchorCornerRadius={2}
              borderStroke="#8b80dd"
              anchorStroke="#8b80dd"
              anchorFill="white"
              rotateAnchorOffset={25}
              onTransformStart={() => {
                const tr = transformer.current!;
                const anchor = tr.getActiveAnchor() || "";
                transformAnchor.current = anchor;
                const corner =
                  (anchor.includes("left") || anchor.includes("right")) &&
                  (anchor.includes("top") || anchor.includes("bottom"));
                tr.keepRatio(
                  selected.length > 1 ||
                    (corner &&
                      (first?.type === "image" || first?.type === "text")),
                );
              }}
              onTransformEnd={commitTransform}
              boundBoxFunc={(old, box) =>
                Math.abs(box.width) < Math.min(4, Math.abs(old.width)) ||
                Math.abs(box.height) < Math.min(4, Math.abs(old.height))
                  ? old
                  : box
              }
              anchorDragBoundFunc={(_old, position) => {
                if (
                  !doc.settings.snap ||
                  transformer.current?.getActiveAnchor() === "rotater"
                )
                  return position;
                const world = worldPoint(position, camera);
                return {
                  x:
                    snap(world.x, doc.settings.gridSize) * camera.zoom +
                    camera.x,
                  y:
                    snap(world.y, doc.settings.gridSize) * camera.zoom +
                    camera.y,
                };
              }}
            />
          </Layer>
        </Stage>
        {editingElement && (
          <textarea
            aria-label="Edit canvas text"
            ref={textarea}
            className="canvas-text-editor"
            value={textValue}
            onChange={(e) => {
              setTextValue(e.target.value);
              session.edit(
                {
                  ...session.doc,
                  elements: session.doc.elements.map((el) =>
                    el.id === editing ? { ...el, text: e.target.value } : el,
                  ),
                },
                false,
              );
            }}
            onBlur={() => finishText()}
            onKeyDown={(e) => {
              e.stopPropagation();
              if (e.key === "Escape") {
                e.preventDefault();
                finishText(true);
              }
              if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
                e.preventDefault();
                finishText();
              }
            }}
            style={{
              left: editingElement.x * camera.zoom + camera.x,
              top: editingElement.y * camera.zoom + camera.y,
              width: editingElement.width,
              height: editingElement.height,
              fontSize: editingElement.fontSize,
              fontFamily: "Arial",
              fontWeight: editingElement.bold ? "bold" : "normal",
              fontStyle: editingElement.italic ? "italic" : "normal",
              textAlign: editingElement.align,
              color: editingElement.color,
              transform: `translate(-50%,-50%) rotate(${radiansToDegrees(editingElement.rotation)}deg) scale(${camera.zoom})`,
              lineHeight: 1.3,
            }}
          />
        )}
      </div>
      <div className="canvas-toolbar" aria-label="Drawing tools">
        {tools.map(({ id, icon: Icon, label }) => (
          <button
            key={id}
            className={`tool ${tool === id && !pending.length ? "active" : ""}`}
            title={label}
            aria-label={label}
            aria-pressed={tool === id}
            disabled={!session.editable && id !== "select"}
            onClick={() => {
              setTool(id);
              setPending([]);
            }}
          >
            <Icon size={18} strokeWidth={1.7} />
          </button>
        ))}
        <span className="toolbar-divider" />
        <button
          className="tool"
          title="Add images"
          aria-label="Add images"
          disabled={!session.editable}
          onClick={() => fileInput.current?.click()}
        >
          <ImagePlus size={18} />
        </button>
        <span className="toolbar-divider" />
        <button
          className="tool"
          aria-label="Undo"
          title="Undo"
          disabled={!session.editable || !session.undoStack.length}
          onClick={() => session.undo()}
        >
          <Undo2 size={17} />
        </button>
        <button
          className="tool"
          aria-label="Redo"
          title="Redo"
          disabled={!session.editable || !session.redoStack.length}
          onClick={() => session.redo()}
        >
          <Redo2 size={17} />
        </button>
      </div>
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
            ? `Click to place image${pending.length > 1 ? ` (${pending.length} remaining)` : ""}`
            : tool === "pen"
              ? "Draw freely. Select a stroke to move or delete it."
              : tool === "text"
                ? "Click anywhere to start writing."
                : `Click to place a ${tool}.`}
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
        title="Canvas & selection settings"
        aria-label="Canvas settings"
        onClick={() => setInspector(!inspector)}
      >
        <SlidersHorizontal size={19} />
      </button>
      {inspector && (
        <aside className="inspector">
          <div className="inspector-head">
            <span>
              {selected.length
                ? `${selected.length} selected`
                : "Canvas settings"}
            </span>
            <button
              className="icon-button"
              aria-label="Close settings"
              onClick={() => setInspector(false)}
            >
              <X size={15} />
            </button>
          </div>
          {selected.length ? (
            <>
              <span className="eyebrow">APPEARANCE</span>
              {first?.type !== "image" && (
                <>
                  <label>
                    Color
                    <input
                      aria-label="Element color"
                      type="color"
                      disabled={!session.editable}
                      value={
                        first?.type === "shape"
                          ? first.border || color
                          : first.color || color
                      }
                      onChange={(e) => {
                        setColor(e.target.value);
                        patch(
                          first.type === "shape"
                            ? { border: e.target.value }
                            : { color: e.target.value },
                        );
                      }}
                    />
                  </label>
                  {first?.type === "shape" && (
                    <>
                      <label>
                        Fill
                        <input
                          aria-label="Shape fill"
                          type="color"
                          disabled={!session.editable}
                          value={
                            first.fill === "transparent"
                              ? "#ffffff"
                              : first.fill || fill
                          }
                          onChange={(e) => {
                            setFill(e.target.value);
                            patch({ fill: e.target.value });
                          }}
                        />
                      </label>
                      <button
                        className="text-button"
                        disabled={!session.editable}
                        onClick={() =>
                          patch({
                            fill:
                              first.fill === "transparent"
                                ? fill
                                : "transparent",
                          })
                        }
                      >
                        {first.fill === "transparent"
                          ? "Enable fill"
                          : "Remove fill"}
                      </button>
                      <label>
                        Border thickness
                        <input
                          aria-label="Border thickness"
                          type="number"
                          min="0"
                          max="100"
                          value={first.borderWidth || 0}
                          disabled={!session.editable}
                          onChange={(e) =>
                            patch({
                              borderWidth: Math.max(
                                0,
                                Math.min(100, +e.target.value),
                              ),
                            })
                          }
                        />
                      </label>
                    </>
                  )}
                  {first.type === "stroke" && (
                    <label>
                      Pen thickness
                      <input
                        type="number"
                        aria-label="Stroke thickness"
                        min="1"
                        max="100"
                        value={first.penWidth || 3}
                        disabled={!session.editable}
                        onChange={(e) =>
                          patch({
                            penWidth: Math.max(
                              1,
                              Math.min(100, +e.target.value),
                            ),
                          })
                        }
                      />
                    </label>
                  )}
                  {first.type === "text" && (
                    <>
                      <label>
                        Font size
                        <input
                          aria-label="Font size"
                          type="number"
                          min="1"
                          max="2000"
                          value={first.fontSize}
                          disabled={!session.editable}
                          onChange={(e) =>
                            patch({
                              fontSize: Math.max(
                                1,
                                Math.min(2000, +e.target.value),
                              ),
                            })
                          }
                        />
                      </label>
                      <div className="segmented">
                        <button
                          disabled={!session.editable}
                          aria-label="Bold"
                          aria-pressed={!!first.bold}
                          className={first.bold ? "active" : ""}
                          onClick={() => patch({ bold: !first.bold })}
                        >
                          <b>B</b>
                        </button>
                        <button
                          disabled={!session.editable}
                          aria-label="Italic"
                          aria-pressed={!!first.italic}
                          className={first.italic ? "active" : ""}
                          onClick={() => patch({ italic: !first.italic })}
                        >
                          <i>I</i>
                        </button>
                      </div>
                      <label>
                        Alignment
                        <select
                          value={first.align}
                          disabled={!session.editable}
                          onChange={(e) =>
                            patch({ align: e.target.value as "left" })
                          }
                        >
                          <option value="left">Left</option>
                          <option value="center">Center</option>
                          <option value="right">Right</option>
                        </select>
                      </label>
                      <button
                        className="secondary full"
                        disabled={!session.editable}
                        onClick={() => startText(first)}
                      >
                        Edit text
                      </button>
                    </>
                  )}
                </>
              )}
              <hr />
              <div className="selection-actions">
                <button
                  className="secondary"
                  disabled={!session.editable}
                  title="Duplicate"
                  aria-label="Duplicate selection"
                  onClick={duplicate}
                >
                  <Copy size={15} />
                </button>
                <button
                  className="secondary"
                  disabled={!session.editable || selected.length < 2}
                  title="Group"
                  aria-label="Group selection"
                  onClick={group}
                >
                  <GroupIcon size={15} />
                </button>
                <button
                  className="secondary"
                  disabled={!session.editable || !first?.groupId}
                  title="Ungroup"
                  aria-label="Ungroup selection"
                  onClick={ungroup}
                >
                  <Ungroup size={15} />
                </button>
                <button
                  className="secondary"
                  disabled={!session.editable}
                  title="Bring forward"
                  aria-label="Bring forward"
                  onClick={() => reorder(1)}
                >
                  <ArrowUp size={15} />
                </button>
                <button
                  className="secondary"
                  disabled={!session.editable}
                  title="Send backward"
                  aria-label="Send backward"
                  onClick={() => reorder(-1)}
                >
                  <ArrowDown size={15} />
                </button>
                <button
                  className="secondary danger"
                  disabled={!session.editable}
                  title="Delete"
                  aria-label="Delete selection"
                  onClick={remove}
                >
                  <Trash2 size={15} />
                </button>
              </div>
              <hr />
            </>
          ) : null}
          <span className="eyebrow">CANVAS</span>
          <label>
            Background
            <select
              value={doc.settings.background}
              disabled={!session.editable}
              onChange={(e) =>
                session.edit({
                  ...doc,
                  settings: {
                    ...doc.settings,
                    background: e.target.value as "dots",
                  },
                })
              }
            >
              <option value="dots">Dot grid</option>
              <option value="plain">Plain</option>
            </select>
          </label>
          <label>
            Grid spacing
            <input
              aria-label="Grid spacing"
              type="number"
              min="1"
              max="1000"
              disabled={!session.editable}
              value={doc.settings.gridSize}
              onChange={(e) =>
                session.edit({
                  ...doc,
                  settings: {
                    ...doc.settings,
                    gridSize: Math.max(1, Math.min(1000, +e.target.value)),
                  },
                })
              }
            />
          </label>
          <label className="check-label">
            <input
              type="checkbox"
              checked={doc.settings.snap}
              disabled={!session.editable}
              onChange={(e) =>
                session.edit({
                  ...doc,
                  settings: { ...doc.settings, snap: e.target.checked },
                })
              }
            />
            Snap when moving & resizing
          </label>
        </aside>
      )}
      {selected.length > 0 && !inspector && (
        <div className="selection-bar">
          <span>{selected.length} selected</span>
          <button
            className="icon-button"
            aria-label="Selection properties"
            onClick={() => setInspector(true)}
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
      <div className="canvas-footer">
        <div className="canvas-info">
          <span data-testid="element-count">
            {doc.elements.length}{" "}
            {doc.elements.length === 1 ? "element" : "elements"}
          </span>
          <span className="info-dot">·</span>
          <span>
            {session.editable ? "Cmd / Ctrl + drag to pan" : "Read-only"}
          </span>
        </div>
        <div className="zoom-controls">
          <button
            className="icon-button"
            aria-label="Keyboard shortcuts"
            onClick={() => setHelp(true)}
          >
            <Keyboard size={16} />
          </button>
          <span className="toolbar-divider" />
          <button
            className={`icon-button ${doc.settings.snap ? "active" : ""}`}
            aria-label="Toggle snap to grid"
            aria-pressed={doc.settings.snap}
            disabled={!session.editable}
            onClick={() =>
              session.edit({
                ...doc,
                settings: { ...doc.settings, snap: !doc.settings.snap },
              })
            }
          >
            <Grid2X2 size={16} />
          </button>
          <button
            className="icon-button"
            aria-label="Zoom out"
            onClick={() =>
              setCamera(
                zoomAt(
                  { x: size.width / 2, y: size.height / 2 },
                  camera,
                  1 / 1.2,
                ),
              )
            }
          >
            <ZoomOut size={16} />
          </button>
          <button
            className="zoom-value"
            title="Reset zoom"
            onClick={() =>
              setCamera(
                zoomAt(
                  { x: size.width / 2, y: size.height / 2 },
                  camera,
                  1 / camera.zoom,
                ),
              )
            }
          >
            {Math.round(camera.zoom * 100)}%
          </button>
          <button
            className="icon-button"
            aria-label="Zoom in"
            onClick={() =>
              setCamera(
                zoomAt({ x: size.width / 2, y: size.height / 2 }, camera, 1.2),
              )
            }
          >
            <ZoomIn size={16} />
          </button>
          <button
            className="icon-button"
            aria-label="Fit content"
            onClick={fit}
          >
            <Scan size={16} />
          </button>
        </div>
      </div>
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
      {help && (
        <Modal title="A few helpful shortcuts" close={() => setHelp(false)}>
          <div className="shortcut-list">
            {[
              ["Pan", "Cmd / Ctrl + drag"],
              ["Zoom around cursor", "Scroll"],
              ["Select enclosed elements", "Drag a freehand lasso"],
              ["Add to selection", "Shift + click / lasso"],
              ["Undo / redo", "Cmd / Ctrl + Z / Shift + Z"],
              ["Copy / paste / duplicate", "Cmd / Ctrl + C / V / D"],
              ["Group / ungroup", "Cmd / Ctrl + G / Shift + G"],
              ["Remove selection", "Delete / Backspace"],
              ["Finish text", "Cmd / Ctrl + Enter"],
              ["Cancel", "Escape"],
            ].map(([label, key]) => (
              <div key={label}>
                <span>{label}</span>
                <kbd>{key}</kbd>
              </div>
            ))}
          </div>
          <button className="primary full" onClick={() => setHelp(false)}>
            <Check size={16} />
            Got it
          </button>
        </Modal>
      )}
    </div>
  );
}
