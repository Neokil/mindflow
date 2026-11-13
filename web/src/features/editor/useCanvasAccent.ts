import { useSyncExternalStore } from "react";

// Konva needs a resolved color: canvas paint styles cannot use CSS variables.
function accent() {
  return getComputedStyle(document.documentElement)
    .getPropertyValue("--accent")
    .trim();
}
function subscribe(changed: () => void) {
  const observer = new MutationObserver(changed);
  observer.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ["data-theme"],
  });
  return () => observer.disconnect();
}
export default function useCanvasAccent() {
  return useSyncExternalStore(subscribe, accent);
}
