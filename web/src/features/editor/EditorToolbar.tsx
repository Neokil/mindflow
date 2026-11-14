import ShapeTool from "./ShapeTool";
import type { ReactNode } from "react";
import {
  Hand,
  Blocks,
  MessageCircle,
  ImagePlus,
  FileDown,
  MousePointer2,
  Pencil,
  Highlighter,
  Redo2,
  Type,
  Undo2,
} from "lucide-react";
import type { Tool } from "./types";
type Props = {
  status?: ReactNode;
  tool: Tool;
  placing: boolean;
  editable: boolean;
  canComment: boolean;
  canUndo: boolean;
  canRedo: boolean;
  onTool: (tool: Tool) => void;
  onImages: () => void;
  onPDF: () => void;
  onTemplates: () => void;
  onUndo: () => void;
  onRedo: () => void;
};
const tools = [
  { id: "pan", icon: Hand, label: "Pan (H)" },
  { id: "select", icon: MousePointer2, label: "Select (V)" },
  { id: "text", icon: Type, label: "Text (T)" },
  { id: "pen", icon: Pencil, label: "Pen (P)" },
  { id: "highlighter", icon: Highlighter, label: "Highlighter (M)" },
  { id: "comment", icon: MessageCircle, label: "Add comment" },
] as const;
export default function EditorToolbar({
  status,
  tool,
  placing,
  editable,
  canComment,
  canUndo,
  canRedo,
  onTool,
  onImages,
  onPDF,
  onTemplates,
  onUndo,
  onRedo,
}: Props) {
  return (
    <div className="canvas-toolbar" aria-label="Drawing tools">
      <div className="canvas-tools">
        {tools.map(({ id, icon: Icon, label }) => (
          <div className="tool-slot" key={id}>
            {id === "text" && (
              <ShapeTool
                tool={tool}
                placing={placing}
                disabled={!editable}
                onTool={onTool}
              />
            )}
            <button
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
          </div>
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
        <button
          className="tool"
          aria-label="Import PDF"
          title="Import PDF"
          disabled={!editable}
          onClick={onPDF}
        >
          <FileDown size={18} />
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
      {status}
    </div>
  );
}
