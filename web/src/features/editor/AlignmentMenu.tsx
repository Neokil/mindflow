import { AlignHorizontalJustifyCenter } from "lucide-react";
import { useState } from "react";
import type { Alignment } from "./alignment";
const choices: [Alignment, string][] = [
  ["left", "Align left"],
  ["center", "Align horizontal centers"],
  ["right", "Align right"],
  ["top", "Align top"],
  ["middle", "Align vertical centers"],
  ["bottom", "Align bottom"],
];
export function AlignmentButtons({
  disabled,
  onAlign,
}: {
  disabled: boolean;
  onAlign: (alignment: Alignment) => void;
}) {
  return (
    <div className="alignment-buttons">
      {choices.map(([key, label]) => (
        <button
          type="button"
          className="secondary"
          key={key}
          disabled={disabled}
          onClick={() => onAlign(key)}
        >
          {label}
        </button>
      ))}
    </div>
  );
}
export default function AlignmentMenu({
  disabled,
  onAlign,
}: {
  disabled: boolean;
  onAlign: (alignment: Alignment) => void;
}) {
  const [open, setOpen] = useState(false);
  return (
    <div className="menu-anchor alignment-menu">
      <button
        className="icon-button"
        aria-label="Align selection"
        data-tooltip="Align selection"
        aria-expanded={open}
        disabled={disabled}
        onClick={() => setOpen(!open)}
      >
        <AlignHorizontalJustifyCenter size={16} />
      </button>
      {open && (
        <>
          <button
            className="menu-dismiss"
            aria-label="Close alignment menu"
            onClick={() => setOpen(false)}
          />
          <div className="dropdown">
            {choices.map(([key, label]) => (
              <button
                key={key}
                onClick={() => {
                  onAlign(key);
                  setOpen(false);
                }}
              >
                {label}
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
