import { Check } from "lucide-react";
import { Modal } from "../../ui";
export default function EditorHelp({ close }: { close: () => void }) {
  return (
    <Modal title="A few helpful shortcuts" close={close}>
      <p className="muted">
        On tablets, drag with one finger or Pencil to use the selected tool. Use
        two fingers to pan and pinch to zoom. Pencil only mode lets a finger pan
        while Pencil draws.
      </p>
      <div className="shortcut-list">
        {[
          ["Pan", "Pan tool (H), Cmd/Ctrl + drag, or middle button"],
          ["Zoom around cursor", "Scroll; two-finger pinch on tablets"],
          ["Create an element", "Drag; Shift for 1:1"],
          ["Select enclosed elements", "Drag a freehand lasso"],
          ["Add to selection", "Shift + click / lasso"],
          ["Undo / redo", "Cmd / Ctrl + Z / Shift + Z"],
          ["Copy / paste / duplicate", "Cmd / Ctrl + C / V / D"],
          ["Group / ungroup", "Cmd / Ctrl + G / Shift + G"],
          ["Remove selection", "Delete / Backspace"],
          ["Finish text", "Done or Cmd / Ctrl + Enter"],
          ["Cancel", "Escape"],
        ].map(([label, key]) => (
          <div key={label}>
            <span>{label}</span>
            <kbd>{key}</kbd>
          </div>
        ))}
      </div>
      <button className="primary full" onClick={close}>
        <Check size={16} />
        Got it
      </button>
    </Modal>
  );
}
