// Goodnotes uses varint-framed protobuf records and Apple's LZ4 block framing.
// Format research: parser-for-goodnotes (MIT); see THIRD_PARTY_NOTICES.md.
export type Field = {
  number: number;
  wire: number;
  value: bigint | Uint8Array;
};
export type Message = Field[];
const decoder = new TextDecoder("utf-8", { fatal: true });
export const MAX_EXPANDED = 128 * 1024 * 1024;
export function invalid(): never {
  throw new Error(
    "The Goodnotes file is corrupt, unsupported, or exceeds the conversion limits.",
  );
}
function varint(data: Uint8Array, position: { value: number }): bigint {
  let value = 0n;
  for (let shift = 0n; shift < 70n; shift += 7n) {
    if (position.value >= data.length) invalid();
    const byte = data[position.value++];
    if (shift === 63n && byte > 1) invalid();
    value |= BigInt(byte & 127) << shift;
    if (!(byte & 128)) return value;
  }
  return invalid();
}
export function message(data: Uint8Array): Message {
  const fields: Message = [],
    position = { value: 0 };
  while (position.value < data.length) {
    if (fields.length >= 10000) invalid();
    const tag = varint(data, position),
      number = Number(tag >> 3n),
      wire = Number(tag & 7n);
    if (number < 1 || number > 536870911) invalid();
    let value: bigint | Uint8Array;
    if (wire === 0) value = varint(data, position);
    else if (wire === 1 || wire === 5) {
      const length = wire === 1 ? 8 : 4;
      if (position.value + length > data.length) invalid();
      value = 0n;
      for (let i = 0; i < length; i++)
        value |= BigInt(data[position.value++]) << BigInt(i * 8);
    } else if (wire === 2) {
      const length = Number(varint(data, position));
      if (
        !Number.isSafeInteger(length) ||
        length < 0 ||
        length > data.length - position.value
      )
        invalid();
      value = data.subarray(position.value, position.value + length);
      position.value += length;
    } else return invalid();
    fields.push({ number, wire, value });
  }
  return fields;
}
export function records(data: Uint8Array): Message[] {
  const result: Message[] = [],
    position = { value: 0 };
  while (position.value < data.length) {
    const length = Number(varint(data, position));
    if (
      !Number.isSafeInteger(length) ||
      length < 0 ||
      length > data.length - position.value ||
      result.length >= 200000
    )
      invalid();
    result.push(
      message(data.subarray(position.value, position.value + length)),
    );
    position.value += length;
  }
  return result;
}
export function bytes(fields: Message, number: number): Uint8Array {
  const value = fields.find((f) => f.number === number)?.value;
  return value instanceof Uint8Array ? value : new Uint8Array();
}
export function integer(fields: Message, number: number, fallback = 0) {
  const value = fields.find((f) => f.number === number)?.value;
  return typeof value === "bigint" ? Number(value) : fallback;
}
export function text(fields: Message, number: number): string {
  try {
    return decoder.decode(bytes(fields, number));
  } catch {
    return "";
  }
}
export function nested(fields: Message, number: number): Message {
  return message(bytes(fields, number));
}
export function float(fields: Message, number: number, fallback = 0): number {
  const field = fields.find((f) => f.number === number);
  if (!field || typeof field.value !== "bigint") return fallback;
  const data = new ArrayBuffer(8),
    view = new DataView(data);
  view.setBigUint64(0, field.value, true);
  const value =
    field.wire === 1
      ? view.getFloat64(0, true)
      : field.wire === 5
        ? view.getFloat32(0, true)
        : NaN;
  if (!Number.isFinite(value) || Math.abs(value) > 1e9) invalid();
  return value;
}
export function concat(parts: Uint8Array[]) {
  const length = parts.reduce((n, part) => n + part.length, 0);
  if (length > MAX_EXPANDED) invalid();
  const result = new Uint8Array(length);
  let offset = 0;
  for (const part of parts) {
    result.set(part, offset);
    offset += part.length;
  }
  return result;
}
export function unsigned(value: bigint | number): Uint8Array {
  let number = BigInt(value);
  if (number < 0n || number > 0xffffffffffffffffn) invalid();
  const bytes = [];
  while (number > 127n) {
    bytes.push(Number(number & 127n) | 128);
    number >>= 7n;
  }
  bytes.push(Number(number));
  return new Uint8Array(bytes);
}
export function field(
  number: number,
  value: string | Uint8Array | number | bigint,
): Uint8Array {
  if (typeof value === "string") value = new TextEncoder().encode(value);
  return value instanceof Uint8Array
    ? concat([unsigned(number * 8 + 2), unsigned(value.length), value])
    : concat([unsigned(number * 8), unsigned(value)]);
}
export function real(number: number, value: number): Uint8Array {
  if (!Number.isFinite(value)) invalid();
  const bytes = new Uint8Array(4);
  new DataView(bytes.buffer).setFloat32(0, value, true);
  return concat([unsigned(number * 8 + 5), bytes]);
}
export function frame(data: Uint8Array): Uint8Array {
  return concat([unsigned(data.length), data]);
}

