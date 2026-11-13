import {
  Circle,
  Hand,
  Blocks,
  MessageCircle,
  ImagePlus,
  MousePointer2,
  Pencil,
  Redo2,
  Square,
  Star,
  Type,
  Undo2,
} from "lucide-react";
import type { Tool } from "./types";
type Props = {
  tool: Tool;
  placing: boolean;
  editable: boolean;
  canComment: boolean;
  canUndo: boolean;
  canRedo: boolean;
  onTool: (tool: Tool) => void;
  onImages: () => void;
  onTemplates: () => void;
  onUndo: () => void;
  onRedo: () => void;
};
const tools = [
  { id: "pan", icon: Hand, label: "Pan (H)" },
  { id: "select", icon: MousePointer2, label: "Select (V)" },
  { id: "rectangle", icon: Square, label: "Rectangle" },
  { id: "circle", icon: Circle, label: "Circle" },
  { id: "star", icon: Star, label: "Star" },
  { id: "text", icon: Type, label: "Text (T)" },
  { id: "pen", icon: Pencil, label: "Pen (P)" },
  { id: "comment", icon: MessageCircle, label: "Add comment" },
] as const;
export default function EditorToolbar({
  tool,
  placing,
  editable,
  canComment,
  canUndo,
  canRedo,
  onTool,
  onImages,
  onTemplates,
  onUndo,
  onRedo,
}: Props) {
  return (
    <div className="canvas-toolbar" aria-label="Drawing tools">
      {tools.map(({ id, icon: Icon, label }) => (
        <button
          key={id}
          className={`tool ${tool === id && !placing ? "active" : ""}`}
          title={label}
          aria-label={label}
          aria-pressed={tool === id}
          disabled={
            id === "comment"
              ? !canComment
              : !editable && id !== "select" && id !== "pan"
          }
          onClick={() => onTool(id)}
        >
          <Icon size={18} strokeWidth={1.7} />
        </button>
      ))}
      <button
        className="tool"
        title="Workspace templates"
        aria-label="Workspace templates"
        onClick={onTemplates}
      >
        <Blocks size={18} />
      </button>
      <span className="toolbar-divider" />
      <button
        className="tool"
        title="Add images"
        aria-label="Add images"
        disabled={!editable}
        onClick={onImages}
      >
        <ImagePlus size={18} />
      </button>
      <span className="toolbar-divider" />
      <button
        className="tool"
        title="Undo"
        aria-label="Undo"
        disabled={!editable || !canUndo}
        onClick={onUndo}
      >
        <Undo2 size={17} />
      </button>
      <button
        className="tool"
        title="Redo"
        aria-label="Redo"
        disabled={!editable || !canRedo}
        onClick={onRedo}
      >
        <Redo2 size={17} />
      </button>
    </div>
  );
}
