import { AlignLeft, List, ListOrdered, ListTodo } from "lucide-react";
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import type { CanvasElement } from "../../model";
import { wholeFontSize } from "../../model";
import { textVerticalOffset } from "./textLayout";
import type { Camera } from "./types";
type Props = {
  element: CanvasElement;
  camera: Camera;
  value: string;
  onChange: (value: string) => void;
  onFinish: (cancel?: boolean) => void;
  onListStyle: (style: CanvasElement["listStyle"]) => void;
};
export default function CanvasTextEditor({
  element,
  camera,
  value,
  onChange,
  onFinish,
  onListStyle,
}: Props) {
  const input = useRef<HTMLTextAreaElement>(null);
  useLayoutEffect(() => {
    // Keep focus in the input gesture's task so iPad can open its keyboard.
    input.current?.focus();
    input.current?.select();
  }, [element.id]);
  const [doneBottom, setDoneBottom] = useState(75);
  useEffect(() => {
    const viewport = window.visualViewport;
    if (!viewport) return;
    const position = () => {
      const bottom =
        input.current?.parentElement?.getBoundingClientRect().bottom;
      if (bottom === undefined) return;
      // Safari's keyboard shrinks the visual viewport without resizing the page.
      setDoneBottom(
        Math.max(75, bottom - viewport.height - viewport.offsetTop + 16),
      );
    };
    position();
    viewport.addEventListener("resize", position);
    viewport.addEventListener("scroll", position);
    return () => {
      viewport.removeEventListener("resize", position);
      viewport.removeEventListener("scroll", position);
    };
  }, []);
  const paddingTop = useMemo(
    () => textVerticalOffset(element, value),
    [element, value],
  );
  return (
    <>
      <textarea
        aria-label="Edit canvas text"
        ref={input}
        className="canvas-text-editor"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        onBlur={(event) => {
          if (!(
            event.relatedTarget instanceof Element &&
            event.relatedTarget.closest(".canvas-text-actions")
          ))
            onFinish();
        }}
        onKeyDown={(event) => {
          event.stopPropagation();
          if (event.key === "Escape") {
            event.preventDefault();
            onFinish(true);
          }
          if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
            event.preventDefault();
            onFinish();
          }
        }}
        style={{
          left: element.x * camera.zoom + camera.x,
          top: element.y * camera.zoom + camera.y,
          width: element.width,
          height: element.height,
          fontSize: wholeFontSize(element.fontSize || 24),
          paddingTop,
          fontFamily: "Arial",
          fontWeight: element.bold ? "bold" : "normal",
          fontStyle: element.italic ? "italic" : "normal",
          textAlign: element.align,
          color: element.color,
          transform: `translate(-50%,-50%) rotate(${(element.rotation * 180) / Math.PI}deg) scale(${camera.zoom})`,
          lineHeight: 1.3,
        }}
      />
      <div className="canvas-text-actions" style={{ bottom: doneBottom }}>
        {(
          [
            { style: undefined, label: "Plain text", icon: AlignLeft },
            { style: "bullet", label: "Bulleted list", icon: List },
            { style: "number", label: "Numbered list", icon: ListOrdered },
            { style: "check", label: "Checklist", icon: ListTodo },
          ] as const
        ).map(({ style, label, icon: Icon }) => (
          <button
            key={label}
            className={`icon-button ${element.listStyle === style ? "active" : ""}`}
            aria-label={label}
            aria-pressed={element.listStyle === style}
            data-tooltip={label}
            disabled={!!style && value.split("\n").length > 1000}
            onPointerDown={(event) => event.preventDefault()}
            onClick={() => onListStyle(style)}
          >
            <Icon size={18} />
          </button>
        ))}
        <button
          className="primary"
          aria-label="Done editing text"
          onTouchEnd={(event) => {
            event.preventDefault();
            onFinish();
          }}
          onClick={() => onFinish()}
          data-tooltip="Done editing text"
        >
          Done
        </button>
      </div>
    </>
  );
}
