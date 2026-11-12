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
export type Workspace = {
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
export const uid = () => crypto.randomUUID();
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
    return e.width > 0 && e.height > 0;
  });
}
