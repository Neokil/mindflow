import ColorInput from "../../ColorInput";
import { canvasElementDefaults } from "../../model";
import { X } from "lucide-react";
import type { CanvasDocument } from "../../model";
export default function CanvasSettings({
  settings,
  editable,
  onSettings,
  onClose,
}: {
  settings: CanvasDocument["settings"];
  editable: boolean;
  onSettings: (settings: CanvasDocument["settings"]) => void;
  onClose: () => void;
}) {
  const defaults = canvasElementDefaults(settings);
  return (
    <aside className="inspector" aria-label="Canvas settings panel">
      <div className="inspector-head">
        <span>Canvas settings</span>
        <button
          className="icon-button"
          aria-label="Close settings"
          onClick={onClose}
        >
          <X size={15} />
        </button>
      </div>
      <span className="eyebrow">CANVAS</span>
      <label>
        Background
        <select
          value={settings.background}
          disabled={!editable}
          onChange={(e) =>
            onSettings({
              ...settings,
              background: e.target.value as "dots",
            })
          }
        >
          <option value="dots">Dot grid</option>
          <option value="plain">Plain</option>
        </select>
      </label>
      <label className="check-label">
        <input
          type="checkbox"
          checked={!!settings.backgroundColor}
          disabled={!editable}
          onChange={(event) =>
            onSettings({
              ...settings,
              backgroundColor: event.target.checked ? "#ffffff" : undefined,
            })
          }
        />
        Custom background color
      </label>
      {settings.backgroundColor && (
        <label>
          Background color
          <ColorInput
            label="Canvas background color"
            disabled={!editable}
            value={settings.backgroundColor}
            onChange={(backgroundColor) =>
              onSettings({ ...settings, backgroundColor })
            }
          />
        </label>
      )}
      <hr />
      <span className="eyebrow">NEW ELEMENT DEFAULTS</span>
      {(
        [
          ["defaultFill", "Default shape fill"],
          ["defaultBorder", "Default shape border"],
          ["defaultTextColor", "Default text color"],
        ] as const
      ).map(([key, label]) => (
        <label key={key}>
          {label}
          <ColorInput
            label={label}
            value={defaults[key]}
            disabled={!editable}
            onChange={(color) => onSettings({ ...settings, [key]: color })}
          />
        </label>
      ))}
      <hr />
      <label>
        Grid spacing
        <input
          aria-label="Grid spacing"
          type="number"
          min="1"
          max="1000"
          disabled={!editable}
          value={settings.gridSize}
          onChange={(e) =>
            onSettings({
              ...settings,
              gridSize: Math.max(1, Math.min(1000, +e.target.value)),
            })
          }
        />
      </label>
      <label className="check-label">
        <input
          type="checkbox"
          checked={settings.snap}
          disabled={!editable}
          onChange={(e) => onSettings({ ...settings, snap: e.target.checked })}
        />
        Snap when moving & resizing
      </label>
    </aside>
  );
}
