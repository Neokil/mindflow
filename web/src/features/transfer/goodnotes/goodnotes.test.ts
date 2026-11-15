import { describe, expect, it } from "vitest";
import fixture from "./fixture.json";
import { readGoodnotes } from "./read";
import { writeGoodnotes } from "./write";
import { exportBounds } from "../exportGeometry";
import {
  concat,
  field,
  frame,
  LZ4,
  lz4Encode,
  message,
  records,
  text,
} from "./wire";

describe("Goodnotes native conversion", () => {
  it("decodes typed protobuf records and rejects truncated framing", () => {
    const encoded = frame(field(1, "hello"));
    expect(text(records(encoded)[0], 1)).toBe("hello");
    expect(() => records(encoded.subarray(0, encoded.length - 1))).toThrow();
    expect(() => message(new Uint8Array([128, 128, 128]))).toThrow();
  });
  it("round-trips native Apple LZ4 blocks with strict expansion limits", () => {
    const source = new TextEncoder().encode(
      "native editable ink ".repeat(1000),
    );
    expect(new LZ4().decode(lz4Encode(source)).data).toEqual(source);
    const malformed = lz4Encode(source);
    new DataView(malformed.buffer).setUint32(4, 0x7fffffff, true);
    expect(() => new LZ4().decode(malformed)).toThrow();
    expect(() =>
      new LZ4().decode(
        concat([
          new TextEncoder().encode("bv41"),
          new Uint8Array(8),
          new Uint8Array([0]),
        ]),
      ),
    ).toThrow();
  });
  it("reads a synthetic native notebook and keeps text, ink, and images editable", async () => {
    const source = Uint8Array.from(atob(fixture.archive), (c) =>
      c.charCodeAt(0),
    );
    const imported = readGoodnotes(source);
    expect(imported.pages).toHaveLength(1);
    expect(
      imported.pages[0].document.elements.map((e) => e.type).sort(),
    ).toEqual(["image", "image", "stroke", "text"]);
    expect(
      imported.pages[0].document.elements.find((e) => e.type === "text")?.text,
    ).toBe("mindflow editable text");
    expect(
      imported.pages[0].document.elements.find((e) => e.type === "stroke")
        ?.points,
    ).toHaveLength(12);
    // Drop the PDF background for this model-only test; browser tests render it.
    const pdfs = new Set(
      imported.assets
        .filter((a) => a.mime === "application/pdf")
        .map((a) => a.id),
    );
    imported.pages[0].document.elements =
      imported.pages[0].document.elements.filter((e) => !pdfs.has(e.assetId!));
    imported.assets = imported.assets.filter((a) => !pdfs.has(a.id));
    const originalText = imported.pages[0].document.elements.find(
      (e) => e.type === "text",
    )!;
    expect(originalText.x).toBe(250);
    expect(originalText.y).toBe(102.5);
    const bounds = exportBounds(imported.pages[0].document);
    const exported = await writeGoodnotes(imported);
    const restored = readGoodnotes(exported.data);
    expect(restored.pages).toHaveLength(1);
    expect(
      restored.pages[0].document.elements.find((e) => e.type === "text")?.text,
    ).toBe("mindflow editable text");
    expect(
      restored.pages[0].document.elements.find((e) => e.type === "stroke")
        ?.points,
    ).toHaveLength(12);
    const restoredText = restored.pages[0].document.elements.find(
      (e) => e.type === "text",
    )!;
    expect(restoredText.x).toBeCloseTo(originalText.x - bounds.x, 3);
    expect(restoredText.y).toBeCloseTo(originalText.y - bounds.y, 3);
    expect(restored.assets.find((a) => a.mime === "image/png")?.data).toEqual(
      imported.assets[0].data,
    );
  });
});
