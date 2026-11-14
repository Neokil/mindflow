import { useEffect, useId, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
type Hint = { anchor: HTMLElement; text: string };
// Controls opt in explicitly with data-tooltip; labels remain their accessible names.
export default function Tooltips() {
  const [hint, setHint] = useState<Hint>();
  const bubble = useRef<HTMLDivElement>(null);
  const id = useId();
  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | undefined,
      anchor: HTMLElement | undefined;
    let mode: "hover" | "focus" | "hold" | undefined;
    let held: HTMLElement | undefined,
      start: { x: number; y: number } | undefined;
    const clear = () => {
      clearTimeout(timer);
      timer = undefined;
    };
    const hide = () => {
      clear();
      anchor = undefined;
      mode = undefined;
      setHint(undefined);
    };
    const control = (target: EventTarget | null) =>
      target instanceof Element
        ? target.closest<HTMLElement>("[data-tooltip]")
        : null;
    const show = (target: HTMLElement) => {
      const text = target.dataset.tooltip;
      if (text && target.isConnected) setHint({ anchor: target, text });
    };
    const over = (event: PointerEvent) => {
      if (event.pointerType !== "mouse" || held) return;
      const next = control(event.target);
      if (next === anchor) return;
      if (!next) {
        if (mode === "hover") hide();
        return;
      }
      hide();
      mode = "hover";
      anchor = next;
      timer = setTimeout(() => show(next), 350);
    };
    const out = (event: PointerEvent) => {
      if (
        event.pointerType !== "mouse" ||
        mode !== "hover" ||
        control(event.target) !== anchor
      )
        return;
      if (
        anchor &&
        event.relatedTarget instanceof Node &&
        anchor.contains(event.relatedTarget)
      )
        return;
      hide();
    };
    const focus = (event: FocusEvent) => {
      hide();
      const next = control(event.target);
      if (next) {
        anchor = next;
        mode = "focus";
        show(next);
      }
    };
    const blur = (event: FocusEvent) => {
      if (control(event.target) === anchor) hide();
    };
    const down = (event: PointerEvent) => {
      hide();
      held = undefined;
      start = undefined;
      const next = control(event.target);
      if (next && event.pointerType !== "mouse") {
        anchor = next;
        start = { x: event.clientX, y: event.clientY };
        timer = setTimeout(() => {
          held = next;
          mode = "hold";
          show(next);
        }, 550);
      }
    };
    const move = (event: PointerEvent) => {
      if (
        start &&
        Math.hypot(event.clientX - start.x, event.clientY - start.y) > 10
      ) {
        hide();
        start = undefined;
      }
    };
    const up = () => {
      clear();
      start = undefined;
    };
    const click = (event: MouseEvent) => {
      if (held && control(event.target) === held) {
        event.preventDefault();
        event.stopPropagation();
        held = undefined;
        return;
      }
      hide();
    };
    const touchEnd = (event: TouchEvent) => {
      if (held) {
        event.preventDefault();
        event.stopPropagation();
      }
    };
    const context = (event: MouseEvent) => {
      if (held && control(event.target) === held) event.preventDefault();
    };
    const key = (event: KeyboardEvent) => {
      if (event.key === "Escape") hide();
    };
    document.addEventListener("pointerover", over, true);
    document.addEventListener("pointerout", out, true);
    document.addEventListener("pointerdown", down, true);
    document.addEventListener("pointermove", move, true);
    document.addEventListener("pointerup", up, true);
    document.addEventListener("pointercancel", hide, true);
    document.addEventListener("focusin", focus, true);
    document.addEventListener("focusout", blur, true);
    document.addEventListener("click", click, true);
    document.addEventListener("contextmenu", context, true);
    document.addEventListener("touchend", touchEnd, {
      capture: true,
      passive: false,
    });
    document.addEventListener("keydown", key, true);
    window.addEventListener("resize", hide);
    return () => {
      clear();
      document.removeEventListener("pointerover", over, true);
      document.removeEventListener("pointerout", out, true);
      document.removeEventListener("pointerdown", down, true);
      document.removeEventListener("pointermove", move, true);
      document.removeEventListener("pointerup", up, true);
      document.removeEventListener("pointercancel", hide, true);
      document.removeEventListener("focusin", focus, true);
      document.removeEventListener("focusout", blur, true);
      document.removeEventListener("click", click, true);
      document.removeEventListener("contextmenu", context, true);
      document.removeEventListener("touchend", touchEnd, true);
      document.removeEventListener("keydown", key, true);
      window.removeEventListener("resize", hide);
    };
  }, []);
  useLayoutEffect(() => {
    const node = bubble.current;
    if (!hint || !node) return;
    node.showPopover?.();
    const previous = hint.anchor.getAttribute("aria-describedby");
    const description = [previous, id].filter(Boolean).join(" ");
    hint.anchor.setAttribute("aria-describedby", description);
    const position = () => {
      const anchor = hint.anchor.getBoundingClientRect(),
        box = node.getBoundingClientRect();
      const x = Math.max(
        8,
        Math.min(
          innerWidth - box.width - 8,
          anchor.left + (anchor.width - box.width) / 2,
        ),
      );
      const below = anchor.bottom + 8,
        y =
          below + box.height < innerHeight - 8
            ? below
            : Math.max(8, anchor.top - box.height - 8);
      node.style.left = `${x}px`;
      node.style.top = `${y}px`;
    };
    position();
    window.addEventListener("scroll", position, true);
    return () => {
      window.removeEventListener("scroll", position, true);
      node.hidePopover?.();
      if (hint.anchor.getAttribute("aria-describedby") === description) {
        if (previous) hint.anchor.setAttribute("aria-describedby", previous);
        else hint.anchor.removeAttribute("aria-describedby");
      }
    };
  }, [hint, id]);
  if (!hint) return null;
  const host =
    hint.anchor.closest("dialog[open]") ||
    document.fullscreenElement ||
    document.body;
  return createPortal(
    <div
      ref={bubble}
      id={id}
      popover={"showPopover" in HTMLElement.prototype ? "manual" : undefined}
      role="tooltip"
      className="app-tooltip"
    >
      {hint.text}
    </div>,
    host,
  );
}
