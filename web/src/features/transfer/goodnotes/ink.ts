import { uid, type CanvasElement } from "../../../model";
import {
  bytes,
  concat,
  field,
  float,
  integer,
  invalid,
  lz4Encode,
  message,
  nested,
  real,
  type LZ4,
  type Message,
} from "./wire";

type Value = number | Value[];
type Node = { kind: string; children: Node[] };
function tpl(data: Uint8Array): { format: string; values: Value[] } {
  if (
    data.length < 9 ||
    String.fromCharCode(...data.subarray(0, 3)) !== "tpl" ||
    data[3] & 1
  )
    invalid();
  const view = new DataView(data.buffer, data.byteOffset, data.byteLength);
  if (view.getUint32(4, true) !== data.length) invalid();
  const end = data.indexOf(0, 8);
  if (end < 8 || end > 2048) invalid();
  const format = new TextDecoder().decode(data.subarray(8, end));
  let index = 0,
    count = 0,
    position = end + 1;
  function nodes(depth = 0, group = false): Node[] {
    if (depth > 16) invalid();
    const result = [];
    while (index < format.length) {
      const kind = format[index++];
      if (kind === ")") {
        if (!group) invalid();
        return result;
      }
      let children: Node[] = [];
      if (kind === "A" || kind === "S") {
        if (format[index++] !== "(") invalid();
        children = nodes(depth + 1, true);
        if (!children.length) invalid();
      } else if (!"cjviuIUf".includes(kind)) invalid();
      result.push({ kind, children });
    }
    if (group) invalid();
    return result;
  }
  function take(size: number) {
    if (position + size > data.length) invalid();
    const at = position;
    position += size;
    return at;
  }
  function read(node: Node): Value {
    if (++count > 1000000) invalid();
    if (node.kind === "S") return node.children.map(read);
    if (node.kind === "A") {
      const length = view.getUint32(take(4), true);
      if (length > 200000 || length * node.children.length > 1000000 - count)
        invalid();
      return Array.from({ length }, () =>
        node.children.length === 1
          ? read(node.children[0])
          : node.children.map(read),
      );
    }
    switch (node.kind) {
      case "c":
        return view.getInt8(take(1));
      case "j":
        return view.getInt16(take(2), true);
      case "v":
        return view.getUint16(take(2), true);
      case "i":
        return view.getInt32(take(4), true);
      case "u":
        return view.getUint32(take(4), true);
      case "I":
        return Number(view.getBigInt64(take(8), true));
      case "U":
        return Number(view.getBigUint64(take(8), true));
      case "f":
        return view.getFloat64(take(8), true);
      default:
        return invalid();
    }
  }
  const values = nodes().map(read);
  if (position !== data.length) invalid();
  return { format, values };
}
function array(value: Value | undefined): Value[] {
  return Array.isArray(value) ? value : [];
}
function bits(value: Value | undefined) {
  if (typeof value !== "number") invalid();
  const buffer = new ArrayBuffer(4),
    view = new DataView(buffer);
  view.setUint32(0, value, true);
  const number = view.getFloat32(0, true);
  if (!Number.isFinite(number) || Math.abs(number) > 1e9) invalid();
  return number;
}
export function rgba(fields: Message, fallback = "#000000") {
  if (!fields.length) return fallback;
  return (
    "#" +
    [1, 2, 3, 4]
      .map((n) =>
        Math.round(
          Math.min(1, Math.max(0, float(fields, n, n === 4 ? 1 : 0))) * 255,
        )
          .toString(16)
          .padStart(2, "0"),
      )
      .join("")
  );
}
export function stroke(
  points: number[],
  color: string,
  width: number,
  highlight = false,
  pressures?: number[],
): CanvasElement {
  if (points.length < 4 || points.length > 40000) invalid();
  let left = Infinity,
    top = Infinity,
    right = -Infinity,
    bottom = -Infinity;
  for (let i = 0; i < points.length; i += 2) {
    if (
      ![points[i], points[i + 1]].every(
        (n) => Number.isFinite(n) && Math.abs(n) <= 1e9,
      )
    )
      invalid();
    left = Math.min(left, points[i]);
    right = Math.max(right, points[i]);
    top = Math.min(top, points[i + 1]);
    bottom = Math.max(bottom, points[i + 1]);
  }
  const w = Math.max(0.1, right - left),
    h = Math.max(0.1, bottom - top);
  return {
    id: uid(),
    type: "stroke",
    x: left + w / 2,
    y: top + h / 2,
    width: w,
    height: h,
    rotation: 0,
    points: points.map((n, i) => (n - (i % 2 ? top : left)) / (i % 2 ? h : w)),
    color,
    penWidth: Math.min(100, Math.max(0.1, width)),
    highlight,
    ...(pressures ? { pressures } : {}),
  };
}
export function readInk(payload: Uint8Array, lz4: LZ4): CanvasElement[] {
  const record = message(payload),
    compressed = bytes(record, 2);
  if (!compressed.length) return [];
  const decoded = lz4.decode(compressed),
    image = tpl(decoded.data),
    values = image.values;
  const offset = nested(record, 6),
    dx = float(offset, 1),
    dy = float(offset, 2);
  const color = rgba(nested(record, 4));
  const highlight =
    float(nested(record, 4), 4, 1) < 0.95 || integer(record, 21) === 26;
  const width = image.format.startsWith("vu") ? bits(values[1]) : 2;
  const groups: { points: number[]; pressures?: number[] }[] = [];
  if (image.format === "vuA(v)A(S(uu))A(S(uuuu))vA(f)") {
    const starts = array(values[3]),
      segments = array(values[4]),
      visible = array(values[2]);
    let points: number[] = [];
    const add = (x: Value | undefined, y: Value | undefined) => {
      const px = bits(x) + dx,
        py = bits(y) + dy;
      if (
        !points.length ||
        Math.hypot(px - points.at(-2)!, py - points.at(-1)!) > 0.001
      )
        points.push(px, py);
    };
    for (const start of starts) {
      const pair = array(start);
      add(pair[0], pair[1]);
    }
    for (let index = 0; index < segments.length; index++) {
      if (visible[index + 1] === 0 && points.length) {
        groups.push({ points });
        points = [];
      }
      const tuple = array(segments[index]);
      add(tuple[0], tuple[1]);
      add(tuple[2], tuple[3]);
    }
    if (points.length) groups.push({ points });
  } else if (image.format.includes("A(S(uuuuuuuuuuu))")) {
    const points: number[] = [],
      pressures: number[] = [];
    const add = (tuple: Value[], start: number) => {
      points.push(bits(tuple[start]) + dx, bits(tuple[start + 1]) + dy);
      pressures.push(Math.min(1, Math.max(0, bits(tuple[start + 4]))));
    };
    for (const start of array(values[3])) add(array(start), 0);
    for (const segment of array(values[4])) {
      add(array(segment), 1);
      add(array(segment), 6);
    }
    if (points.length) groups.push({ points, pressures });
  } else if (
    image.format === "vuA(v)A(u)A(u)A(v)A(v)A(u)A(u)A(u)A(u)A(v)" ||
    image.format === "vA(v)A(u)A(u)A(v)A(v)A(u)A(u)A(u)A(u)A(v)"
  ) {
    // Explicit native CGPath commands, not coordinate guesses from opaque arrays.
    const shift = image.format.startsWith("vu") ? 1 : 0;
    const counts = array(values[4 + shift]),
      commands = array(values[5 + shift]),
      starts = array(values[6 + shift]),
      curves = array(values[8 + shift]),
      arcs = array(values[9 + shift]);
    let ci = 0,
      si = 0,
      bi = 0,
      ai = 0;
    for (const count of counts) {
      if (typeof count !== "number" || count > 20000 || count < 0) invalid();
      let points: number[] = [];
      for (let i = 0; i < count; i++) {
        const command = commands[ci++];
        if (command === (shift ? 2 : 0)) {
          if (points.length >= 4) groups.push({ points });
          points = [bits(starts[si++]) + dx, bits(starts[si++]) + dy];
        } else if (command === (shift ? 4 : 2)) {
          if (points.length < 2) invalid();
          const x0 = points.at(-2)!,
            y0 = points.at(-1)!,
            x1 = bits(curves[bi++]) + dx,
            y1 = bits(curves[bi++]) + dy,
            x2 = bits(curves[bi++]) + dx,
            y2 = bits(curves[bi++]) + dy,
            x3 = bits(curves[bi++]) + dx,
            y3 = bits(curves[bi++]) + dy;
          for (let step = 1; step <= 12; step++) {
            const t = step / 12,
              a = 1 - t;
            points.push(
              a ** 3 * x0 +
                3 * a ** 2 * t * x1 +
                3 * a * t ** 2 * x2 +
                t ** 3 * x3,
              a ** 3 * y0 +
                3 * a ** 2 * t * y1 +
                3 * a * t ** 2 * y2 +
                t ** 3 * y3,
            );
          }
        } else if (command === (shift ? 5 : 3)) {
          const cx = bits(arcs[ai++]) + dx,
            cy = bits(arcs[ai++]) + dy,
            radius = bits(arcs[ai++]),
            from = bits(arcs[ai++]),
            to = bits(arcs[ai++]);
          for (let step = 0; step <= 32; step++) {
            const angle = from + ((to - from) * step) / 32;
            points.push(
              cx + radius * Math.cos(angle),
              cy + radius * Math.sin(angle),
            );
          }
        } else throw new Error("Unsupported Goodnotes ink encoding.");
      }
      if (points.length) groups.push({ points });
    }
  } else throw new Error("Unsupported Goodnotes ink encoding.");
  return groups
    .filter((g) => g.points.length)
    .map((g) => {
      if (g.points.length === 2) {
        g.points.push(...g.points);
        g.pressures?.push(g.pressures[0]);
      }
      return stroke(g.points, color, width, highlight, g.pressures);
    });
}
export function colorFields(color: string, alpha?: number): Uint8Array {
  const raw =
    color === "transparent"
      ? "00000000"
      : color.replace("#", "").padEnd(8, "f");
  return concat(
    [1, 2, 3, 4].map((n) =>
      real(
        n,
        n === 4 && alpha !== undefined
          ? alpha
          : parseInt(raw.slice((n - 1) * 2, n * 2), 16) / 255,
      ),
    ),
  );
}
export function writeInk(
  element: CanvasElement,
  id: string,
  revision: Uint8Array,
): Uint8Array {
  const points = element.points!;
  if (points.length < 4 || points.length > 40000) invalid();
  const transformed: number[] = [];
  for (let i = 0; i < points.length; i += 2) {
    const x = (points[i] - 0.5) * element.width,
      y = (points[i + 1] - 0.5) * element.height;
    transformed.push(
      element.x +
        x * Math.cos(element.rotation) -
        y * Math.sin(element.rotation),
      element.y +
        x * Math.sin(element.rotation) +
        y * Math.cos(element.rotation),
    );
  }
  const format = new TextEncoder().encode("vuA(v)A(S(uu))A(S(uuuu))vA(f)\0"),
    count = points.length / 2;
  const buffer = new Uint8Array(
      6 + 4 + 2 * count + 4 + 8 + 4 + 16 * (count - 1) + 6,
    ),
    view = new DataView(buffer.buffer);
  let at = 0;
  const short = (n: number) => {
      view.setUint16(at, n, true);
      at += 2;
    },
    int = (n: number) => {
      view.setUint32(at, n, true);
      at += 4;
    },
    number = (n: number) => {
      view.setFloat32(at, n, true);
      at += 4;
    };
  short(2);
  number(element.penWidth || 2);
  int(count);
  for (let i = 0; i < count; i++) short(1);
  int(1);
  number(transformed[0]);
  number(transformed[1]);
  int(count - 1);
  for (let i = 2; i < transformed.length; i += 2)
    for (const n of transformed.slice(i - 2, i + 2)) number(n);
  short(1);
  int(0);
  const header = new Uint8Array(8);
  header.set(new TextEncoder().encode("tpl"));
  new DataView(header.buffer).setUint32(
    4,
    8 + format.length + buffer.length,
    true,
  );
  return concat([
    field(1, id),
    field(2, lz4Encode(concat([header, format, buffer]))),
    field(
      4,
      colorFields(
        element.color || "#000000",
        element.highlight ? 0.35 : undefined,
      ),
    ),
    field(15, revision),
    field(21, 24),
  ]);
}
