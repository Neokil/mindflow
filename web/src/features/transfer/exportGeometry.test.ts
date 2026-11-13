import { expect, it } from "vitest";
import { emptyDocument } from "../../model";
import { exportBounds, exportSize, safeFilename } from "./exportGeometry";
it("fits rotated elements and stroke widths including negative coordinates", () => {
  const doc = emptyDocument();
  doc.elements = [
    {
      id: "a",
      type: "shape",
      shape: "rectangle",
      x: -200,
      y: 100,
      width: 200,
      height: 100,
      rotation: Math.PI / 2,
      borderWidth: 10,
    },
    {
      id: "s",
      type: "stroke",
      x: 200,
      y: 0,
      width: 10,
      height: 10,
      rotation: 0,
      penWidth: 40,
      points: [0, 0, 1, 1],
    },
  ];
  const b = exportBounds(doc);
  expect(b.x).toBeCloseTo(-287);
  expect(b.y).toBeCloseTo(-57);
  expect(b.width).toBeCloseTo(544);
  expect(b.height).toBeCloseTo(294);
  const size = exportSize(doc, 2048);
  expect(Math.max(size.width, size.height)).toBe(2048);
});
it("uses finite bounds for an empty canvas and safe portable filenames", () => {
  const size = exportSize(emptyDocument(), 1024);
  expect(size.width).toBe(1024);
  expect(size.height).toBe(717);
  expect(safeFilename("../ a/b:*")).toBe("_ a_b__");
  expect(safeFilename("...")).toBe("workspace");
});
