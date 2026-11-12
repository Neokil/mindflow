import { useRef, useState } from "react";
import type { PointerEvent } from "react";
import type { CanvasElement, ImageCrop, Point } from "../../model";
import { localImages } from "../../session";
import { Modal, ErrorMessage } from "../../ui";
import { croppedElement, fullImageCrop } from "./imageCrop";

const clamp = (value: number, min = 0, max = 1) =>
  Math.max(min, Math.min(max, value));
type Gesture = { start: Point; crop: ImageCrop; mode: string };
export default function ImageCropDialog({
  element,
  onApply,
  onClose,
}: {
  element: CanvasElement;
  onApply: (patch: Partial<CanvasElement>) => void;
  onClose: () => void;
}) {
  const [crop, setCrop] = useState(element.crop || fullImageCrop);
  const [ratio, setRatio] = useState(1);
  const [loaded, setLoaded] = useState(false);
  const [failed, setFailed] = useState(false);
  const gesture = useRef<Gesture | undefined>(undefined);
  const source =
    localImages.get(element.assetId!)?.url ||
    `/api/v1/assets/${element.assetId}`;
  function point(event: PointerEvent<HTMLDivElement>) {
    const bounds = event.currentTarget.getBoundingClientRect();
    return {
      x: clamp((event.clientX - bounds.left) / bounds.width),
      y: clamp((event.clientY - bounds.top) / bounds.height),
    };
  }
  function move(event: PointerEvent<HTMLDivElement>) {
    const current = gesture.current;
    if (!current) return;
    const p = point(event),
      initial = current.crop;
    if (current.mode === "move") {
      setCrop({
        ...initial,
        x: clamp(initial.x + p.x - current.start.x, 0, 1 - initial.width),
        y: clamp(initial.y + p.y - current.start.y, 0, 1 - initial.height),
      });
      return;
    }
    let left = initial.x,
      top = initial.y,
      right = left + initial.width,
      bottom = top + initial.height;
    if (current.mode === "draw") {
      left = Math.min(current.start.x, p.x);
      top = Math.min(current.start.y, p.y);
      right = Math.max(current.start.x, p.x);
      bottom = Math.max(current.start.y, p.y);
    } else {
      if (current.mode.includes("w")) left = Math.min(p.x, right - 0.01);
      if (current.mode.includes("e")) right = Math.max(p.x, left + 0.01);
      if (current.mode.includes("n")) top = Math.min(p.y, bottom - 0.01);
      if (current.mode.includes("s")) bottom = Math.max(p.y, top + 0.01);
    }
    if (right - left >= 0.01 && bottom - top >= 0.01)
      setCrop({ x: left, y: top, width: right - left, height: bottom - top });
  }
  function setEdge(edge: string, value: number) {
    if (!Number.isFinite(value)) return;
    const left = crop.x,
      top = crop.y,
      right = crop.x + crop.width,
      bottom = crop.y + crop.height;
    const fraction = value / 100;
    if (edge === "left") {
      const x = clamp(fraction, 0, right - 0.01);
      setCrop({ ...crop, x, width: right - x });
    }
    if (edge === "top") {
      const y = clamp(fraction, 0, bottom - 0.01);
      setCrop({ ...crop, y, height: bottom - y });
    }
    if (edge === "right")
      setCrop({ ...crop, width: clamp(1 - fraction, left + 0.01, 1) - left });
    if (edge === "bottom")
      setCrop({ ...crop, height: clamp(1 - fraction, top + 0.01, 1) - top });
  }
  return (
    <Modal title="Crop image" wide close={onClose}>
      <p className="muted">
        Drag to select an area. Move the selection or drag its corners to adjust
        it.
      </p>
      <div
        className="crop-preview"
        data-testid="crop-preview"
        style={{
          width: `min(560px, 70vw, ${55 * ratio}vh)`,
          aspectRatio: ratio,
        }}
        onPointerDown={(event) => {
          if (!loaded || event.button !== 0) return;
          event.preventDefault();
          event.currentTarget.setPointerCapture(event.pointerId);
          const target = event.target as HTMLElement;
          gesture.current = {
            start: point(event),
            crop,
            mode:
              target.dataset.corner ||
              (target.closest(".crop-selection") &&
              (crop.width < 1 || crop.height < 1)
                ? "move"
                : "draw"),
          };
        }}
        onPointerMove={move}
        onPointerUp={() => {
          gesture.current = undefined;
        }}
        onPointerCancel={() => {
          gesture.current = undefined;
        }}
      >
        <img
          src={source}
          alt="Original image for cropping"
          draggable={false}
          onLoad={(event) => {
            setRatio(
              event.currentTarget.naturalWidth /
                event.currentTarget.naturalHeight,
            );
            setLoaded(true);
          }}
          onError={() => setFailed(true)}
        />
        {loaded && (
          <div
            className="crop-selection"
            data-testid="crop-selection"
            style={{
              left: `${crop.x * 100}%`,
              top: `${crop.y * 100}%`,
              width: `${crop.width * 100}%`,
              height: `${crop.height * 100}%`,
            }}
          >
            {["nw", "ne", "sw", "se"].map((corner) => (
              <span
                key={corner}
                className={`crop-corner ${corner}`}
                data-corner={corner}
              />
            ))}
          </div>
        )}
      </div>
      <ErrorMessage
        message={failed ? "Image could not be loaded." : undefined}
      />
      <div className="crop-fields">
        {Object.entries({
          left: crop.x,
          top: crop.y,
          right: 1 - crop.x - crop.width,
          bottom: 1 - crop.y - crop.height,
        }).map(([edge, value]) => (
          <label key={edge}>
            {edge[0].toUpperCase() + edge.slice(1)} (%)
            <input
              aria-label={`Crop ${edge} (%)`}
              type="number"
              min="0"
              max="99"
              step="0.1"
              value={Math.round(value * 1000) / 10}
              onChange={(event) => setEdge(edge, event.target.valueAsNumber)}
            />
          </label>
        ))}
      </div>
      <div className="crop-actions">
        <button className="secondary" onClick={() => setCrop(fullImageCrop)}>
          Reset crop
        </button>
        <button className="secondary" onClick={onClose}>
          Cancel
        </button>
        <button
          className="primary"
          disabled={!loaded || failed}
          onClick={() => onApply(croppedElement(element, crop))}
        >
          Apply crop
        </button>
      </div>
    </Modal>
  );
}
