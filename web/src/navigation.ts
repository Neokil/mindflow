import { useCallback, useEffect, useRef, useState } from "react";
export type WorkspaceRoute = { workspaceId?: string; pageId?: string };
export function parseRoute(hash: string): WorkspaceRoute {
  const match = /^#\/workspaces\/([a-f0-9]+)(?:\/pages\/([a-f0-9]+))?$/.exec(
    hash,
  );
  return match ? { workspaceId: match[1], pageId: match[2] } : {};
}
export function routeHash(route: WorkspaceRoute) {
  return route.workspaceId
    ? `#/workspaces/${route.workspaceId}${route.pageId ? `/pages/${route.pageId}` : ""}`
    : "#/";
}
export function useWorkspaceRoute(dirty: { current: boolean }) {
  const [route, setRoute] = useState(() => parseRoute(location.hash));
  const previous = useRef(route);
  const navigate = useCallback((next: WorkspaceRoute, replace = false) => {
    const hash = routeHash(next);
    if (location.hash !== hash)
      history[replace ? "replaceState" : "pushState"](null, "", hash);
    previous.current = next;
    setRoute(next);
  }, []);
  useEffect(() => {
    const change = () => {
      const next = parseRoute(location.hash);
      if (
        previous.current.workspaceId !== next.workspaceId &&
        dirty.current &&
        !confirm(
          "There are unsaved changes. Leave this workspace and discard them?",
        )
      ) {
        history.replaceState(null, "", routeHash(previous.current));
        return;
      }
      previous.current = next;
      setRoute(next);
    };
    window.addEventListener("hashchange", change);
    window.addEventListener("popstate", change);
    return () => {
      window.removeEventListener("hashchange", change);
      window.removeEventListener("popstate", change);
    };
  }, [dirty]);
  return { route, navigate };
}
