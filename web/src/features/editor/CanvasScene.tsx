import Konva from "konva";
import type { RefObject } from "react";
import { useEffect, useRef } from "react";
import { Group, Layer, Line, Rect, Stage, Transformer } from "react-konva";
import { selectedWithGroup, snap, worldPoint } from "../../geometry";
import type { CanvasDocument, CanvasElement, Point } from "../../model";
import { wholeFontSize } from "../../model";
import CanvasElementView from "./CanvasElement";
import type { Camera, Tool } from "./types";
type Props = {
  document: CanvasDocument;
  camera: Camera;
  size: { width: number; height: number };
  selection: string[];
  editable: boolean;
  tool: Tool;
  editing?: string;
  placement?: CanvasElement;
  creation?: CanvasElement;
  onObjectSettings: (id: string, position: Point) => void;
  cursor: Point;
  refresh: number;
  stageRef: RefObject<Konva.Stage | null>;
  previewRef: RefObject<Konva.Line | null>;
  isPanning: () => boolean;
  onPointerDown: (event: Konva.KonvaEventObject<MouseEvent>) => void;
  onPointerMove: (event: Konva.KonvaEventObject<MouseEvent>) => void;
  onPointerUp: () => void;
  onZoom: (delta: number, mode: number) => void;
  onSelectElement: (id: string, shift?: boolean) => void;
  onSelectionChange: (ids: string[]) => void;
  onTextEdit: (element: CanvasElement) => void;
  onElementsChange: (elements: CanvasElement[]) => void;
};
const radiansToDegrees = (value: number) => (value * 180) / Math.PI;
export default function CanvasScene({
  document,
  camera,
  size,
  selection,
  editable,
  tool,
  editing,
  placement,
  creation,
  onObjectSettings,
  cursor,
  refresh,
  stageRef,
  previewRef,
  isPanning,
  onPointerDown,
  onPointerMove,
  onPointerUp,
  onZoom,
  onSelectElement,
  onSelectionChange,
  onTextEdit,
  onElementsChange,
}: Props) {
  const nodes = useRef(new Map<string, Konva.Group>()),
    transformer = useRef<Konva.Transformer>(null),
    transformAnchor = useRef("");
  const drag = useRef<
    { source: string; start: Point; positions: Map<string, Point> } | undefined
  >(undefined);
  const selected = document.elements.filter((element) =>
      selection.includes(element.id),
    ),
    first = selected[0];
  useEffect(() => {
    const tr = transformer.current;
    if (!tr) return;
    tr.nodes(
      selection
        .map((id) => nodes.current.get(id))
        .filter((node): node is Konva.Group => !!node),
    );
    tr.getLayer()?.batchDraw();
  }, [selection, document, camera.zoom, editing, editable]);
  useEffect(() => {
    const stage = stageRef.current;
    if (!stage) return;
    stage.container().style.cursor = "";
    // Recheck after Konva redraws when text editing or a tool change ends.
    const frame = requestAnimationFrame(() => {
      const point = stage.getPointerPosition();
      const hit = point && stage.getIntersection(point);
      if (
        editable &&
        tool === "select" &&
        !placement &&
        !editing &&
        hit?.findAncestor(".canvas-element", true)
      )
        stage.container().style.cursor = "grab";
    });
    return () => cancelAnimationFrame(frame);
  }, [editable, tool, placement, editing, stageRef]);
  function updateCursor(event: Konva.KonvaEventObject<MouseEvent>) {
    const container = stageRef.current?.container();
    if (
      !container ||
      event.target.getParent()?.getClassName() === "Transformer"
    )
      return;
    container.style.cursor =
      isPanning() || drag.current
        ? "grabbing"
        : event.evt.metaKey || event.evt.ctrlKey
          ? "grab"
          : editable &&
              tool === "select" &&
              !placement &&
              !editing &&
              event.target.findAncestor(".canvas-element", true)
            ? "grab"
            : "";
  }
  function commitTransform() {
    const anchor = transformAnchor.current;
    const corner =
      anchor.includes("left") || anchor.includes("right")
        ? anchor.includes("top") || anchor.includes("bottom")
        : false;
    const elements = document.elements.map((e) => {
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
          ? { fontSize: wholeFontSize((e.fontSize || 24) * sx) }
          : {}),
      };
      node.scale({ x: 1, y: 1 });
      return result;
    });
    onElementsChange(elements);
  }

  return (
    <Stage
      ref={stageRef}
      width={size.width}
      height={size.height}
      x={camera.x}
      y={camera.y}
      scaleX={camera.zoom}
      scaleY={camera.zoom}
      onMouseDown={(event) => {
        onPointerDown(event);
        updateCursor(event);
      }}
      onMouseOver={updateCursor}
      onMouseMove={(event) => {
        updateCursor(event);
        onPointerMove(event);
      }}
      onMouseUp={(event) => {
        onPointerUp();
        updateCursor(event);
      }}
      onMouseLeave={() => {
        onPointerUp();
        const container = stageRef.current?.container();
        if (container) container.style.cursor = "";
      }}
      onWheel={(event) => {
        event.evt.preventDefault();
        onZoom(event.evt.deltaY, event.evt.deltaMode);
      }}
    >
      <Layer>
        {document.elements.map((e) => {
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
              name="canvas-element"
              ref={(node) => {
                if (node) nodes.current.set(e.id, node);
                else nodes.current.delete(e.id);
              }}
              x={e.x}
              y={e.y}
              rotation={radiansToDegrees(e.rotation)}
              visible={editing !== e.id}
              draggable={
                editable &&
                tool === "select" &&
                !placement &&
                !editing &&
                !creation
              }
              onMouseDown={(event) => {
                if (
                  event.evt.button !== 0 ||
                  event.evt.metaKey ||
                  event.evt.ctrlKey ||
                  tool !== "select" ||
                  !!placement
                )
                  return;
                onSelectElement(e.id, event.evt.shiftKey);
              }}
              onContextMenu={(event) => {
                event.evt.preventDefault();
                onObjectSettings(e.id, {
                  x: event.evt.clientX,
                  y: event.evt.clientY,
                });
              }}
              onDblClick={() => {
                if (e.type === "text") onTextEdit(e);
              }}
              onDragStart={(event) => {
                const container = stageRef.current?.container();
                if (container) container.style.cursor = "grabbing";
                if (isPanning()) {
                  event.target.stopDrag();
                  return;
                }
                const ids = selection.includes(e.id)
                  ? selection
                  : selectedWithGroup(document.elements, e.id);
                onSelectionChange(ids);
                drag.current = {
                  source: e.id,
                  start: { x: event.target.x(), y: event.target.y() },
                  positions: new Map(
                    document.elements
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
                if (document.settings.snap) {
                  x = snap(x, document.settings.gridSize);
                  y = snap(y, document.settings.gridSize);
                }
                const dx = x - original.x,
                  dy = y - original.y;
                for (const [id, p] of g.positions)
                  nodes.current.get(id)?.position({ x: p.x + dx, y: p.y + dy });
              }}
              onDragEnd={() => {
                const container = stageRef.current?.container();
                if (container) container.style.cursor = "grab";
                const g = drag.current;
                if (!g) return;
                drag.current = undefined;
                onElementsChange(
                  document.elements.map((el) => {
                    const node = nodes.current.get(el.id);
                    return g.positions.has(el.id) && node
                      ? { ...el, x: node.x(), y: node.y() }
                      : el;
                  }),
                );
              }}
            >
              <CanvasElementView e={e} zoom={camera.zoom} refresh={refresh} />
            </Group>
          );
        })}
      </Layer>
      <Layer>
        {creation && (
          <Group x={creation.x} y={creation.y} opacity={0.65} listening={false}>
            <Rect
              x={-creation.width / 2}
              y={-creation.height / 2}
              width={creation.width}
              height={creation.height}
              stroke="#8b80dd"
              strokeWidth={1 / camera.zoom}
              dash={[4 / camera.zoom, 4 / camera.zoom]}
            />
            <CanvasElementView
              e={creation}
              zoom={camera.zoom}
              refresh={refresh}
            />
          </Group>
        )}
        <Line
          ref={previewRef}
          listening={false}
          lineCap="round"
          lineJoin="round"
        />
        {placement && (
          <Group x={cursor.x} y={cursor.y} opacity={0.65} listening={false}>
            <CanvasElementView
              e={placement}
              zoom={camera.zoom}
              refresh={refresh}
            />
          </Group>
        )}
        <Transformer
          ref={transformer}
          visible={editable && !editing && !placement && !creation}
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
                (corner && (first?.type === "image" || first?.type === "text")),
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
              !document.settings.snap ||
              transformer.current?.getActiveAnchor() === "rotater"
            )
              return position;
            const world = worldPoint(position, camera);
            return {
              x:
                snap(world.x, document.settings.gridSize) * camera.zoom +
                camera.x,
              y:
                snap(world.y, document.settings.gridSize) * camera.zoom +
                camera.y,
            };
          }}
        />
      </Layer>
    </Stage>
  );
}
