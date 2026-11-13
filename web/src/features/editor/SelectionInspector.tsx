import {
  ArrowDown,
  ArrowUp,
  Copy,
  Group as GroupIcon,
  Trash2,
  Ungroup,
  X,
} from "lucide-react";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import type { CanvasElement, Point } from "../../model";
import { wholeFontSize } from "../../model";
type Props = {
  anchor: Point;
  selected: CanvasElement[];
  editable: boolean;
  color: string;
  fill: string;
  onColor: (color: string) => void;
  onFill: (color: string) => void;
  onPatch: (values: Partial<CanvasElement>) => void;
  onDuplicate: () => void;
  onGroup: () => void;
  onUngroup: () => void;
  onReorder: (direction: number) => void;
  onDelete: () => void;
  onEditText: (element: CanvasElement) => void;
  onCrop: (element: CanvasElement) => void;
  onClose: () => void;
};
export default function SelectionInspector({
  anchor,
  selected,
  editable,
  color,
  fill,
  onColor,
  onFill,
  onPatch,
  onDuplicate,
  onGroup,
  onUngroup,
  onReorder,
  onDelete,
  onEditText,
  onCrop,
  onClose,
}: Props) {
  useEffect(() => {
    const escape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.stopPropagation();
        onClose();
      }
    };
    document.addEventListener("keydown", escape);
    return () => document.removeEventListener("keydown", escape);
  }, [onClose]);
  const first = selected[0];
  const panel = useRef<HTMLElement>(null);
  const [position, setPosition] = useState(anchor);
  useLayoutEffect(() => {
    const layout = () => {
      const box = panel.current!.getBoundingClientRect();
      const gap = 12;
      const x = Math.max(
        gap,
        Math.min(anchor.x, window.innerWidth - box.width - gap),
      );
      const below = anchor.y + gap;
      const y =
        below + box.height <= window.innerHeight - gap
          ? below
          : Math.max(gap, anchor.y - box.height - gap);
      setPosition({ x, y });
    };
    layout();
    window.addEventListener("resize", layout);
    return () => window.removeEventListener("resize", layout);
  }, [anchor, selected.length, first?.type]);
  return createPortal(
    <aside
      ref={panel}
      className="object-inspector"
      aria-label="Object properties"
      style={{ left: position.x, top: position.y }}
    >
      <div className="inspector-head">
        <span>{`${selected.length} selected`}</span>
        <button
          className="icon-button"
          aria-label="Close settings"
          onClick={onClose}
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
                  disabled={!editable}
                  value={
                    first?.type === "shape"
                      ? first.border || color
                      : first.color || color
                  }
                  onChange={(e) => {
                    onColor(e.target.value);
                    onPatch(
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
                      disabled={!editable}
                      value={
                        first.fill === "transparent"
                          ? "#ffffff"
                          : first.fill || fill
                      }
                      onChange={(e) => {
                        onFill(e.target.value);
                        onPatch({ fill: e.target.value });
                      }}
                    />
                  </label>
                  <button
                    className="text-button"
                    disabled={!editable}
                    onClick={() =>
                      onPatch({
                        fill:
                          first.fill === "transparent" ? fill : "transparent",
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
                      disabled={!editable}
                      onChange={(e) =>
                        onPatch({
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
                    disabled={!editable}
                    onChange={(e) =>
                      onPatch({
                        penWidth: Math.max(1, Math.min(100, +e.target.value)),
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
                      step="1"
                      inputMode="numeric"
                      value={wholeFontSize(first.fontSize || 24)}
                      disabled={!editable}
                      onChange={(event) => {
                        const value = event.target.valueAsNumber;
                        if (Number.isInteger(value))
                          onPatch({ fontSize: wholeFontSize(value) });
                      }}
                    />
                  </label>
                  <div className="segmented">
                    <button
                      disabled={!editable}
                      aria-label="Bold"
                      aria-pressed={!!first.bold}
                      className={first.bold ? "active" : ""}
                      onClick={() => onPatch({ bold: !first.bold })}
                    >
                      <b>B</b>
                    </button>
                    <button
                      disabled={!editable}
                      aria-label="Italic"
                      aria-pressed={!!first.italic}
                      className={first.italic ? "active" : ""}
                      onClick={() => onPatch({ italic: !first.italic })}
                    >
                      <i>I</i>
                    </button>
                  </div>
                  <label>
                    Alignment
                    <select
                      value={first.align}
                      disabled={!editable}
                      onChange={(e) =>
                        onPatch({ align: e.target.value as "left" })
                      }
                    >
                      <option value="left">Left</option>
                      <option value="center">Center</option>
                      <option value="right">Right</option>
                    </select>
                  </label>
                  <label>
                    Vertical alignment
                    <select
                      aria-label="Vertical alignment"
                      value={first.verticalAlign || "top"}
                      disabled={!editable}
                      onChange={(event) =>
                        onPatch({
                          verticalAlign: event.target
                            .value as CanvasElement["verticalAlign"],
                        })
                      }
                    >
                      <option value="top">Top</option>
                      <option value="middle">Middle</option>
                      <option value="bottom">Bottom</option>
                    </select>
                  </label>
                  <button
                    className="secondary full"
                    disabled={!editable}
                    onClick={() => onEditText(first)}
                  >
                    Edit text
                  </button>
                </>
              )}
            </>
          )}
          {first.type === "image" && selected.length === 1 && (
            <button
              className="secondary full"
              disabled={!editable}
              onClick={() => onCrop(first)}
            >
              Crop image
            </button>
          )}
          <hr />
          <div className="selection-actions">
            <button
              className="secondary"
              disabled={!editable}
              title="Duplicate"
              aria-label="Duplicate selection"
              onClick={onDuplicate}
            >
              <Copy size={15} />
            </button>
            <button
              className="secondary"
              disabled={!editable || selected.length < 2}
              title="Group"
              aria-label="Group selection"
              onClick={onGroup}
            >
              <GroupIcon size={15} />
            </button>
            <button
              className="secondary"
              disabled={!editable || !first?.groupId}
              title="Ungroup"
              aria-label="Ungroup selection"
              onClick={onUngroup}
            >
              <Ungroup size={15} />
            </button>
            <button
              className="secondary"
              disabled={!editable}
              title="Bring forward"
              aria-label="Bring forward"
              onClick={() => onReorder(1)}
            >
              <ArrowUp size={15} />
            </button>
            <button
              className="secondary"
              disabled={!editable}
              title="Send backward"
              aria-label="Send backward"
              onClick={() => onReorder(-1)}
            >
              <ArrowDown size={15} />
            </button>
            <button
              className="secondary danger"
              disabled={!editable}
              title="Delete"
              aria-label="Delete selection"
              onClick={onDelete}
            >
              <Trash2 size={15} />
            </button>
          </div>
          <hr />
        </>
      ) : null}
    </aside>,
    document.body,
  );
}