// Each decoder has a shared expansion budget, independent of archive compression.
export class LZ4 {
  private remaining = MAX_EXPANDED;
  decode(data: Uint8Array): { data: Uint8Array; consumed: number } {
    const output: Uint8Array[] = [];
    let position = 0;
    let history = new Uint8Array(0);
    while (position + 4 <= data.length) {
      const magic = String.fromCharCode(
        ...data.subarray(position, position + 4),
      );
      position += 4;
      if (magic === "bv4$") return { data: concat(output), consumed: position };
      if (!["bv41", "bv4-"].includes(magic) || position + 8 > data.length)
        invalid();
      const view = new DataView(data.buffer, data.byteOffset + position, 8),
        size = view.getUint32(0, true),
        stored = view.getUint32(4, true);
      position += 8;
      if (
        size > this.remaining ||
        size > 16 * 1024 * 1024 ||
        stored > data.length - position
      )
        invalid();
      this.remaining -= size;
      const block = data.subarray(position, position + stored);
      position += stored;
      if (magic === "bv4-") {
        if (stored !== size) invalid();
        const decoded = block.slice();
        output.push(decoded);
        history = concat([history, decoded]).slice(-65536);
        continue;
      }

      const decoded = new Uint8Array(size);
      let source = 0,
        target = 0;
      const length = (base: number) => {
        let count = base;
        if (base === 15) {
          let extension: number;
          do {
            if (source >= block.length) invalid();
            extension = block[source++];
            count += extension;
            if (count > size) invalid();
          } while (extension === 255);
        }
        return count;
      };
      while (source < block.length) {
        const token = block[source++],
          literals = length(token >> 4);
        if (literals > block.length - source || literals > size - target)
          invalid();
        decoded.set(block.subarray(source, source + literals), target);
        target += literals;
        source += literals;
        if (source === block.length) break;
        if (source + 2 > block.length) invalid();
        const offset = block[source] | (block[source + 1] << 8);
        source += 2;
        if (offset === 0 || offset > history.length + target) invalid();
        const count = length(token & 15) + 4;
        if (count > size - target) invalid();
        for (let i = 0; i < count; i++) {
          const from = target - offset;
          decoded[target++] =
            from < 0 ? history[history.length + from] : decoded[from];
        }
      }
      if (target !== size) invalid();
      output.push(decoded);
      history = concat([history, decoded]).slice(-65536);
    }
    return invalid();
  }
}
export function lz4Encode(data: Uint8Array): Uint8Array {
  // An all-literal LZ4 block is valid and interoperable; the outer ZIP compresses it.
  const extensions = [];
  let remaining = data.length - 15;
  if (remaining >= 0) {
    while (remaining >= 255) {
      extensions.push(255);
      remaining -= 255;
    }
    extensions.push(remaining);
  }
  const block = concat([
    new Uint8Array([Math.min(data.length, 15) << 4, ...extensions]),
    data,
  ]);
  const header = new Uint8Array(12);
  header.set(new TextEncoder().encode("bv41"));
  new DataView(header.buffer).setUint32(4, data.length, true);
  new DataView(header.buffer).setUint32(8, block.length, true);
  return concat([header, block, new TextEncoder().encode("bv4$")]);
}
