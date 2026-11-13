import { afterEach, expect, it, vi } from "vitest";
import { api, apiFile, APIError, setCSRF } from "./api";
afterEach(() => {
  vi.unstubAllGlobals();
  setCSRF("");
});
it("binary export uses the same authenticated CSRF transport as JSON", async () => {
  const blob = new Blob(["archive"]),
    fetch = vi.fn(async () => new Response(blob));
  vi.stubGlobal("fetch", fetch);
  setCSRF("token");
  expect(
    await (await apiFile("/workspaces/id/export", { pageIds: [] })).text(),
  ).toBe("archive");
  expect(fetch).toHaveBeenCalledWith(
    "/api/v1/workspaces/id/export",
    expect.objectContaining({
      method: "POST",
      credentials: "same-origin",
      headers: expect.objectContaining({ "X-CSRF-Token": "token" }),
    }),
  );
});
it("binary errors retain the server message and JSON calls accept empty acknowledgements", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(
      async () =>
        new Response(
          JSON.stringify({ code: "forbidden", message: "No access" }),
          { status: 403 },
        ),
    ),
  );
  await expect(apiFile("/export", {})).rejects.toEqual(
    new APIError(403, "forbidden", "No access"),
  );
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => new Response(null, { status: 204 })),
  );
  expect(await api("/resource", "DELETE")).toEqual({});
});
