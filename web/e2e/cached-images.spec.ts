import { expect, test } from "@playwright/test";
import { activate, login } from "./helpers";
import { emptyDocument } from "../src/model";

test("returning to a page paints cached images while metadata and image requests are stalled", async ({
  page,
}, info) => {
  await login(page);
  const { csrf } = await (await page.request.get("/api/v1/auth/me")).json();
  const headers = { "X-CSRF-Token": csrf };
  const workspace = await (
    await page.request.post("/api/v1/workspaces", {
      headers,
      data: { name: `Cached images ${info.project.name}` },
    })
  ).json();
  const encoded = await page.evaluate(() => {
    const canvas = document.createElement("canvas");
    canvas.width = 200;
    canvas.height = 100;
    const context = canvas.getContext("2d")!;
    context.fillStyle = "#ff00ff";
    context.fillRect(0, 0, 200, 100);
    return canvas.toDataURL("image/png").split(",")[1];
  });
  const asset = await (
    await page.request.post("/api/v1/assets", {
      headers,
      multipart: {
        workspaceId: workspace.id,
        file: {
          name: "cached.png",
          mimeType: "image/png",
          buffer: Buffer.from(encoded, "base64"),
        },
      },
    })
  ).json();
  const clientId = "cached-image-fixture";
  const lease = await (
    await page.request.post(`/api/v1/pages/${workspace.pageId}/lease`, {
      headers,
      data: { clientId, action: "acquire" },
    })
  ).json();
  const saved = await page.request.put(
    `/api/v1/pages/${workspace.pageId}/document`,
    {
      headers,
      data: {
        clientId,
        epoch: lease.epoch,
        baseRevision: 0,
        mutationId: "cached-image-document",
        document: {
          ...emptyDocument(),
          elements: [
            {
              id: "cached-image",
              type: "image",
              assetId: asset.id,
              x: 0,
              y: 0,
              width: 400,
              height: 200,
              rotation: 0,
            },
          ],
        },
      },
    },
  );
  expect(saved.ok()).toBe(true);
  await page.request.post(`/api/v1/pages/${workspace.pageId}/lease`, {
    headers,
    data: { clientId, epoch: lease.epoch, action: "release" },
  });
  await page.request.post(`/api/v1/workspaces/${workspace.id}/pages`, {
    headers,
    data: { name: "Other page" },
  });
  await page.goto(`/#/workspaces/${workspace.id}/pages/${workspace.pageId}`);
  const color = () =>
    page
      .locator(".konvajs-content canvas")
      .first()
      .evaluate((node) => {
        const canvas = node as HTMLCanvasElement;
        return Array.from(
          canvas
            .getContext("2d")!
            .getImageData(
              Math.floor(canvas.width / 2),
              Math.floor(canvas.height / 2),
              1,
              1,
            ).data,
        ).slice(0, 3);
      });
  await expect.poll(color).toEqual([255, 0, 255]);
  await activate(
    page,
    page.getByRole("tab", { name: "Other page", exact: true }),
  );
  await expect(page.getByTestId("element-count")).toHaveText("0 elements");
  // Expire the metadata TTL while retaining the decoded image. A slow API must
  // not make a previously painted image disappear on the return trip.
  await page.waitForTimeout(5100);
  let release!: () => void;
  const stalled = new Promise<void>((resolve) => {
    release = resolve;
  });
  await page.route("**/api/v1/assets/**", async (route) => {
    await stalled;
    await route.continue().catch(() => {});
  });
  try {
    await activate(
      page,
      page.getByRole("tab", { name: "Untitled", exact: true }),
    );
    await expect(page.getByTestId("element-count")).toHaveText("1 element");
    await expect.poll(color, { timeout: 1500 }).toEqual([255, 0, 255]);
  } finally {
    release();
  }
});
