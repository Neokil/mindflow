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
};
export default function CanvasTextEditor({
  element,
  camera,
  value,
  onChange,
  onFinish,
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
        onBlur={() => onFinish()}
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
      <button
        className="primary canvas-text-done"
        aria-label="Done editing text"
        style={{ bottom: doneBottom }}
        onPointerDown={(event) => event.preventDefault()}
        onClick={() => onFinish()}
      >
        Done
      </button>
    </>
  );
}
