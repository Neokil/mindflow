import { Maximize, Minimize } from "lucide-react";
import { useEffect, useId, useRef, useState } from "react";
import type { CanvasFullscreen } from "../../useCanvasFullscreen";

export default function FullscreenButton({
  fullscreen,
}: {
  fullscreen: CanvasFullscreen;
}) {
  const disabled = !fullscreen.available && !fullscreen.active;
  const [visible, setVisible] = useState(false);
  const host = useRef<HTMLSpanElement>(null);
  const hint = useId();
  const showingHint = disabled && visible;
  useEffect(() => {
    if (!showingHint) return;
    const outside = (event: PointerEvent) => {
      if (event.target instanceof Node && !host.current?.contains(event.target))
        setVisible(false);
    };
    const escape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setVisible(false);
    };
    document.addEventListener("pointerdown", outside, true);
    document.addEventListener("keydown", escape);
    return () => {
      document.removeEventListener("pointerdown", outside, true);
      document.removeEventListener("keydown", escape);
    };
  }, [showingHint]);
  const label = fullscreen.active ? "Exit fullscreen" : "Enter fullscreen";
  return (
    <span
      ref={host}
      className="fullscreen-control"
      tabIndex={disabled ? 0 : undefined}
      aria-label={disabled ? "Fullscreen availability" : undefined}
      aria-describedby={showingHint ? hint : undefined}
      onPointerEnter={(event) => {
        if (disabled && event.pointerType === "mouse") setVisible(true);
      }}
      onPointerLeave={(event) => {
        if (event.pointerType === "mouse") setVisible(false);
      }}
      onPointerDown={(event) => {
        if (disabled && event.pointerType !== "mouse") {
          event.preventDefault();
          setVisible((value) => !value);
        }
      }}
      onFocus={() => disabled && setVisible(true)}
      onBlur={() => setVisible(false)}
    >
      <button
        className="fullscreen-toggle icon-button"
        aria-label={label}
        title={disabled ? undefined : label}
        aria-pressed={fullscreen.active}
        aria-describedby={showingHint ? hint : undefined}
        disabled={disabled}
        onClick={() => void fullscreen.toggle()}
      >
        {fullscreen.active ? <Minimize size={19} /> : <Maximize size={19} />}
      </button>
      {showingHint && (
        <span id={hint} className="fullscreen-hint" role="tooltip">
          On iPad, fullscreen mode is only supported in Chrome and Safari.
        </span>
      )}
    </span>
  );
}
