import ColorInput from "../../ColorInput";
import type { Stabilization } from "./InkStroke";
export default function PenOptions({
  highlighter,
  color,
  width,
  pencilOnly,
  pressure,
  stabilization,
  onColor,
  onWidth,
  onPencilOnly,
  onPressure,
  onStabilization,
}: {
  highlighter: boolean;
  color: string;
  width: number;
  pencilOnly: boolean;
  pressure: boolean;
  stabilization: Stabilization;
  onColor: (value: string) => void;
  onWidth: (value: number) => void;
  onPencilOnly: (value: boolean) => void;
  onPressure: (value: boolean) => void;
  onStabilization: (value: Stabilization) => void;
}) {
  return (
    <div className="pen-options" aria-label="Ink settings">
      <label>
        Color
        <ColorInput label="Pen color" value={color} onChange={onColor} />
      </label>
      <label className="check-label">
        <input
          type="checkbox"
          aria-label="Pencil only"
          checked={pencilOnly}
          onChange={(e) => onPencilOnly(e.target.checked)}
        />
        Pencil only
      </label>
      {!highlighter && (
        <label className="check-label">
          <input
            type="checkbox"
            aria-label="Pressure-sensitive ink"
            checked={pressure}
            onChange={(e) => onPressure(e.target.checked)}
          />
          Pressure-sensitive ink
        </label>
      )}
      <label>
        Width
        <input
          aria-label="Pen thickness"
          type="range"
          min="1"
          max="40"
          value={width}
          onChange={(e) => onWidth(Number(e.target.value))}
        />
        <span>
          {width}px{!highlighter && pressure ? " maximum with Pencil" : ""}
        </span>
      </label>
      <label>
        Stabilization
        <select
          aria-label="Stroke stabilization"
          value={stabilization}
          onChange={(e) => onStabilization(e.target.value as Stabilization)}
        >
          <option value="off">Off</option>
          <option value="light">Light</option>
          <option value="strong">Strong</option>
        </select>
      </label>
      <small>
        Smooths small hand movements. Hold still at the end to straighten.
      </small>
    </div>
  );
}
