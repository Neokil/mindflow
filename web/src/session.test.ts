import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { APIError } from "./api";
import { emptyDocument } from "./model";
import type { Transport } from "./session";
import { PageSession, localImages } from "./session";
let sessions: PageSession[] = [];
beforeEach(() => {
  vi.stubGlobal("navigator", { onLine: true });
});
afterEach(() => {
  sessions.forEach((s) => s.dispose());
  sessions = [];
  vi.unstubAllGlobals();
});
function make(transport: Transport) {
  const s = new PageSession("page", transport);
  s.loaded = true;
  s.editable = true;
  s.epoch = 1;
  sessions.push(s);
  return s;
}
it("does not mark an edit made during an in-flight save as saved", async () => {
  let resolve!: (v: unknown) => void;
  const transport = vi.fn(
    () =>
      new Promise((r) => {
        resolve = r;
      }),
  ) as unknown as Transport;
  const s = make(transport);
  s.edit({
    ...emptyDocument(),
    settings: { background: "plain", gridSize: 20, snap: false },
  });
  const first = s.flush();
  await Promise.resolve();
  await Promise.resolve();
  s.edit({ ...s.doc, settings: { ...s.doc.settings, gridSize: 40 } });
  resolve({ revision: 1 });
  await first;
  expect(s.dirty).toBe(true);
  expect(s.revision).toBe(1);
});
it("retains dirty local work when another editor takes control", async () => {
  const transport = vi.fn(async (_path: string, method?: string) => {
    if (method === "PUT") throw new APIError(409, "lease_lost", "Lost control");
    return { document: emptyDocument(), revision: 2 };
  }) as unknown as Transport;
  const s = make(transport);
  s.edit({
    ...emptyDocument(),
    settings: { background: "plain", gridSize: 33, snap: true },
  });
  await s.flush();
  expect(s.editable).toBe(false);
  expect(s.recovery?.settings.gridSize).toBe(33);
  expect(s.doc.settings.gridSize).toBe(20);
  expect(s.unsaved).toBe(true);
});
it("retries an uncertain save with the same mutation identity", async () => {
  const attempts: unknown[] = [];
  let count = 0;
  const transport = vi.fn(
    async (_path: string, method?: string, body?: unknown) => {
      if (method !== "PUT") return {};
      attempts.push(body);
      if (++count === 1) throw new TypeError("connection lost");
      return { revision: 1 };
    },
  ) as unknown as Transport;
  const s = make(transport);
  s.edit({
    ...emptyDocument(),
    settings: { background: "plain", gridSize: 20, snap: false },
  });
  await s.flush();
  expect(s.dirty).toBe(true);
  await s.flush();
  expect(attempts[0]).toEqual(attempts[1]);
  expect(s.dirty).toBe(false);
});
it("allows a corrected edit to replace a rejected snapshot", async () => {
  let count = 0;
  const bodies: {
    mutationId: string;
    document: ReturnType<typeof emptyDocument>;
  }[] = [];
  const transport = vi.fn(
    async (_path: string, method?: string, body?: unknown) => {
      if (method !== "PUT") return {};
      bodies.push(
        body as {
          mutationId: string;
          document: ReturnType<typeof emptyDocument>;
        },
      );
      if (++count === 1)
        throw new APIError(400, "validation_failed", "Rejected");
      return { revision: 1 };
    },
  ) as unknown as Transport;
  const s = make(transport);
  s.edit({
    ...emptyDocument(),
    settings: { background: "plain", gridSize: 20, snap: false },
  });
  await s.flush();
  s.edit({ ...s.doc, settings: { ...s.doc.settings, gridSize: 40 } });
  await s.flush();
  expect(bodies[1].document.settings.gridSize).toBe(40);
  expect(bodies[1].mutationId).not.toBe(bodies[0].mutationId);
  expect(s.dirty).toBe(false);
});
it("automatically resumes an unmodified page after its offline lease expires", async () => {
  let attempts = 0;
  const epochs: number[] = [];
  const transport = vi.fn(
    async (_path: string, method?: string, body?: unknown) => {
      const command = body as { epoch: number; action?: string } | undefined;
      if (method === "PUT") {
        epochs.push(command!.epoch);
        attempts++;
        if (attempts === 1) throw new TypeError("offline");
        if (attempts === 2) throw new APIError(409, "lease_lost", "Expired");
        return { revision: 1 };
      }
      if (command?.action === "acquire")
        return { editable: true, epoch: 2, revision: 0, expiresAt: 9999999999 };
      return { editable: false, epoch: 1, revision: 0, expiresAt: 0 };
    },
  ) as unknown as Transport;
  const s = make(transport);
  s.edit({
    ...emptyDocument(),
    settings: { background: "plain", gridSize: 20, snap: false },
  });
  await s.flush();
  await s.flush();
  expect(s.editable).toBe(true);
  expect(s.recovery).toBeUndefined();
  await s.flush();
  expect(s.dirty).toBe(false);
  expect(epochs).toEqual([1, 1, 2]);
});

it("uploads pending recovery images into the chosen workspace", async () => {
  const transport = vi.fn(
    async (path: string, _method?: string, body?: unknown) => {
      if (path === "/assets") {
        expect((body as FormData).get("workspaceId")).toBe(
          "recovery-workspace",
        );
        return { id: "recovered-asset" };
      }
      return { id: "recovered-page" };
    },
  ) as unknown as Transport;
  const session = make(transport);
  session.workspaceId = "former-shared-workspace";
  const localID = "local:recovery-upload";
  localImages.set(localID, {
    file: new File(["image"], "recovery.png", { type: "image/png" }),
    url: "blob:test",
    width: 1,
    height: 1,
  });
  try {
    session.recovery = {
      ...emptyDocument(),
      elements: [
        {
          id: "pending-image",
          type: "image",
          assetId: localID,
          x: 0,
          y: 0,
          width: 100,
          height: 100,
          rotation: 0,
        },
      ],
    };
    await expect(session.recover("recovery-workspace")).resolves.toBe(
      "recovered-page",
    );
  } finally {
    localImages.delete(localID);
    localImages.delete("recovered-asset");
  }
});
it("export waits for an in-flight save and refuses an unsaved offline draft", async () => {
  let resolve!: (value: unknown) => void;
  const transport = vi.fn(
    () =>
      new Promise((done) => {
        resolve = done;
      }),
  ) as unknown as Transport;
  const session = make(transport);
  session.edit({
    ...emptyDocument(),
    settings: { ...emptyDocument().settings, gridSize: 33 },
  });
  const saving = session.flush();
  await Promise.resolve();
  await Promise.resolve();
  const exporting = session.savePendingChanges();
  await vi.waitFor(() => expect(resolve).toBeTypeOf("function"));
  resolve({ revision: 1 });
  await saving;
  await exporting;
  expect(session.unsaved).toBe(false);
  session.edit(emptyDocument());
  session.editable = false;
  await expect(session.savePendingChanges()).rejects.toThrow("unsaved changes");
});
