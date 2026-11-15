import { expect, test, type Locator } from "@playwright/test";
import { login, activate } from "./helpers";
import { emptyDocument } from "../src/model";
import type {
  CanvasDocument,
  Page as CanvasPage,
  Workspace,
} from "../src/model";

async function centerPixel(image: Locator) {
  if (!(await image.count())) return [];
  return image
    .evaluate((node) => {
      const image = node as HTMLImageElement;
      if (!image.complete || !image.naturalWidth) return [];
      const canvas = document.createElement("canvas");
      canvas.width = image.naturalWidth;
      canvas.height = image.naturalHeight;
      const context = canvas.getContext("2d")!;
      context.drawImage(image, 0, 0);
      return Array.from(
        context.getImageData(
          Math.floor(canvas.width / 2),
          Math.floor(canvas.height / 2),
          1,
          1,
        ).data,
      ).slice(0, 3);
    })
    .catch(() => []);
}
function coloredPage(color: string): CanvasDocument {
  return {
    ...emptyDocument(),
    settings: {
      ...emptyDocument().settings,
      background: "plain",
      backgroundColor: "#ffffff",
    },
    elements: [
      {
        id: "cover-shape",
        type: "shape",
        shape: "rectangle",
        x: 0,
        y: 0,
        width: 200,
        height: 120,
        rotation: 0,
        fill: color,
        border: color,
        borderWidth: 0,
      },
    ],
  };
}
test("workspace thumbnails follow the first page, saved edits, ordering, and deletion", async ({
  page,
}, info) => {
  await login(page);
  const me = await (await page.request.get("/api/v1/auth/me")).json();
  const headers = { "X-CSRF-Token": me.csrf };
  const created = await page.request.post("/api/v1/workspaces", {
    headers,
    data: { name: `Preview ${info.project.name}` },
  });
  expect(created.ok()).toBe(true);
  const workspace = (await created.json()) as { id: string; pageId: string };
  const card = page.locator(".workspace-card").filter({
    has: page.getByRole("button", {
      name: `Open Preview ${info.project.name}`,
      exact: true,
    }),
  });
  const image = card.getByRole("img", {
    name: `Preview of the first page of Preview ${info.project.name}`,
    exact: true,
  });
  await expect(
    card.getByText("Preview available after opening the workspace"),
  ).toBeVisible();
  async function renderSavedPreview() {
    const stored = page.waitForResponse(
      (response) =>
        response.url().endsWith(`/workspaces/${workspace.id}/preview`) &&
        response.request().method() === "POST" &&
        response.ok(),
    );
    await page.goto(`/#/workspaces/${workspace.id}/pages/${workspace.pageId}`);
    await expect(page.getByTestId("canvas")).toBeVisible();
    await stored;
    await activate(
      page,
      page.getByRole("button", { name: "Back to workspaces", exact: true }),
    );
    await expect(card).toBeVisible();
  }
  let dashboardDocuments = 0;
  page.on("request", (request) => {
    if (
      new URL(page.url()).pathname === "/" &&
      !page.url().includes("#/workspaces/") &&
      request.url().endsWith("/document")
    )
      dashboardDocuments++;
  });
  async function save(id: string, color: string, revision = 0) {
    const leaseResponse = await page.request.post(`/api/v1/pages/${id}/lease`, {
      headers,
      data: { clientId: "thumbnail-test-client", action: "acquire" },
    });
    expect(leaseResponse.ok()).toBe(true);
    const lease = await leaseResponse.json();
    expect(lease.editable).toBe(true);
    const response = await page.request.put(`/api/v1/pages/${id}/document`, {
      headers,
      data: {
        clientId: "thumbnail-test-client",
        epoch: lease.epoch,
        baseRevision: revision,
        mutationId: `thumbnail-${id}-${revision}`,
        document: coloredPage(color),
      },
    });
    expect(response.ok(), await response.text()).toBe(true);
  }
  await save(workspace.pageId, "#ff0000");
  await renderSavedPreview();
  await expect.poll(() => centerPixel(image)).toEqual([255, 0, 0]);
  await expect(image).toHaveJSProperty("naturalWidth", 512);
  const displayed = (await image.boundingBox())!;
  const frame = (await card.locator(".workspace-preview-sheet").boundingBox())!;
  expect(displayed.width).toBeLessThanOrEqual(frame.width);
  expect(displayed.height).toBeLessThanOrEqual(frame.height);

  expect(await image.getAttribute("src")).toContain(
    `/api/v1/workspaces/${workspace.id}/previews/`,
  );
  const secondResponse = await page.request.post(
    `/api/v1/workspaces/${workspace.id}/pages`,
    { headers, data: { name: "Second page" } },
  );
  expect(secondResponse.ok()).toBe(true);
  const second = (await secondResponse.json()) as { id: string };
  await save(second.id, "#0000ff");
  await expect(card.locator(".workspace-page-count")).toHaveText("2 pages");
  await expect.poll(() => centerPixel(image)).toEqual([255, 0, 0]);
  const all = (await (
    await page.request.get("/api/v1/workspaces")
  ).json()) as Workspace[];
  const summary = all.find((w) => w.id === workspace.id)!;
  const reorder = await page.request.put(
    `/api/v1/workspaces/${workspace.id}/page-order`,
    {
      headers,
      data: { ids: [second.id, workspace.pageId], revision: summary.revision },
    },
  );
  expect(reorder.ok()).toBe(true);
  await renderSavedPreview();
  await expect.poll(() => centerPixel(image)).toEqual([0, 0, 255]);
  await save(second.id, "#00ff00", 1);
  await renderSavedPreview();
  await expect.poll(() => centerPixel(image)).toEqual([0, 255, 0]);
  await page.screenshot({
    path: `../.cache/e2e/workspace-previews-${info.project.name}.png`,
  });
  const pages = (await (
    await page.request.get(`/api/v1/workspaces/${workspace.id}/pages`)
  ).json()) as CanvasPage[];
  for (const id of [second.id, workspace.pageId]) {
    const removed = await page.request.delete(`/api/v1/pages/${id}`, {
      headers,
      data: {
        revision: pages.find((p) => p.id === id)!.metadataRevision,
        confirm: true,
      },
    });
    expect(removed.ok()).toBe(true);
    if (id === second.id) {
      await renderSavedPreview();
      await expect.poll(() => centerPixel(image)).toEqual([255, 0, 0]);
      await expect(card.locator(".workspace-page-count")).toHaveCount(0);
    }
  }
  await expect(card.getByText("No pages", { exact: true })).toBeVisible();
  await expect(image).toHaveCount(0);
  expect(dashboardDocuments).toBe(0);
  expect(await card.locator(".preview-stack").count()).toBe(0);
});
