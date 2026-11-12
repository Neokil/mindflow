export type ImageCrop = { x: number; y: number; width: number; height: number };
export type CanvasElement = {
  id: string;
  type: "image" | "shape" | "text" | "stroke";
  x: number;
  y: number;
  width: number;
  height: number;
  rotation: number;
  groupId?: string;
  shape?: "rectangle" | "circle" | "star";
  fill?: string;
  border?: string;
  borderWidth?: number;
  text?: string;
  fontSize?: number;
  bold?: boolean;
  italic?: boolean;
  align?: "left" | "center" | "right";
  verticalAlign?: "top" | "middle" | "bottom";
  crop?: ImageCrop;
  color?: string;
  assetId?: string;
  points?: number[];
  penWidth?: number;
};
export type CanvasDocument = {
  schemaVersion: 1;
  elements: CanvasElement[];
  settings: { background: "dots" | "plain"; gridSize: number; snap: boolean };
};
export type WorkspacePermission = "owner" | "read" | "comment" | "full";
export type Workspace = {
  role?: WorkspacePermission;
  shared?: boolean;
  id: string;
  name: string;
  revision: number;
  updatedAt: number;
  pageCount: number;
};
export type Page = {
  id: string;
  name: string;
  workspaceId: string;
  position: number;
  revision: number;
  metadataRevision: number;
};
export type Point = { x: number; y: number };
// getRandomValues is available on HTTP origins where randomUUID is unavailable.
export function uid(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  // Set the UUID v4 version and RFC variant bits.
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = Array.from(bytes, (byte) =>
    byte.toString(16).padStart(2, "0"),
  ).join("");
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}
export const clone = <T>(v: T): T => structuredClone(v);
export const emptyDocument = (): CanvasDocument => ({
  schemaVersion: 1,
  elements: [],
  settings: { background: "dots", gridSize: 20, snap: false },
});
export const MAX_ELEMENTS = 10000;
export function duplicateElements(elements: CanvasElement[], offset = 20) {
  const groups = new Map<string, string>();
  return clone(elements).map((e) => {
    if (e.groupId && !groups.has(e.groupId)) groups.set(e.groupId, uid());
    return {
      ...e,
      id: uid(),
      x: e.x + offset,
      y: e.y + offset,
      groupId: e.groupId ? groups.get(e.groupId) : undefined,
    };
  });
}

export function validClipboardElements(
  value: unknown,
): value is CanvasElement[] {
  if (!Array.isArray(value) || value.length > MAX_ELEMENTS) return false;
  return value.every((e) => {
    if (
      !e ||
      typeof e !== "object" ||
      !["image", "shape", "text", "stroke"].includes(e.type)
    )
      return false;
    for (const key of ["x", "y", "width", "height", "rotation"])
      if (typeof e[key] !== "number" || !Number.isFinite(e[key])) return false;
    for (const key of [
      "id",
      "groupId",
      "shape",
      "fill",
      "border",
      "text",
      "align",
      "verticalAlign",
      "color",
      "assetId",
    ])
      if (e[key] !== undefined && typeof e[key] !== "string") return false;
    for (const key of ["borderWidth", "fontSize", "penWidth"])
      if (
        e[key] !== undefined &&
        (typeof e[key] !== "number" || !Number.isFinite(e[key]))
      )
        return false;
    for (const key of ["bold", "italic"])
      if (e[key] !== undefined && typeof e[key] !== "boolean") return false;
    if (
      e.type === "stroke" &&
      (!Array.isArray(e.points) ||
        e.points.length > 40000 ||
        e.points.some(
          (p: unknown) => typeof p !== "number" || !Number.isFinite(p),
        ))
    )
      return false;
    if (
      e.verticalAlign !== undefined &&
      !["top", "middle", "bottom"].includes(e.verticalAlign)
    )
      return false;
    if (e.crop !== undefined && (e.type !== "image" || !validImageCrop(e.crop)))
      return false;
    return e.width > 0 && e.height > 0;
  });
}

export function validImageCrop(value: unknown): value is ImageCrop {
  if (!value || typeof value !== "object") return false;
  const crop = value as ImageCrop;
  return (
    [crop.x, crop.y, crop.width, crop.height].every(Number.isFinite) &&
    crop.x >= 0 &&
    crop.y >= 0 &&
    crop.width > 0 &&
    crop.height > 0 &&
    crop.x + crop.width <= 1 + 1e-9 &&
    crop.y + crop.height <= 1 + 1e-9
  );
}
export const wholeFontSize = (value: number) =>
  Math.max(1, Math.min(2000, Math.round(value)));
