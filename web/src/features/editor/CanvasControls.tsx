import { Grid2X2, Keyboard, Scan, ZoomIn, ZoomOut } from "lucide-react";
type Props = {
  count: number;
  editable: boolean;
  snap: boolean;
  zoom: number;
  onHelp: () => void;
  onSnap: () => void;
  onZoom: (factor: number) => void;
  onFit: () => void;
};
export default function CanvasControls({
  count,
  editable,
  snap,
  zoom,
  onHelp,
  onSnap,
  onZoom,
  onFit,
}: Props) {
  return (
    <div className="canvas-footer">
      <div className="canvas-info">
        <span data-testid="element-count">
          {count} {count === 1 ? "element" : "elements"}
        </span>
        <span className="info-dot">·</span>
        <span>{editable ? "Cmd / Ctrl + drag to pan" : "Read-only"}</span>
      </div>
      <div className="zoom-controls">
        <button
          className="icon-button"
          aria-label="Keyboard shortcuts"
          onClick={onHelp}
          data-tooltip="Keyboard shortcuts"
        >
          <Keyboard size={16} />
        </button>
        <span className="toolbar-divider" />
        <button
          className={`icon-button ${snap ? "active" : ""}`}
          aria-label="Toggle snap to grid"
          aria-pressed={snap}
          disabled={!editable}
          onClick={onSnap}
          data-tooltip="Toggle snap to grid"
        >
          <Grid2X2 size={16} />
        </button>
        <button
          className="icon-button"
          aria-label="Zoom out"
          onClick={() => onZoom(1 / 1.2)}
          data-tooltip="Zoom out"
        >
          <ZoomOut size={16} />
        </button>
        <button
          className="zoom-value"
          data-tooltip="Reset zoom"
          onClick={() => onZoom(1 / zoom)}
        >
          {Math.round(zoom * 100)}%
        </button>
        <button
          className="icon-button"
          aria-label="Zoom in"
          onClick={() => onZoom(1.2)}
          data-tooltip="Zoom in"
        >
          <ZoomIn size={16} />
        </button>
        <button
          className="icon-button"
          aria-label="Fit content"
          onClick={onFit}
          data-tooltip="Fit content"
        >
          <Scan size={16} />
        </button>
      </div>
    </div>
  );
}
