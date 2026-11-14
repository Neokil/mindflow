import { ChevronDown, Circle, Square, Star } from "lucide-react";
import { createPortal } from "react-dom";
import { useLayoutEffect, useEffect, useId, useRef, useState } from "react";
import type { Tool } from "./types";
const shapes = [
  { id: "rectangle", label: "Rectangle", icon: Square },
  { id: "circle", label: "Circle", icon: Circle },
  { id: "star", label: "Star", icon: Star },
] as const;
export default function ShapeTool({
  tool,
  placing,
  disabled,
  onTool,
}: {
  tool: Tool;
  placing: boolean;
  disabled: boolean;
  onTool: (tool: Tool) => void;
}) {
  const [open, setOpen] = useState(false),
    [last, setLast] = useState<Tool>("rectangle");
  const host = useRef<HTMLDivElement>(null),
    trigger = useRef<HTMLButtonElement>(null),
    menu = useRef<HTMLDivElement>(null),
    id = useId();
  const active = shapes.find((shape) => shape.id === tool),
    Icon = (active || shapes.find((shape) => shape.id === last)!).icon;
  useEffect(() => {
    if (!open) return;
    (
      menu.current?.querySelector<HTMLButtonElement>('[aria-checked="true"]') ||
      menu.current?.querySelector<HTMLButtonElement>("button")
    )?.focus();
    const outside = (event: PointerEvent) => {
      if (
        event.target instanceof Node &&
        !host.current?.contains(event.target) &&
        !menu.current?.contains(event.target)
      )
        setOpen(false);
    };
    document.addEventListener("pointerdown", outside, true);
    return () => document.removeEventListener("pointerdown", outside, true);
  }, [open]);
  useLayoutEffect(() => {
    if (!open || !menu.current || !trigger.current) return;
    const node = menu.current;
    node.showPopover?.();
    const box = trigger.current.getBoundingClientRect(),
      height = node.getBoundingClientRect().height;
    node.style.left = `${Math.max(8, Math.min(innerWidth - 173, box.left))}px`;
    node.style.top = `${box.bottom + height + 8 < innerHeight ? box.bottom + 8 : Math.max(8, box.top - height - 8)}px`;
    return () => node.hidePopover?.();
  }, [open]);
  useEffect(() => {
    if (disabled) setOpen(false);
  }, [disabled]);
  return (
    <div className="shape-tool" ref={host}>
      <button
        ref={trigger}
        className={`tool ${active && !placing ? "active" : ""}`}
        aria-label="Shapes"
        title="Shapes: rectangle, circle, or star"
        aria-pressed={!!active && !placing}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={id}
        disabled={disabled}
        onClick={() => setOpen(!open)}
        onKeyDown={(event) => {
          if (event.key === "ArrowDown" || event.key === "ArrowUp") {
            event.preventDefault();
            setOpen(true);
          }
        }}
      >
        <Icon size={18} strokeWidth={1.7} />
        <ChevronDown size={9} className="shape-chevron" />
      </button>
      {open &&
        createPortal(
          <div
            id={id}
            popover={
              "showPopover" in HTMLElement.prototype ? "manual" : undefined
            }
            ref={menu}
            className="dropdown shape-menu"
            role="menu"
            aria-label="Shape tools"
            onKeyDown={(event) => {
              const choices = [
                ...menu.current!.querySelectorAll<HTMLButtonElement>("button"),
              ];
              const index = choices.indexOf(
                document.activeElement as HTMLButtonElement,
              );
              if (event.key === "Escape") {
                event.preventDefault();
                event.stopPropagation();
                setOpen(false);
                trigger.current?.focus();
              } else if (
                ["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)
              ) {
                event.preventDefault();
                choices[
                  event.key === "Home"
                    ? 0
                    : event.key === "End"
                      ? choices.length - 1
                      : (index +
                          (event.key === "ArrowDown"
                            ? 1
                            : choices.length - 1)) %
                        choices.length
                ]?.focus();
              }
            }}
          >
            {shapes.map(({ id, label, icon: Shape }) => (
              <button
                key={id}
                role="menuitemradio"
                aria-checked={tool === id && !placing}
                onClick={() => {
                  setLast(id);
                  onTool(id);
                  setOpen(false);
                  trigger.current?.focus();
                }}
              >
                <Shape size={17} />
                {label}
              </button>
            ))}
          </div>,
          document.fullscreenElement || document.body,
        )}
    </div>
  );
}
