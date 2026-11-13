import CanvasList from "./CanvasList";
import { memo, useEffect, useState } from "react";
import { Ellipse, Image as KImage, Line, Rect, Text } from "react-konva";
import { api } from "../../api";
import {
  getAssetMetadata,
  getDecodedImage,
  invalidateMedia,
} from "../../media";
import type { CanvasElement } from "../../model";
import { wholeFontSize } from "../../model";
import { localImages } from "../../session";
function ImageContent({
  element: e,
  zoom,
  refresh,
  loadedImage,
}: {
  element: CanvasElement;
  zoom: number;
  refresh: number;
  loadedImage?: HTMLImageElement;
}) {
  const [image, setImage] = useState<HTMLImageElement>();
  const [failed, setFailed] = useState(false);
  const assetID = e.assetId!;
  const renderWidth =
    Math.ceil(
      ((e.width / (e.crop?.width || 1)) * zoom * (devicePixelRatio || 1)) / 64,
    ) * 64;
  const renderHeight =
    Math.ceil(
      ((e.height / (e.crop?.height || 1)) * zoom * (devicePixelRatio || 1)) /
        64,
    ) * 64;
  useEffect(() => {
    if (loadedImage) return;
    let canceled = false;
    let retry: ReturnType<typeof setTimeout>;
    const local = localImages.get(assetID);
    const load = (url: string) => {
      getDecodedImage(url)
        .then((im) => {
          if (!canceled) {
            setImage(im);
            setFailed(false);
          }
        })
        .catch(() => {
          if (!canceled) setFailed(true);
        });
    };
    if (e.type === "pdf") {
      load(`/api/v1/assets/${assetID}?bucket=1024`);
      return () => {
        canceled = true;
      };
    }
    if (assetID.startsWith("local:")) {
      if (local) load(local.url);
      return () => {
        canceled = true;
      };
    }
    if (local) load(local.url);
    const run = async () => {
      try {
        const meta = await getAssetMetadata(assetID);
        if (canceled) return;
        const need =
          Math.max(renderWidth / meta.width, renderHeight / meta.height) *
          Math.max(meta.width, meta.height);
        const bucket = meta.variants.find((b) => b >= need);
        load(`/api/v1/assets/${assetID}${bucket ? `?bucket=${bucket}` : ""}`);
        if (!bucket && need < Math.max(meta.width, meta.height) * 0.8) {
          await api(`/assets/${assetID}/variant`, "POST", {
            width: renderWidth,
            height: renderHeight,
          });
          retry = setTimeout(() => {
            invalidateMedia(assetID);
            void run();
          }, 2500);
        }
      } catch {
        if (!local) setFailed(true);
      }
    };
    const delay = setTimeout(run, 300);
    return () => {
      canceled = true;
      clearTimeout(delay);
      clearTimeout(retry);
    };
    // Re-evaluate image resolution only when the actual display requirement changes.
  }, [assetID, renderWidth, renderHeight, refresh, e.type, loadedImage]);
  const displayed = loadedImage || image;
  return displayed ? (
    <KImage
      image={displayed}
      crop={
        e.crop
          ? {
              x: e.crop.x * displayed.naturalWidth,
              y: e.crop.y * displayed.naturalHeight,
              width: e.crop.width * displayed.naturalWidth,
              height: e.crop.height * displayed.naturalHeight,
            }
          : undefined
      }
      x={-e.width / 2}
      y={-e.height / 2}
      width={e.width}
      height={e.height}
    />
  ) : (
    <>
      <Rect
        x={-e.width / 2}
        y={-e.height / 2}
        width={e.width}
        height={e.height}
        fill="#ecebf2"
        cornerRadius={3}
      />
      <Text
        text={failed ? "Image unavailable" : "Loading image…"}
        x={-e.width / 2 + 8}
        y={-8}
        width={e.width - 16}
        fontSize={12}
        align="center"
        fill="#88848e"
      />
    </>
  );
}
const ElementContent = memo(function ElementContent({
  e,
  zoom,
  refresh,
  onToggleCheck,
  loadedImage,
  exporting = false,
}: {
  e: CanvasElement;
  zoom: number;
  refresh: number;
  loadedImage?: HTMLImageElement;
  exporting?: boolean;
  onToggleCheck?: (index: number) => void;
}) {
  if (e.type === "image" || e.type === "pdf")
    return (
      <>
        <ImageContent
          element={e}
          zoom={zoom}
          refresh={refresh}
          loadedImage={loadedImage}
        />
        {e.type === "pdf" && !exporting && (
          <Text
            text="PDF · Open viewer in properties"
            x={-e.width / 2}
            y={e.height / 2 - 24}
            width={e.width}
            height={24}
            fontSize={12}
            fill="#1b5e3b"
            align="center"
            listening={false}
          />
        )}
      </>
    );
  if (e.type === "text" && e.listStyle)
    return <CanvasList element={e} onToggle={onToggleCheck} />;
  if (e.type === "text")
    return (
      <Text
        x={-e.width / 2}
        y={-e.height / 2}
        width={e.width}
        height={e.height}
        text={e.text || ""}
        fontSize={wholeFontSize(e.fontSize || 24)}
        fontFamily="Arial"
        fontStyle={
          `${e.bold ? "bold" : ""} ${e.italic ? "italic" : ""}`.trim() ||
          "normal"
        }
        align={e.align || "left"}
        verticalAlign={e.verticalAlign || "top"}
        fill={e.color || "#28282d"}
        lineHeight={1.3}
        wrap="word"
      />
    );
  if (e.type === "stroke")
    return (
      <Line
        points={(e.points || []).map(
          (p, i) => (p - 0.5) * (i % 2 ? e.height : e.width),
        )}
        stroke={e.color || "#28282d"}
        opacity={e.highlight ? 0.35 : 1}
        globalCompositeOperation={e.highlight ? "multiply" : "source-over"}
        strokeWidth={e.penWidth || 3}
        lineCap="round"
        lineJoin="round"
        hitStrokeWidth={Math.max(10 / zoom, e.penWidth || 3)}
      />
    );
  const props = {
    fill: e.fill === "transparent" ? undefined : e.fill,
    stroke: e.border,
    strokeWidth: e.borderWidth || 0,
  };
  if (e.shape === "circle")
    return <Ellipse radiusX={e.width / 2} radiusY={e.height / 2} {...props} />;
  if (e.shape === "star")
    return (
      <Line
        closed
        points={Array.from({ length: 10 }, (_, i) => {
          const angle = (i * Math.PI) / 5 - Math.PI / 2,
            factor = i % 2 ? 0.45 : 1;
          return [
            ((Math.cos(angle) * e.width) / 2) * factor,
            ((Math.sin(angle) * e.height) / 2) * factor,
          ];
        }).flat()}
        {...props}
      />
    );
  return (
    <Rect
      x={-e.width / 2}
      y={-e.height / 2}
      width={e.width}
      height={e.height}
      {...props}
    />
  );
});

export default ElementContent;
