import { PDFDocument, rgb } from "pdf-lib";
import { clone, uid, type CanvasElement } from "../../../model";
import { exportSize } from "../exportGeometry";
import { readZip, writeZip, type Notebook } from "./archive";
import { colorFields, writeInk } from "./ink";
import { concat, field, frame, lz4Encode, real } from "./wire";
const id = () => uid().toUpperCase();
const point = (x: number, y: number) => concat([real(1, x), real(2, y)]);
export async function writeGoodnotes(
  notebook: Notebook,
): Promise<{ data: Uint8Array; warnings: string[] }> {
  if (!notebook.pages.length || notebook.pages.length > 100)
    throw new Error("Export 1–100 Goodnotes pages at a time.");
  const root = id(),
    device = id(),
    revision = concat([field(1, 1), field(2, 1)]),
    files: Record<string, Uint8Array> = {
      "document.info.pb": field(1, 4),
      "schema.pb": field(1, 31),
    };
  const events: Uint8Array[] = [
    frame(
      concat([
        field(1, root),
        field(
          30,
          concat([
            field(1, root),
            field(2, concat([field(1, notebook.name), field(2, revision)])),
            field(3, concat([field(1, device), field(2, revision)])),
            field(15, 4),
            field(20, 31),
          ]),
        ),
      ]),
    ),
  ];
  const notesIndex: Uint8Array[] = [],
    attachmentIndex: Uint8Array[] = [],
    assets = new Map(notebook.assets.map((asset) => [asset.id, asset]));
  const warnings = new Set<string>([
    "Goodnotes native export is experimental. Text and ink remain editable; verify the exported notebook in Goodnotes before deleting the source. Font metrics, pressure, and highlighter blending can differ.",
    "Comments, templates, group locks, links, and editing history are not exported. List markers become text; text alignment and wrapping can differ.",
  ]);
  for (const [index, source] of notebook.pages.entries()) {
    const page = id().slice(0, -1) + "A",
      note = page.slice(0, -1) + "B",
      template = id(),
      attachment = id(),
      document = clone(source.document);
    const bounds = exportSize(document, 2048).bounds;
    const scale = Math.min(1, 14400 / bounds.width, 14400 / bounds.height);
    const pdf = await PDFDocument.create(),
      sheet = pdf.addPage([
        Math.max(1, bounds.width * scale),
        Math.max(1, bounds.height * scale),
      ]);
    const background = document.settings.backgroundColor || "#ffffff";
    if (/^#[a-f0-9]{6}$/i.test(background))
      sheet.drawRectangle({
        x: 0,
        y: 0,
        width: sheet.getWidth(),
        height: sheet.getHeight(),
        color: rgb(
          parseInt(background.slice(1, 3), 16) / 255,
          parseInt(background.slice(3, 5), 16) / 255,
          parseInt(background.slice(5, 7), 16) / 255,
        ),
      });
    files[`attachments/${attachment}`] = await pdf.save({
      useObjectStreams: false,
    });
    attachmentIndex.push(
      frame(
        concat([field(1, attachment), field(2, `attachments/${attachment}`)]),
      ),
    );
    events.push(
      frame(
        concat([
          field(1, template),
          field(
            2,
            concat([
              field(1, root),
              field(2, template),
              field(4, attachment),
              field(5, 1),
            ]),
          ),
        ]),
      ),
    );
    events.push(
      frame(
        concat([
          field(1, page),
          field(
            54,
            concat([
              field(1, root),
              field(2, page),
              field(3, concat([field(1, template), field(2, revision)])),
              field(
                4,
                concat([
                  field(1, index.toString().padStart(6, "0")),
                  field(2, revision),
                ]),
              ),
              field(15, 31),
              field(16, 1),
            ]),
          ),
        ]),
      ),
    );
    events.push(
      frame(
        concat([
          field(1, page),
          field(
            64,
            concat([field(1, root), field(2, page), field(3, source.name)]),
          ),
        ]),
      ),
    );
    notesIndex.push(frame(concat([field(1, note), field(2, `notes/${note}`)])));
    const records: Uint8Array[] = [];
    for (const [z, sourceElement] of document.elements.entries()) {
      const element = clone(sourceElement),
        identity = id();
      element.x = (element.x - bounds.x) * scale;
      element.y = (element.y - bounds.y) * scale;
      element.width *= scale;
      element.height *= scale;
      if (element.fontSize) element.fontSize *= scale;
      if (element.penWidth) element.penWidth *= scale;
      if (element.type === "stroke") {
        records.push(
          metadata(identity, 24, z + 1, revision),
          frame(field(7, writeInk(element, identity, revision))),
        );
        if (element.pressures?.length)
          warnings.add(
            "Pressure-varying ink uses editable constant-width strokes in Goodnotes exports.",
          );
      } else if (element.type === "text") {
        if (element.rotation)
          throw new Error(
            "Rotated text must be converted to an individual image before Goodnotes export.",
          );
        const font = element.bold ? "HelveticaNeue-Bold" : "HelveticaNeue";
        const style = concat([
          field(30, font),
          real(40, element.fontSize || 24),
          field(3, colorFields(element.color || "#000000")),
          ...(element.italic ? [field(50, 1)] : []),
        ]);
        const rich = field(
          1,
          concat([field(1, listText(element)), field(2, style)]),
        );
        const content = concat([
          field(1, identity),
          field(2, 31),
          field(3, revision),
          field(
            20,
            concat([
              field(
                1,
                point(
                  element.x - element.width / 2,
                  element.y - element.height / 2,
                ),
              ),
              real(3, 1),
            ]),
          ),
          field(
            32,
            concat([
              field(1, field(2, lz4Encode(rich))),
              field(2, point(element.width, element.height)),
              field(5, field(1, style)),
            ]),
          ),
        ]);
        records.push(
          metadata(identity, 31, z + 1, revision),
          frame(field(21, content)),
        );
      } else if (element.type === "image") {
        const asset = assets.get(element.assetId!);
        if (!asset) throw new Error("An exported image is unavailable.");
        const attachment = id(),
          crop = element.crop || { x: 0, y: 0, width: 1, height: 1 },
          width = element.width / crop.width,
          height = element.height / crop.height;
        files[`attachments/${attachment}`] = asset.data;
        attachmentIndex.push(
          frame(
            concat([
              field(1, attachment),
              field(2, `attachments/${attachment}`),
            ]),
          ),
        );
        const originalX = element.x - element.width / 2 - crop.x * width,
          originalY = element.y - element.height / 2 - crop.y * height;
        const geometry = concat([
          field(
            2,
            concat([
              field(1, point(originalX, originalY)),
              field(2, point(width, height)),
            ]),
          ),
          field(
            3,
            concat([
              field(1, point(element.x, element.y)),
              field(2, point(element.width, element.height)),
              real(3, element.rotation),
            ]),
          ),
        ]);
        records.push(
          frame(
            concat([
              field(1, identity),
              field(2, revision),
              field(4, attachment),
              field(8, 1),
              field(9, z + 1),
              field(14, 1),
              field(16, 24),
            ]),
          ),
          frame(
            field(
              1,
              concat([
                field(1, identity),
                geometry,
                field(4, attachment),
                field(6, 3),
                field(15, revision),
                field(18, 24),
              ]),
            ),
          ),
        );
      } else
        throw new Error(
          "Unsupported Goodnotes object: convert it to an individual image before exporting.",
        );
    }
    files[`notes/${note}`] = concat(records);
    if (scale < 1)
      warnings.add(
        "Very large canvases are scaled to fit Goodnotes/PDF page dimensions while keeping supported objects editable.",
      );
  }
  files["index.notes.pb"] = concat(notesIndex);
  files["index.attachments.pb"] = concat(attachmentIndex);
  files["index.events.pb"] = concat(events);
  return { data: writeZip(files), warnings: [...warnings] };
}
function metadata(id: string, type: number, z: number, revision: Uint8Array) {
  return frame(
    concat([
      field(1, id),
      field(2, revision),
      field(8, 1),
      field(9, z),
      field(14, 1),
      field(16, type),
    ]),
  );
}
export function fromNative(raw: Uint8Array): Notebook {
  const files = readZip(raw),
    manifest = JSON.parse(new TextDecoder().decode(files["manifest.json"]));
  if (manifest.format !== "mindflow" || manifest.version !== 1)
    throw new Error("Unsupported mindflow transfer format.");
  return {
    name: manifest.name,
    pages: manifest.pages,
    assets: manifest.assets.map((asset: { id: string; mime: string }) => ({
      ...asset,
      data: files[`assets/${asset.id}.bin`],
    })),
    warnings: [],
  };
}
// Used after converting PDF previews or unsupported per-object representations.
export function replaceWithImage(
  element: CanvasElement,
  assetId: string,
): CanvasElement {
  return {
    id: element.id,
    type: "image",
    assetId,
    x: element.x,
    y: element.y,
    width: element.width,
    height: element.height,
    rotation: element.rotation,
  };
}

function listText(element: CanvasElement): string {
  const lines = (element.text || "").split("\n");
  if (!element.listStyle) return element.text || "";
  return lines
    .map(
      (line, index) =>
        `${element.listStyle === "number" ? `${index + 1}.` : element.listStyle === "check" ? (element.checked?.includes(index) ? "☑" : "☐") : "•"} ${line}`,
    )
    .join("\n");
}
