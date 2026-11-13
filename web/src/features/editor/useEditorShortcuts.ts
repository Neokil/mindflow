import type { Dispatch, SetStateAction } from "react";
import { useEffect } from "react";
import { worldPoint } from "../../geometry";
import type { CanvasElement, Point } from "../../model";
import { duplicateElements, uid, validClipboardElements } from "../../model";
import { PageSession } from "../../session";
import type { Camera, Tool } from "./types";
type Props = {
  session: PageSession;
  selected: CanvasElement[];
  camera: Camera;
  textColor: string;
  filesAdded: (files: File[]) => Promise<void>;
  pointer: () => Point;
  editElements: (elements: CanvasElement[]) => void;
  setSelection: Dispatch<SetStateAction<string[]>>;
  setTool: Dispatch<SetStateAction<Tool>>;
  setError: Dispatch<SetStateAction<string>>;
  duplicate: () => void;
  group: () => void;
  ungroup: () => void;
  remove: () => void;
  cancel: () => void;
};
export default function useEditorShortcuts({
  session,
  selected,
  camera,
  textColor,
  filesAdded,
  pointer,
  editElements,
  setSelection,
  setTool,
  setError,
  duplicate,
  group,
  ungroup,
  remove,
  cancel,
}: Props) {
  useEffect(() => {
    const onPaste = (event: ClipboardEvent) => {
      if (
        event.target instanceof Element &&
        event.target.closest("input,textarea,dialog,[contenteditable=true]")
      )
        return;
      if (!session.editable) return;
      const directFiles = [...(event.clipboardData?.files || [])];
      const files = directFiles.length
        ? directFiles
        : [...(event.clipboardData?.items || [])]
            .filter((i) => i.kind === "file")
            .map((i) => i.getAsFile())
            .filter((f): f is File => !!f);
      if (files.length) {
        event.preventDefault();
        void filesAdded(files);
        return;
      }
      const raw =
        event.clipboardData?.getData("application/x-mindflow") ||
        event.clipboardData?.getData("text/plain");
      if (raw?.startsWith('{"mindflow":')) {
        try {
          const parsed = JSON.parse(raw);
          if (
            parsed.mindflow === 1 &&
            validClipboardElements(parsed.elements)
          ) {
            const copied = duplicateElements(parsed.elements);
            editElements([...session.doc.elements, ...copied]);
            setSelection(copied.map((e) => e.id));
            event.preventDefault();
          }
        } catch {
          setError("Clipboard data could not be pasted.");
        }
      } else if (raw) {
        const p = worldPoint(pointer(), camera);
        const text: CanvasElement = {
          id: uid(),
          type: "text",
          ...p,
          width: 260,
          height: Math.max(100, raw.split("\n").length * 32),
          rotation: 0,
          text: raw.slice(0, 100000),
          fontSize: 24,
          color: textColor,
          align: "center",
          verticalAlign: "middle",
        };
        editElements([...session.doc.elements, text]);
        setSelection([text.id]);
        event.preventDefault();
      }
    };
    const onCopy = (event: ClipboardEvent) => {
      if (
        (event.target instanceof Element &&
          event.target.closest(
            "input,textarea,dialog,[contenteditable=true]",
          )) ||
        !selected.length
      )
        return;
      event.preventDefault();
      const value = JSON.stringify({ mindflow: 1, elements: selected });
      event.clipboardData?.setData("application/x-mindflow", value);
      event.clipboardData?.setData("text/plain", value);
    };
    const onKey = (e: KeyboardEvent) => {
      if (
        e.target instanceof Element &&
        e.target.closest("input,textarea,select,[contenteditable=true],dialog")
      )
        return;
      const command = e.metaKey || e.ctrlKey;
      if (e.key === "Escape") {
        cancel();
        return;
      }
      if (command && e.key.toLowerCase() === "a") {
        e.preventDefault();
        setSelection(session.doc.elements.map((e) => e.id));
        return;
      }
      if (!command && e.key.toLowerCase() === "h") {
        setTool("pan");
        return;
      }
      if (!session.editable) return;
      if (command && e.key.toLowerCase() === "z") {
        e.preventDefault();
        if (e.shiftKey) session.redo();
        else session.undo();
      } else if (command && e.key.toLowerCase() === "d") {
        e.preventDefault();
        duplicate();
      } else if (command && e.key.toLowerCase() === "g") {
        e.preventDefault();
        if (e.shiftKey) ungroup();
        else group();
      } else if (e.key === "Delete" || e.key === "Backspace") {
        e.preventDefault();
        remove();
      } else if (!command && e.key.toLowerCase() === "v") setTool("select");
      else if (!command && e.key.toLowerCase() === "p") setTool("pen");
      else if (!command && e.key.toLowerCase() === "m") setTool("highlighter");
      else if (!command && e.key.toLowerCase() === "t") setTool("text");
    };
    window.addEventListener("paste", onPaste);
    window.addEventListener("copy", onCopy);
    window.addEventListener("keydown", onKey);
    return () => {
      window.removeEventListener("paste", onPaste);
      window.removeEventListener("copy", onCopy);
      window.removeEventListener("keydown", onKey);
    };
  });
}
