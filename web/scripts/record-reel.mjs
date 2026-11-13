/* global document */
// Isolated promotional fixture: never point this recorder at a deployed instance.
import { chromium, expect } from "@playwright/test";
import { spawn } from "node:child_process";
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { DatabaseSync } from "node:sqlite";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const output = join(root, ".cache/reel");
const frames = join(output, "frames");
const data = mkdtempSync(join(tmpdir(), "mindflow-reel-"));
const origin = "http://127.0.0.1:8091";
mkdirSync(frames, { recursive: true });
let bootstrap = "",
  logs = "",
  browser;
const server = spawn(join(output, "mindflow"), [], {
  env: {
    ...process.env,
    MINDFLOW_DEV: "1",
    MINDFLOW_ADDR: "127.0.0.1:8091",
    MINDFLOW_DATA: data,
  },
  stdio: ["ignore", "ignore", "pipe"],
});
const stopped = new Promise((resolve) => {
  server.once("exit", resolve);
  server.once("error", resolve);
});
server.stderr.on("data", (chunk) => {
  logs += chunk;
  bootstrap =
    logs.match(/credential \(operator only\): ([a-f0-9]+)/)?.[1] || bootstrap;
});
const delay = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
const interrupt = () => {
  server.kill("SIGTERM");
  void browser?.close().catch(() => {});
};
process.once("SIGINT", interrupt);
process.once("SIGTERM", interrupt);
try {
  await expect
    .poll(
      async () => {
        try {
          return !!bootstrap && (await fetch(`${origin}/health/ready`)).ok;
        } catch {
          return false;
        }
      },
      { timeout: 15000 },
    )
    .toBeTruthy();
  browser = await chromium.launch();
  const page = await browser.newPage({
    viewport: { width: 1000, height: 1000 },
    colorScheme: "dark",
    deviceScaleFactor: 1,
  });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto(origin);
  for (const [label, value] of [
    ["Setup credential", bootstrap],
    ["Email address", "alex@demo.invalid"],
    ["Password", "local reel fixture password"],
    ["Public application URL", origin],
    ["SMTP host", "127.0.0.1"],
    ["Port", "9"],
    ["Sender email", "mindflow@demo.invalid"],
  ])
    await page.getByLabel(label, { exact: true }).fill(value);
  await page
    .getByRole("button", { name: "Create mindflow", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Sign in", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Workspaces", exact: true }),
  ).toBeVisible();
  const me = await (await page.request.get(`${origin}/api/v1/auth/me`)).json();
  const request = async (method, path, body) => {
    const response = await page.request.fetch(`${origin}/api/v1${path}`, {
      method,
      headers: { "X-CSRF-Token": me.csrf },
      data: body,
    });
    if (!response.ok())
      throw new Error(
        `Fixture ${method} ${path}: ${response.status()} ${await response.text()}`,
      );
    return response.json();
  };
  await request("POST", "/admin/users", {
    email: "sam@demo.invalid",
    role: "user",
  });
  const db = new DatabaseSync(join(data, "mindflow.sqlite"), {
    readOnly: true,
  });
  let token;
  try {
    token = JSON.parse(
      db
        .prepare(
          "SELECT payload FROM mail_jobs WHERE user_id=(SELECT id FROM users WHERE email=?)",
        )
        .get("sam@demo.invalid").payload,
    ).body.match(/#accept-invite\/([a-f0-9]+)/)[1];
  } finally {
    db.close();
  }
  await request("POST", "/auth/accept-invite", {
    token,
    password: "another local fixture password",
  });
  const workspace = await request("POST", "/workspaces", {
    name: "Room for what's next",
  });
  const pages = await request("GET", `/workspaces/${workspace.id}/pages`);
  const pageId = pages[0].id;
  const clientId = "reel-fixture";
  const lease = await request("POST", `/pages/${pageId}/lease`, {
    clientId,
    action: "acquire",
  });
  const snapshot = await request("GET", `/pages/${pageId}/document`);
  const base = (id, type, x, y, width, height, extra) => ({
    id,
    type,
    x,
    y,
    width,
    height,
    rotation: 0,
    ...extra,
  });
  const text = (id, x, y, w, h, value, fontSize, extra = {}) =>
    base(id, "text", x, y, w, h, {
      text: value,
      fontSize,
      color: "#ececf2",
      align: "left",
      verticalAlign: "middle",
      ...extra,
    });
  snapshot.document.elements = [
    text("heading", 450, 130, 780, 150, "Space for ideas.", 62, {
      bold: true,
      color: "#d9a441",
    }),
    text(
      "subheading",
      430,
      235,
      740,
      50,
      "A weekend escape. A bigger idea. Start here.",
      25,
    ),
    base("note", "shape", 710, 440, 310, 270, {
      shape: "rectangle",
      fill: "#1b5e3b",
      border: "#4caf50",
      borderWidth: 2,
    }),
    text("note-title", 710, 357, 260, 55, "Make it happen", 29, {
      bold: true,
      color: "#a4d65e",
    }),
    text(
      "checklist",
      710,
      470,
      260,
      145,
      "Find inspiration\nSketch a route\nInvite a friend",
      24,
      { listStyle: "check", checked: [] },
    ),
    base("star", "shape", 810, 120, 70, 70, {
      shape: "star",
      fill: "#d9a441",
      border: "#d9a441",
      borderWidth: 1,
    }),
    text("footer", 450, 710, 780, 90, "Room to think. Space to create.", 30, {
      color: "#d9a441",
    }),
  ];
  await request("PUT", `/pages/${pageId}/document`, {
    clientId,
    epoch: lease.epoch,
    baseRevision: snapshot.revision,
    mutationId: "reel-seed",
    document: snapshot.document,
  });
  await request("POST", `/pages/${pageId}/lease`, {
    clientId,
    epoch: lease.epoch,
    action: "release",
  });
  await page.goto(`${origin}/#/workspaces/${workspace.id}/pages/${pageId}`);
  await expect(page.getByTestId("canvas")).toBeVisible();
  await expect(page.getByTestId("element-count")).toHaveText("7 elements");
  await page
    .getByRole("button", { name: "Hide app header and pages", exact: true })
    .click();
  await page.getByRole("button", { name: "Fit content", exact: true }).click();
  await delay(400);
  // Native app geometry, matching the document bounds and Fit content padding.
  const zoom = 800 / 805;
  const point = (x, y) => ({
    x: 500 + (x - 462.5) * zoom,
    y: 500 + (y - 405) * zoom,
  });
  const encoded = await page.evaluate(() => {
    const canvas = document.createElement("canvas");
    canvas.width = 900;
    canvas.height = 540;
    const ctx = canvas.getContext("2d");
    const gradient = ctx.createLinearGradient(0, 0, 0, 540);
    gradient.addColorStop(0, "#d6e9c1");
    gradient.addColorStop(1, "#f7efd8");
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, 900, 540);
    ctx.fillStyle = "#d9a441";
    ctx.beginPath();
    ctx.arc(690, 135, 65, 0, Math.PI * 2);
    ctx.fill();
    for (const [color, points] of [
      [
        "#a4d65e",
        [
          [0, 390],
          [230, 150],
          [480, 410],
          [690, 240],
          [900, 410],
        ],
      ],
      [
        "#4caf50",
        [
          [0, 460],
          [230, 300],
          [490, 470],
          [700, 330],
          [900, 440],
        ],
      ],
      [
        "#1b5e3b",
        [
          [0, 500],
          [200, 460],
          [440, 510],
          [670, 420],
          [900, 480],
        ],
      ],
    ]) {
      ctx.fillStyle = color;
      ctx.beginPath();
      ctx.moveTo(0, 540);
      for (const [x, y] of points) ctx.lineTo(x, y);
      ctx.lineTo(900, 540);
      ctx.fill();
    }
    return canvas.toDataURL("image/png").split(",")[1];
  });
  await page.addStyleTag({
    content:
      "#reel-pointer{position:fixed;width:14px;height:14px;border:2px solid #f8f7ef;background:#a4d65eaa;border-radius:50%;z-index:10000;pointer-events:none;transform:translate(-50%,-50%);box-shadow:0 2px 10px #0006}",
  });
  await page.evaluate(() => {
    const pointer = document.createElement("div");
    pointer.id = "reel-pointer";
    pointer.style.left = "-50px";
    document.body.append(pointer);
    document.addEventListener("mousemove", (e) => {
      pointer.style.left = `${e.clientX}px`;
      pointer.style.top = `${e.clientY}px`;
    });
  });
  const clips = [];
  const capture = async (name, action) => {
    console.log(`Recording ${name}`);
    let recording = true;
    const entries = [];
    const started = performance.now();
    const framesTask = (async () => {
      while (recording) {
        const file = `${name}-${String(entries.length).padStart(4, "0")}.png`;
        const time = performance.now() - started;
        await page.screenshot({ path: join(frames, file) });
        entries.push({ file, time });
        await delay(90);
      }
    })();
    void framesTask.catch(() => {});
    try {
      await delay(400);
      await action();
      await delay(900);
    } finally {
      recording = false;
      await framesTask;
    }
    clips.push({
      name,
      frames: entries,
      duration: performance.now() - started,
    });
  };
  await capture("collect", async () => {
    await page.getByRole("button", { name: "Add images", exact: true }).click();
    await page.locator('input[type="file"]').setInputFiles({
      name: "weekend-inspiration.png",
      mimeType: "image/png",
      buffer: Buffer.from(encoded, "base64"),
    });
    await expect(
      page.getByText("Drag to size image", { exact: false }),
    ).toBeVisible();
    const a = point(60, 300),
      b = point(490, 560);
    await page.mouse.move(a.x, a.y);
    await page.mouse.down();
    for (let i = 1; i <= 28; i++) {
      await page.mouse.move(
        a.x + ((b.x - a.x) * i) / 28,
        a.y + ((b.y - a.y) * i) / 28,
      );
      await delay(35);
    }
    await page.mouse.up();
    await expect(page.getByTestId("element-count")).toHaveText("8 elements");
    await page.getByRole("button", { name: "Select (V)", exact: true }).click();
    await page.mouse.click(point(525, 625).x, point(525, 625).y);
    await expect(page.getByTestId("save-status")).toHaveText("Saved");
  });
  await capture("draw", async () => {
    await page.getByRole("button", { name: "Pen (P)", exact: true }).click();
    await page.getByLabel("Pen color", { exact: true }).fill("#a4d65e");
    const a = point(580, 389);
    await page.mouse.move(a.x, a.y);
    await page.mouse.down();
    for (let i = 1; i <= 34; i++) {
      const p = point(580 + i * 7, 389 + Math.sin(i / 3) * 5);
      await page.mouse.move(p.x, p.y);
      await delay(35);
    }
    await delay(800);
    await page.mouse.up();
    await page
      .getByRole("button", { name: "Highlighter (M)", exact: true })
      .click();
    const from = point(65, 726);
    await page.mouse.move(from.x, from.y);
    await page.mouse.down();
    for (let i = 1; i <= 28; i++) {
      const p = point(65 + i * 22, 726 + Math.sin(i / 5) * 3);
      await page.mouse.move(p.x, p.y);
      await delay(25);
    }
    await page.mouse.up();
    await page.getByRole("button", { name: "Select (V)", exact: true }).click();
    await expect(page.getByTestId("save-status")).toHaveText("Saved");
  });
  await capture("share", async () => {
    await page
      .getByRole("button", { name: "Show app header and pages", exact: true })
      .click();
    await page
      .getByRole("button", { name: "Workspace sharing", exact: true })
      .click();
    await page
      .getByLabel("Share with email", { exact: true })
      .fill("sam@demo.invalid");
    await page
      .getByLabel("Sharing access", { exact: true })
      .selectOption("comment");
    await page.getByRole("button", { name: "Add user", exact: true }).click();
    await expect(page.getByLabel("Access for sam@demo.invalid")).toHaveValue(
      "comment",
    );
    await delay(1000);
    await page
      .getByRole("button", { name: "Close dialog", exact: true })
      .click();
    await page
      .getByRole("button", { name: "Hide app header and pages", exact: true })
      .click();
    await page
      .getByRole("button", { name: "Add comment", exact: true })
      .click();
    const pin = point(830, 590);
    await page.mouse.click(pin.x, pin.y);
    await page
      .getByLabel("Comment text", { exact: true })
      .pressSequentially("Let's make this happen.", { delay: 45 });
    await page
      .getByRole("button", { name: "Post comment", exact: true })
      .click();
    await page
      .getByRole("button", {
        name: "Comment by alex@demo.invalid",
        exact: true,
      })
      .hover();
    await expect(
      page.getByText("Let's make this happen.", { exact: true }),
    ).toBeVisible();
    await delay(900);
  });
  expect(errors).toEqual([]);
  await page.screenshot({ path: join(output, "board.png") });
  writeFileSync(join(output, "clips.json"), JSON.stringify(clips, null, 2));
  console.log(
    `Recorded ${clips.reduce((n, c) => n + c.frames.length, 0)} real UI frames`,
  );
} finally {
  process.removeListener("SIGINT", interrupt);
  process.removeListener("SIGTERM", interrupt);
  await browser?.close();
  server.kill("SIGTERM");
  await stopped;
  rmSync(data, { recursive: true, force: true });
}
