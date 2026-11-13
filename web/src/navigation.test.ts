import { expect, it } from "vitest";
import { parseRoute, routeHash } from "./navigation";
it("roundtrips workspace and page links without treating account links as workspace routes", () => {
  const route = { workspaceId: "abc123", pageId: "def456" };
  expect(parseRoute(routeHash(route))).toEqual(route);
  expect(parseRoute("#accept-invite/abc123")).toEqual({});
  expect(parseRoute("#/workspaces/../../private")).toEqual({});
  expect(routeHash({})).toBe("#/");
});
