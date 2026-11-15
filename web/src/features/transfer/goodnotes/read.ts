import { emptyDocument, uid, type CanvasElement } from "../../../model";
import { readZip, type Notebook, type PortableAsset } from "./archive";
import { readInk, rgba } from "./ink";
import {
  bytes,
  float,
  integer,
  invalid,
  LZ4,
  message,
  nested,
  records,
  text,
  type Message,
} from "./wire";
function key(id: string) {
  return id.length >= 32 ? id.slice(0, 32) : id;
}
function geometry(fields: Message) {
  return { x: float(fields, 1), y: float(fields, 2) };
}
function mime(data: Uint8Array) {
  if (
    data.length > 4 &&
    data[0] === 137 &&
    data[1] === 80 &&
    data[2] === 78 &&
    data[3] === 71
  )
    return "image/png";
  if (data[0] === 255 && data[1] === 216) return "image/jpeg";
  if (String.fromCharCode(...data.subarray(0, 5)) === "%PDF-")
    return "application/pdf";
  if (String.fromCharCode(...data.subarray(8, 12)) === "WEBP")
    return "image/webp";
  return "";
}
function indexed(data: Uint8Array | undefined): Map<string, string> {
  if (!data) invalid();
  const values = new Map<string, string>();
  for (const record of records(data)) {
    const id = text(record, 1),
      path = text(record, 2);
    if (!id || !path || values.has(id)) invalid();
    values.set(id, path);
  }
  return values;
}
function readText(record: Message, lz4: LZ4): CanvasElement | undefined {
  const source = nested(record, 21),
    box = nested(source, 32);
  if (!box.length) return;
  const compressed = bytes(nested(box, 1), 2);
  if (!compressed.length) return;
  const content = message(lz4.decode(compressed).data),
    runs = content
      .filter((f) => f.value instanceof Uint8Array)
      .map((f) => message(f.value as Uint8Array));
  const string = runs.map((run) => text(run, 1)).join("");
  if (!string) return;
  if (string.length > 100000)
    throw new Error(
      "A Goodnotes text box exceeds 100,000 characters. Split the source text box before importing.",
    );
  const style = runs.length ? nested(runs[0], 2) : nested(nested(box, 5), 1);
  const position = geometry(nested(nested(source, 20), 1)),
    size = geometry(nested(box, 2)),
    inset = geometry(nested(box, 10));
  const fontSize = Math.min(
    2000,
    Math.max(1, Math.round(float(style, 40, 24))),
  );
  return {
    id: uid(),
    type: "text",
    x: position.x + Math.max(0.1, size.x + inset.x * 2) / 2,
    y: position.y + Math.max(0.1, size.y + inset.y * 2) / 2,
    width: Math.max(0.1, size.x + inset.x * 2),
    height: Math.max(0.1, size.y + inset.y * 2),
    rotation: 0,
    text: string,
    fontSize,
    color: rgba(nested(style, 3)),
    bold:
      text(style, 30).toLowerCase().includes("bold") ||
      integer(style, 60) > 1e15,
    italic: integer(style, 50) === 1,
    align: "left",
    verticalAlign: "top",
  };
}
function readImage(
  record: Message,
  next: Message | undefined,
  attachments: Map<string, string>,
  files: Record<string, Uint8Array>,
): { element: CanvasElement; data: Uint8Array } | undefined {
  const attachment = text(record, 7) || text(record, 4),
    path = attachments.get(attachment);
  if (!path || integer(record, 3) === 1) return;
  const data = files[path];
  if (!data) invalid();
  for (const candidate of [record, next || []]) {
    for (const f of candidate) {
      if (!(f.value instanceof Uint8Array) || f.value.length > 4096) continue;
      let container: Message;
      try {
        container = message(f.value);
      } catch {
        continue;
      }
      const rect = nested(container, 2),
        originalPosition = geometry(nested(rect, 1)),
        originalSize = geometry(nested(rect, 2));
      if (originalSize.x <= 0 || originalSize.y <= 0) continue;
      const transform = nested(container, 3),
        size = geometry(nested(transform, 2)),
        center = geometry(nested(transform, 1));
      const cropped = size.x > 0 && size.y > 0;
      const width = cropped ? size.x : originalSize.x,
        height = cropped ? size.y : originalSize.y;
      const x = cropped ? center.x : originalPosition.x + width / 2,
        y = cropped ? center.y : originalPosition.y + height / 2;
      const cropX = (x - width / 2 - originalPosition.x) / originalSize.x,
        cropY = (y - height / 2 - originalPosition.y) / originalSize.y;
      const cropWidth = width / originalSize.x,
        cropHeight = height / originalSize.y;
      const crop =
        cropX >= 0 &&
        cropY >= 0 &&
        cropX + cropWidth <= 1.00001 &&
        cropY + cropHeight <= 1.00001 &&
        (cropX > 0.00001 ||
          cropY > 0.00001 ||
          cropWidth < 0.99999 ||
          cropHeight < 0.99999)
          ? {
              x: cropX,
              y: cropY,
              width: Math.min(cropWidth, 1 - cropX),
              height: Math.min(cropHeight, 1 - cropY),
            }
          : undefined;
      return {
        element: {
          id: uid(),
          type: "image",
          x,
          y,
          width,
          height,
          rotation: float(transform, 3),
          ...(crop ? { crop } : {}),
        },
        data,
      };
    }
  }
}
export function readGoodnotes(raw: Uint8Array): Notebook {
  const files = readZip(raw);
  if (integer(message(files["document.info.pb"] || new Uint8Array()), 1) !== 4)
    throw new Error(
      "Only modern Goodnotes ZIP notebooks are supported. Export older notebooks as PDF.",
    );
  const entries = indexed(files["index.notes.pb"]),
    attachments = indexed(files["index.attachments.pb"]),
    events = records(files["index.events.pb"] || new Uint8Array());
  if (!entries.size || entries.size > 100) invalid();
  const notebook: Notebook = {
      name: "Imported Goodnotes notebook",
      pages: [],
      assets: [],
      warnings: [],
    },
    notes = new Set<string>();
  const templates = new Map<string, { attachment: string; page: number }>(),
    backgrounds = new Map<string, string>(),
    titles = new Map<string, string>(),
    order = new Map<string, string>(),
    deleted = new Set<string>();
  for (const event of events) {
    const root = nested(event, 30),
      title = text(nested(root, 2), 1);
    if (title) notebook.name = title.slice(0, 180);
    const template = nested(event, 2);
    if (text(template, 2) && text(template, 4))
      templates.set(text(template, 2), {
        attachment: text(template, 4),
        page: integer(template, 5, 1),
      });
    const page = event.some((f) => f.number === 54)
      ? nested(event, 54)
      : nested(event, 3);
    if (text(page, 2)) {
      backgrounds.set(key(text(page, 2)), text(nested(page, 3), 1));
      order.set(key(text(page, 2)), text(nested(page, 4), 1));
    }
    const renamed = nested(event, 64);
    if (text(renamed, 2) && text(renamed, 3))
      titles.set(key(text(renamed, 2)), text(renamed, 3));
    const removed = nested(event, 56);
    if (text(removed, 2)) deleted.add(key(text(removed, 2)));
  }
  const lz4 = new LZ4();
  let elementCount = 0;
  const addAsset = (
    data: Uint8Array,
    pdfPage = 1,
    background = false,
  ): PortableAsset => {
    if (notebook.assets.length >= 1000 || data.length > 50 * 1024 * 1024)
      invalid();
    const type = mime(data);
    if (!type)
      throw new Error(
        "The notebook contains an unsupported image format. Convert it to PNG or PDF in Goodnotes first.",
      );
    const asset = {
      id: `a${notebook.assets.length + 1}`,
      mime: type,
      data,
      ...(type === "application/pdf" ? { pdfPage, background } : {}),
    };
    notebook.assets.push(asset);
    return asset;
  };
  const sorted = [...entries]
    .filter(([id]) => !deleted.has(key(id)))
    .sort((a, b) =>
      (order.get(key(a[0])) || "").localeCompare(order.get(key(b[0])) || ""),
    );
  for (const [id, path] of sorted) {
    if (!files[path]) invalid();
    const source = records(files[path]),
      metadata = new Map<string, { erased: boolean; z: number }>();
    for (const record of source)
      if (text(record, 1) && record.some((f) => f.number === 16))
        metadata.set(text(record, 1), {
          erased: integer(record, 3) === 1,
          z: integer(record, 9),
        });
    const elements: { element: CanvasElement; z: number }[] = [];
    const template = templates.get(backgrounds.get(key(id)) || ""),
      background = template && attachments.get(template.attachment);
    if (background && files[background]) {
      const asset = addAsset(files[background], template!.page, true);
      elements.push({
        element: {
          id: uid(),
          type: "image",
          assetId: asset.id,
          x: 306,
          y: 396,
          width: 612,
          height: 792,
          rotation: 0,
        },
        z: -Infinity,
      });
    }
    for (let index = 0; index < source.length; index++) {
      const record = source[index];
      const ink = bytes(record, 7),
        textPayload = bytes(record, 21);
      let object: Message | undefined;
      // Image attachment IDs are strings, while stroke/text payloads are messages.
      if (ink.length && ink[0] === 10) object = message(ink);
      else if (textPayload.length) object = message(textPayload);
      const identity = object ? text(object, 1) : text(record, 1),
        meta = metadata.get(identity);
      if (meta?.erased) continue;
      const z = meta?.z || index;
      if (object && ink.length && bytes(object, 2).length) {
        try {
          for (const element of readInk(ink, lz4))
            elements.push({ element, z });
        } catch (error) {
          if (
            (error as Error).message === "Unsupported Goodnotes ink encoding."
          )
            notes.add(
              "Some unsupported Goodnotes ink encodings were omitted. Keep the original file.",
            );
          else throw error;
        }
      } else if (textPayload.length) {
        const element = readText(record, lz4);
        if (element) elements.push({ element, z });
        else
          notes.add(
            "Some Goodnotes shapes, stickers, or other unsupported objects were omitted. Keep the original file.",
          );
      }
      const image = readImage(record, source[index + 1], attachments, files);
      if (image) {
        const asset = addAsset(image.data);
        image.element.assetId = asset.id;
        elements.push({ element: image.element, z });
      }
    }
    elements.sort((a, b) => a.z - b.z);
    elementCount += elements.length;
    if (elementCount > 100000) invalid();
    // Preserve large notes without silently dropping strokes at mindflow's page limit.
    for (let start = 0; start < Math.max(1, elements.length); start += 5000) {
      if (notebook.pages.length >= 100) invalid();
      const document = emptyDocument();
      document.settings.background = "plain";
      document.settings.backgroundColor = "#ffffff";
      document.elements = elements
        .slice(start, start + 5000)
        .map((entry) => entry.element);
      const name = titles.get(key(id)) || `Page ${notebook.pages.length + 1}`;
      notebook.pages.push({
        id: `p${notebook.pages.length + 1}`,
        name: (name + (start ? ` (continued ${start / 5000 + 1})` : "")).slice(
          0,
          190,
        ),
        document,
      });
    }
    if (elements.length > 5000)
      notes.add(
        "Large Goodnotes pages are split into continuation pages of at most 5,000 elements, preserving object coordinates and stacking order within each continuation.",
      );
  }
  if (!notebook.pages.length) invalid();
  notes.add(
    "Goodnotes import is experimental. Text uses one style per text box; font metrics, pressure, curves, and highlighter blending can differ. Comments, recordings, links, and editing history are not imported.",
  );
  notebook.warnings = [...notes];
  return notebook;
}
