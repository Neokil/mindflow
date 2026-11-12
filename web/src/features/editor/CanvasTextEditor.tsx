import { useEffect, useMemo, useRef } from "react";
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
  useEffect(() => {
    const timer = setTimeout(() => {
      input.current?.focus();
      input.current?.select();
    }, 0);
    return () => clearTimeout(timer);
  }, [element.id]);
  const paddingTop = useMemo(
    () => textVerticalOffset(element, value),
    [element, value],
  );
  return (
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
  );
}
