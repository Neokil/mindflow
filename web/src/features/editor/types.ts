export type Tool =
  | "select"
  | "pan"
  | "rectangle"
  | "circle"
  | "star"
  | "text"
  | "pen"
  | "highlighter"
  | "comment";
export type Camera = { x: number; y: number; zoom: number };
