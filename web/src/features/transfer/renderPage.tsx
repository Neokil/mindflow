import { createRoot } from "react-dom/client";
import { flushSync } from "react-dom";
import { Group, Image as KImage, Layer, Rect, Stage } from "react-konva";
import type Konva from "konva";
import ElementContent from "../editor/CanvasElement";
import type { CanvasDocument } from "../../model";
import { getAssetMetadata, getDecodedImage } from "../../media";
import { exportSize } from "./exportGeometry";
export type RenderOptions = {
  resolution: number;
  background: boolean;
  grid: boolean;
  backgroundColor: string;
};
export async function renderPage(
  doc: CanvasDocument,
  options: RenderOptions,
): Promise<Blob> {
  const { bounds, scale, width, height } = exportSize(doc, options.resolution);
  const images = new Map<string, HTMLImageElement>();
  let decodedPixels = 0;
  // Prefer an existing display variant only when it satisfies the export resolution.
  for (const e of doc.elements) {
    if ((e.type === "image" || e.type === "pdf") && !images.has(e.assetId!)) {
      try {
        const similar = doc.elements.filter(
          (item) => item.assetId === e.assetId,
        );
        let suffix = "?bucket=1024";
        if (e.type === "image") {
          const meta = await getAssetMetadata(e.assetId!);
          const need = Math.max(
            ...similar.map(
              (item) =>
                Math.max(
                  item.width / (item.crop?.width || 1) / meta.width,
                  item.height / (item.crop?.height || 1) / meta.height,
                ) *
                scale *
                Math.max(meta.width, meta.height),
            ),
          );
          const bucket = meta.variants.find((value) => value >= need);
          suffix = bucket ? `?bucket=${bucket}` : "";
        }
        const image = await getDecodedImage(
          `/api/v1/assets/${e.assetId}${suffix}`,
        );
        decodedPixels += image.naturalWidth * image.naturalHeight;
        if (decodedPixels > 64 * 1024 * 1024) throw new Error("media budget");
        images.set(e.assetId!, image);
      } catch {
        throw new Error(
          "Media could not be loaded or exceeds the export memory limit. Check your connection or export fewer elements per page.",
        );
      }
    }
  }
  await document.fonts.ready;
  const host = document.createElement("div");
  host.style.cssText = "position:fixed;left:-20000px;top:0;pointer-events:none";
  document.body.append(host);
  const root = createRoot(host);
  let stage: Konva.Stage | null = null;
  const grid = document.createElement("canvas");
  grid.width = options.grid ? width : 1;
  grid.height = options.grid ? height : 1;
  const spacing = doc.settings.gridSize;
  if (
    options.grid &&
    doc.settings.background === "dots" &&
    spacing * scale >= 4
  ) {
    const context = grid.getContext("2d")!;
    context.fillStyle = "#8d968d";
    for (
      let x = Math.ceil(bounds.x / spacing) * spacing;
      x <= bounds.x + bounds.width;
      x += spacing
    )
      for (
        let y = Math.ceil(bounds.y / spacing) * spacing;
        y <= bounds.y + bounds.height;
        y += spacing
      )
        context.fillRect((x - bounds.x) * scale, (y - bounds.y) * scale, 1, 1);
  }
  try {
    flushSync(() =>
      root.render(
        <Stage
          ref={(node) => {
            stage = node;
          }}
          width={width}
          height={height}
          listening={false}
        >
          <Layer>
            {options.background && (
              <Rect
                width={width}
                height={height}
                fill={doc.settings.backgroundColor || options.backgroundColor}
              />
            )}
            {options.grid && (
              <KImage image={grid} width={width} height={height} />
            )}
            <Group
              x={-bounds.x * scale}
              y={-bounds.y * scale}
              scaleX={scale}
              scaleY={scale}
            >
              {doc.elements.map((e) => (
                <Group
                  key={e.id}
                  x={e.x}
                  y={e.y}
                  rotation={(e.rotation * 180) / Math.PI}
                >
                  <ElementContent
                    e={e}
                    zoom={scale}
                    refresh={0}
                    loadedImage={images.get(e.assetId!)}
                    exporting
                  />
                </Group>
              ))}
            </Group>
          </Layer>
        </Stage>,
      ),
    );
    const canvas = (stage as Konva.Stage | null)?.toCanvas({ pixelRatio: 1 });
    if (!canvas) throw new Error("Unable to render the export.");
    return await new Promise<Blob>((resolve, reject) =>
      canvas.toBlob(
        (blob) =>
          blob
            ? resolve(blob)
            : reject(new Error("Unable to create the image.")),
        "image/png",
      ),
    );
  } finally {
    root.unmount();
    host.remove();
  }
}
