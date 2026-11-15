import { useEffect, useRef, useState } from "react";
import type { Point } from "../../model";
import { clone } from "../../model";
import type { PageSession } from "../../session";
import type { Camera } from "../editor/types";
import { ErrorMessage } from "../../ui";
import { renderPage } from "./renderPage";

// This overlay captures a region without changing tools, selection, or document content.
export default function ScreenshotExport({
  session,
  camera,
  close,
}: {
  session: PageSession;
  camera: Camera;
  close: () => void;
}) {
  const host = useRef<HTMLDivElement>(null);
  const pointer = useRef<number | undefined>(undefined);
  const [start, setStart] = useState<Point>();
  const [end, setEnd] = useState<Point>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [status, setStatus] = useState("");
  const clipboard =
    window.isSecureContext &&
    !!navigator.clipboard?.write &&
    typeof ClipboardItem !== "undefined";
  const region =
    start && end
      ? {
          x: Math.min(start.x, end.x),
          y: Math.min(start.y, end.y),
          width: Math.abs(end.x - start.x),
          height: Math.abs(end.y - start.y),
        }
      : undefined;
  useEffect(() => {
    const key = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !busy) {
        event.stopPropagation();
        close();
      }
    };
    document.addEventListener("keydown", key, true);
    return () => document.removeEventListener("keydown", key, true);
  }, [busy, close]);
  function point(event: React.PointerEvent<HTMLDivElement>): Point {
    const box = host.current!.getBoundingClientRect();
    return {
      x: Math.max(0, Math.min(box.width, event.clientX - box.left)),
      y: Math.max(0, Math.min(box.height, event.clientY - box.top)),
    };
  }
  async function image() {
    if (!region || region.width < 2 || region.height < 2)
      throw new Error("Drag to select an area first.");
    await session.savePendingChanges();
    const document = clone(session.doc);
    return renderPage(document, {
      region: {
        x: (region.x - camera.x) / camera.zoom,
        y: (region.y - camera.y) / camera.zoom,
        width: region.width / camera.zoom,
        height: region.height / camera.zoom,
      },
      resolution: Math.min(
        4096,
        Math.ceil(Math.max(region.width, region.height) * devicePixelRatio),
      ),
      background: true,
      grid: true,
      backgroundColor:
        getComputedStyle(window.document.documentElement)
          .getPropertyValue("--bg")
          .trim() || "#ffffff",
    });
  }
  async function output(copy: boolean) {
    setBusy(true);
    setError("");
    setStatus("");
    try {
      if (copy) {
        // Start clipboard.write in the user gesture; Safari accepts a promised PNG.
        await navigator.clipboard.write([
          new ClipboardItem({ "image/png": image() }),
        ]);
        setStatus("Screenshot copied to clipboard.");
      } else {
        const { downloadFile } = await import("./exportFiles");
        downloadFile({ blob: await image(), name: "mindflow-screenshot.png" });
        setStatus("Screenshot downloaded.");
      }
    } catch (error) {
      setError(
        (error as Error).message ||
          "Screenshot export failed. Try downloading a PNG instead.",
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <div
      ref={host}
      className="screenshot-selection"
      role="region"
      aria-label="Screenshot selection"
      onWheel={(event) => event.stopPropagation()}
      onPointerDown={(event) => {
        if (
          busy ||
          event.button !== 0 ||
          pointer.current !== undefined ||
          event.target !== event.currentTarget
        )
          return;
        event.preventDefault();
        pointer.current = event.pointerId;
        event.currentTarget.setPointerCapture(event.pointerId);
        const p = point(event);
        setStart(p);
        setEnd(p);
        setError("");
        setStatus("");
      }}
      onPointerMove={(event) => {
        if (pointer.current === event.pointerId) setEnd(point(event));
      }}
      onPointerUp={(event) => {
        if (pointer.current === event.pointerId) {
          setEnd(point(event));
          pointer.current = undefined;
        }
      }}
      onPointerCancel={() => {
        pointer.current = undefined;
        setStart(undefined);
        setEnd(undefined);
      }}
    >
      {region && (
        <div
          className="screenshot-region"
          style={{
            left: region.x,
            top: region.y,
            width: region.width,
            height: region.height,
          }}
        />
      )}
      <div className="screenshot-actions">
        <p>
          Drag over the canvas to select a screenshot area. Only canvas content
          is included.
        </p>
        <div className="form-actions">
          <button className="secondary" disabled={busy} onClick={close}>
            Cancel
          </button>
          <button
            className="secondary"
            disabled={
              busy ||
              !region ||
              region.width < 2 ||
              region.height < 2 ||
              !clipboard
            }
            data-tooltip={
              clipboard
                ? "Copy screenshot"
                : "Clipboard images require HTTPS and browser support. Download PNG instead."
            }
            onClick={() => void output(true)}
          >
            Copy to clipboard
          </button>
          <button
            className="primary"
            disabled={busy || !region || region.width < 2 || region.height < 2}
            onClick={() => void output(false)}
          >
            Download PNG
          </button>
        </div>
        <ErrorMessage message={error} />
        {status && <p role="status">{status}</p>}
      </div>
    </div>
  );
}
