import { useCallback, useEffect, useRef } from "react";

// Fullscreen must be requested directly from the user's click or tap, not an effect.
export function useCanvasFullscreen(editorOpen: boolean) {
  const wanted = useRef(false);
  const owned = useRef(false);
  const pending = useRef(false);
  const exit = useCallback(async () => {
    wanted.current = false;
    if (
      owned.current &&
      document.fullscreenElement === document.documentElement
    ) {
      owned.current = false;
      try {
        await document.exitFullscreen();
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
      owned.current = true;
      // Restoring navigation or leaving the editor can race the request.
      if (!wanted.current) await exit();
    } catch {
      // Collapsing navigation still works when fullscreen is unavailable/denied.
    } finally {
      pending.current = false;
    }
  }, [exit]);
  useEffect(() => {
    const changed = () => {
      if (document.fullscreenElement !== document.documentElement)
        owned.current = false;
    };
    document.addEventListener("fullscreenchange", changed);
    return () => document.removeEventListener("fullscreenchange", changed);
  }, []);
  useEffect(() => {
    if (!editorOpen) void exit();
    return () => {
      void exit();
    };
  }, [editorOpen, exit]);
  return { enter, exit };
}
