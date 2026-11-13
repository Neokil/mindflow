import ColorInput from "../../ColorInput";
import {
  ArrowDown,
  ArrowUp,
  Copy,
  LockKeyhole,
  LockKeyholeOpen,
  Group as GroupIcon,
  Trash2,
  Ungroup,
  X,
} from "lucide-react";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import type { CanvasElement, CanvasElementDefaults, Point } from "../../model";
import { wholeFontSize } from "../../model";
type Props = {
  anchor: Point;
  selected: CanvasElement[];
  editable: boolean;
  canUnlock: boolean;
  onLock: () => void;
  onSaveTemplate: () => void;
  canSaveTemplate: boolean;
  defaults: CanvasElementDefaults;
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
  canUnlock,
  onLock,
  onSaveTemplate,
  canSaveTemplate,
  defaults,
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
  const locked = selected.some((element) => element.locked);
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
          {selected.some((element) => element.groupId) && (
            <button
              className="secondary full"
              disabled={!editable && !canUnlock}
              onClick={onLock}
            >
              {locked ? (
                <LockKeyholeOpen size={15} />
              ) : (
                <LockKeyhole size={15} />
              )}
              {locked ? "Unlock groups" : "Lock groups"}
            </button>
          )}
          {locked && (
            <p className="muted">Unlock the group to edit its elements.</p>
          )}
          <span className="eyebrow">APPEARANCE</span>
          {first?.type !== "image" && (
            <>
              <label>
                Color
                <ColorInput
                  label="Element color"
                  disabled={!editable}
                  value={
                    first.type === "shape"
                      ? first.border || defaults.defaultBorder
                      : first.color || defaults.defaultTextColor
                  }
                  onChange={(color) =>
                    onPatch(
                      first.type === "shape" ? { border: color } : { color },
                    )
                  }
                />
              </label>
              {first?.type === "shape" && (
                <>
                  <label>
                    Fill
                    <ColorInput
                      label="Shape fill"
                      disabled={!editable}
                      value={first.fill || defaults.defaultFill}
                      onChange={(fill) => onPatch({ fill })}
                    />
                  </label>
                  <button
                    className="text-button"
                    disabled={!editable}
                    onClick={() =>
                      onPatch({
                        fill:
                          first.fill === "transparent"
                            ? defaults.defaultFill
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
                    List style
                    <select
                      aria-label="List style"
                      value={first.listStyle || ""}
                      disabled={
                        !editable ||
                        (first.text || "").split("\n").length > 1000
                      }
                      onChange={(event) =>
                        onPatch({
                          listStyle:
                            (event.target
                              .value as CanvasElement["listStyle"]) ||
                            undefined,
                          checked:
                            event.target.value === "check"
                              ? first.checked
                              : undefined,
                          ...(event.target.value
                            ? { align: "left", verticalAlign: "top" }
                            : {}),
                        })
                      }
                    >
                      <option value="">Plain text</option>
                      <option value="bullet">Bulleted list</option>
                      <option value="number">Numbered list</option>
                      <option value="check">Checklist</option>
                    </select>
                  </label>
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
          <button
            className="secondary full"
            disabled={!canSaveTemplate || selected.length > 500}
            onClick={onSaveTemplate}
          >
            Save as template
          </button>
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
