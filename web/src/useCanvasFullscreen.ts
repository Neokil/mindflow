import { useCallback, useEffect, useRef, useState } from "react";

export type CanvasFullscreen = ReturnType<typeof useCanvasFullscreen>;

// Fullscreen must be requested directly from the user's click or tap, not an effect.
// The document stays fullscreen so authentication and editor-control dialogs remain visible.
export function useCanvasFullscreen(editorOpen: boolean) {
  const [active, setActive] = useState(false);
  const available =
    !!document.fullscreenEnabled &&
    !!document.documentElement.requestFullscreen;
  const wanted = useRef(false);
  const owned = useRef(false);
  const pending = useRef(false);
  const exit = useCallback(async () => {
    wanted.current = false;
    if (document.fullscreenElement === document.documentElement) {
      try {
        await document.exitFullscreen();
        owned.current = false;
      } catch {
        // The browser may already be leaving fullscreen.
      }
    }
  }, []);
  const enter = useCallback(async () => {
    if (
      !document.fullscreenEnabled ||
      !document.documentElement.requestFullscreen ||
      document.fullscreenElement ||
      pending.current
    )
      return;
    wanted.current = true;
    pending.current = true;
    try {
      await document.documentElement.requestFullscreen();
      owned.current = document.fullscreenElement === document.documentElement;
      if (!wanted.current) await exit();
    } catch {
      wanted.current = false;
      // Leave the canvas in its normal layout if the browser denies fullscreen.
    } finally {
      pending.current = false;
    }
  }, [exit]);
  const toggle = useCallback(async () => {
    if (
      wanted.current ||
      document.fullscreenElement === document.documentElement
    )
      await exit();
    else await enter();
  }, [enter, exit]);
  useEffect(() => {
    const changed = () => {
      const fullscreen =
        document.fullscreenElement === document.documentElement;
      setActive(fullscreen);
      if (!fullscreen) {
        owned.current = false;
        wanted.current = false;
      }
    };
    document.addEventListener("fullscreenchange", changed);
    return () => document.removeEventListener("fullscreenchange", changed);
  }, []);
  useEffect(() => {
    const leave = () => {
      wanted.current = false;
      if (owned.current) void exit();
    };
    if (!editorOpen) leave();
    return leave;
  }, [editorOpen, exit]);
  return { active, available, toggle };
}
