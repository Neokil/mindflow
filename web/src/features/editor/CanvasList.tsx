import { Group, Rect, Text } from "react-konva";
import type { CanvasElement } from "../../model";
import { listRows } from "./textLists";
export default function CanvasList({
  element: e,
  onToggle,
}: {
  element: CanvasElement;
  onToggle?: (index: number) => void;
}) {
  const { rows, indent, fontSize, offset } = listRows(e);
  const color = e.color || "#d9a441";
  const box = fontSize * 0.7;
  return (
    <Group
      x={-e.width / 2}
      y={-e.height / 2 + offset}
      clipWidth={e.width}
      clipHeight={Math.max(0, e.height - offset)}
    >
      {rows.map((row) => (
        <Group key={row.index} y={row.top}>
          {e.listStyle === "check" ? (
            <Group
              onPointerDown={
                onToggle
                  ? (event) => {
                      event.cancelBubble = true;
                    }
                  : undefined
              }
              onMouseDown={
                onToggle
                  ? (event) => {
                      event.cancelBubble = true;
                    }
                  : undefined
              }
              onTouchStart={
                onToggle
                  ? (event) => {
                      event.cancelBubble = true;
                    }
                  : undefined
              }
              onPointerClick={
                onToggle
                  ? (event) => {
                      event.cancelBubble = true;
                      onToggle(row.index);
                    }
                  : undefined
              }
            >
              <Rect width={indent} height={fontSize * 1.3} fill="transparent" />
              <Rect
                x={fontSize * 0.15}
                y={(fontSize * 1.3 - box) / 2}
                width={box}
                height={box}
                stroke={color}
                strokeWidth={Math.max(1, fontSize / 18)}
                cornerRadius={2}
                listening={false}
              />
              {e.checked?.includes(row.index) && (
                <Text
                  text="✓"
                  x={fontSize * 0.15}
                  y={(fontSize * 1.3 - box) / 2}
                  width={box}
                  height={box}
                  fontSize={box}
                  fill={color}
                  align="center"
                  listening={false}
                />
              )}
            </Group>
          ) : (
            <Text
              text={e.listStyle === "number" ? `${row.index + 1}.` : "•"}
              width={indent - fontSize * 0.3}
              fontSize={fontSize}
              fill={color}
              lineHeight={1.3}
            />
          )}
          <Text
            x={indent}
            width={Math.max(1, e.width - indent)}
            height={row.height}
            text={row.text}
            fontSize={fontSize}
            fontFamily="Arial"
            fontStyle={
              `${e.bold ? "bold" : ""} ${e.italic ? "italic" : ""}`.trim() ||
              "normal"
            }
            fill={color}
            lineHeight={1.3}
            align={e.align || "left"}
            textDecoration={
              e.checked?.includes(row.index) ? "line-through" : ""
            }
          />
        </Group>
      ))}
    </Group>
  );
}
