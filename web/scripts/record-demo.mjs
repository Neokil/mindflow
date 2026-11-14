/* global document */
// Record only an isolated local fixture. Never run this workflow against a real instance.
import { chromium, expect } from "@playwright/test";
import { spawn, spawnSync } from "node:child_process";
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { DatabaseSync } from "node:sqlite";

const guide = process.argv.includes("--guide");
const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const output = join(root, ".cache/demo");
const frames = mkdtempSync(join(tmpdir(), "mindflow-frames-"));
const data = mkdtempSync(join(tmpdir(), "mindflow-demo-"));
const origin = "http://127.0.0.1:8089";
mkdirSync(output, { recursive: true });
let bootstrap = "",
  logs = "";
const server = spawn(join(output, "mindflow"), [], {
  env: {
    ...process.env,
    MINDFLOW_DEV: "1",
    MINDFLOW_ADDR: "127.0.0.1:8089",
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
let browser,
  recording = false,
  capture;
const interrupt = () => {
  recording = false;
  server.kill("SIGTERM");
  void browser?.close().catch(() => {});
};
process.once("SIGINT", interrupt);
process.once("SIGTERM", interrupt);
const delay = (milliseconds) =>
  new Promise((resolve) => setTimeout(resolve, milliseconds));
try {
  await expect
    .poll(
      async () => {
        try {
          return bootstrap && (await fetch(`${origin}/health/ready`)).ok;
        } catch {
          return false;
        }
      },
      { timeout: 15000 },
    )
    .toBeTruthy();
  browser = await chromium.launch();
  const context = await browser.newContext({
    viewport: { width: 1200, height: 800 },
    deviceScaleFactor: 1,
    colorScheme: "dark",
    recordVideo: {
      dir: join(output, "video"),
      size: { width: 1200, height: 800 },
    },
  });
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto(origin);
  await page.getByLabel("Setup credential").fill(bootstrap);
  await page
    .getByLabel("Email address", { exact: true })
    .fill("alex@example.com");
  await page
    .getByLabel("Password", { exact: true })
    .fill("a private demo passphrase");
  await page.getByLabel("Public application URL").fill(origin);
  await page.getByLabel("SMTP host").fill("127.0.0.1");
  await page.getByLabel("Port", { exact: true }).fill("9");
  await page.getByLabel("Sender email").fill("mindflow@example.com");
  await page
    .getByRole("button", { name: "Create mindflow", exact: true })
    .click();
  await page
    .getByLabel("Email address", { exact: true })
    .fill("alex@example.com");
  await page
    .getByLabel("Password", { exact: true })
    .fill("a private demo passphrase");
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Workspaces", exact: true }),
  ).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  const me = await (await page.request.get(`${origin}/api/v1/auth/me`)).json();
  const api = async (path, body) => {
    const response = await page.request.post(`${origin}/api/v1${path}`, {
      headers: { "X-CSRF-Token": me.csrf },
      data: body,
    });
    if (!response.ok())
      throw new Error(`Fixture request failed: ${path} (${response.status()})`);
    return response.json();
  };
  await api("/admin/users", { email: "sam@example.com", role: "user" });
  const database = new DatabaseSync(join(data, "mindflow.sqlite"), {
    readOnly: true,
  });
  let invitation;
  try {
    invitation = JSON.parse(
      database
        .prepare(
          "SELECT payload FROM mail_jobs WHERE user_id=(SELECT id FROM users WHERE email=?)",
        )
        .get("sam@example.com").payload,
    ).body.match(/#accept-invite\/([a-f0-9]+)/)[1];
  } finally {
    database.close();
  }
  await api("/auth/accept-invite", {
    token: invitation,
    password: "another private demo passphrase",
  });
  if (guide)
    await page.addStyleTag({
      content: "#demo-caption,#demo-pointer{display:none!important}",
    });
  await page.addStyleTag({
    content: `#demo-caption{position:fixed;left:24px;bottom:65px;z-index:10000;padding:12px 18px;border:1px solid var(--line);border-radius:12px;background:var(--panel);box-shadow:var(--shadow);color:var(--text);font:500 15px Arial;pointer-events:none}#demo-pointer{position:fixed;width:12px;height:12px;border:2px solid white;background:var(--sprout);border-radius:50%;z-index:10001;pointer-events:none;transform:translate(-50%,-50%);box-shadow:0 1px 5px #0004}`,
  });
  await page.evaluate(() => {
    const caption = document.createElement("div");
    caption.id = "demo-caption";
    document.body.append(caption);
    const pointer = document.createElement("div");
    pointer.id = "demo-pointer";
    pointer.style.left = "-30px";
    document.body.append(pointer);
    document.addEventListener("mousemove", (event) => {
      pointer.style.left = `${event.clientX}px`;
      pointer.style.top = `${event.clientY}px`;
    });
  });
  const caption = async (text) => {
    console.log(text);
    if (guide) return;
    await page
      .locator("#demo-caption")
      .evaluate((element, value) => (element.textContent = value), text);
    await delay(700);
  };
  const manifest = [];
  recording = !guide;
  capture = (async () => {
    while (recording) {
      const file = `${String(manifest.length).padStart(5, "0")}.png`;
      const started = performance.now();
      await page.screenshot({ path: join(frames, file) });
      manifest.push({ file, time: started });
      await delay(170);
    }
  })();
  // Mark screenshot failures as handled while the UI step reports its own failure.
  void capture.catch(() => {});
  await caption("1. Create a workspace for your next idea");
  await page
    .getByRole("button", { name: "New workspace", exact: true })
    .click();
  await page
    .getByLabel("Name", { exact: true })
    .pressSequentially("A little room for ideas", { delay: 70 });
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByTestId("canvas")).toBeVisible();
  await caption("2. Add an image, then drag to choose its size");
  const encoded = await page.evaluate(() => {
    const image = document.createElement("canvas");
    image.width = 640;
    image.height = 400;
    const ctx = image.getContext("2d");
    const sky = ctx.createLinearGradient(0, 0, 0, 400);
    sky.addColorStop(0, "#c9dcf5");
    sky.addColorStop(1, "#f6e6dc");
    ctx.fillStyle = sky;
    ctx.fillRect(0, 0, 640, 400);
    ctx.fillStyle = "#fff0c8";
    ctx.beginPath();
    ctx.arc(490, 100, 40, 0, Math.PI * 2);
    ctx.fill();
    for (const [color, points] of [
      [
        "#a7b8d7",
        [
          [0, 300],
          [160, 110],
          [320, 290],
          [450, 170],
          [640, 310],
        ],
      ],
      [
        "#708eaa",
        [
          [0, 330],
          [160, 210],
          [350, 360],
          [520, 220],
          [640, 340],
        ],
      ],
      [
        "#496c78",
        [
          [0, 365],
          [170, 330],
          [350, 370],
          [500, 315],
          [640, 350],
        ],
      ],
    ]) {
      ctx.fillStyle = color;
      ctx.beginPath();
      ctx.moveTo(0, 400);
      for (const [x, y] of points) ctx.lineTo(x, y);
      ctx.lineTo(640, 400);
      ctx.fill();
    }
    return image.toDataURL("image/png").split(",")[1];
  });
  await page.getByRole("button", { name: "Add images", exact: true }).click();
  await page.locator('input[type="file"]').setInputFiles({
    name: "weekend-escape.png",
    mimeType: "image/png",
    buffer: Buffer.from(encoded, "base64"),
  });
  await expect(
    page.getByText("Drag to size image", { exact: false }),
  ).toBeVisible();
  const bounds = await page.getByTestId("canvas").boundingBox();
  const drag = async (x, y, width, height) => {
    await page.mouse.move(bounds.x + x, bounds.y + y);
    await page.mouse.down();
    await page.mouse.move(bounds.x + x + width, bounds.y + y + height, {
      steps: 30,
    });
    await delay(500);
    await page.mouse.up();
    await delay(700);
  };
  await drag(170, 190, 370, 230);
  await expect(page.getByTestId("element-count")).toHaveText("1 element");
  await caption("3. Frame your thoughts with shapes and text");
  await page.getByRole("button", { name: "Shapes", exact: true }).click();
  await page
    .getByRole("menuitemradio", { name: "Rectangle", exact: true })
    .click();
  await drag(610, 200, 300, 180);
  await page.getByRole("button", { name: "Text (T)", exact: true }).click();
  await drag(625, 215, 270, 135);
  await page
    .getByLabel("Edit canvas text")
    .pressSequentially("Make space\nfor the next big idea.", { delay: 65 });
  await page.getByLabel("Edit canvas text").press("Control+Enter");
  // Gold keeps the heading readable on the Forest card.
  await page
    .getByRole("button", { name: "Selection properties", exact: true })
    .click();
  await page.getByLabel("Element color", { exact: true }).fill("#d9a441");
  await page
    .getByRole("button", { name: "Close settings", exact: true })
    .click();
  await page.getByRole("button", { name: "Shapes", exact: true }).click();
  await page.getByRole("menuitemradio", { name: "Star", exact: true }).click();
  await drag(940, 220, 90, 90);
  await page.getByRole("button", { name: "Pen (P)", exact: true }).click();
  await page.getByLabel("Pen color", { exact: true }).fill("#a4d65e");
  await page.mouse.move(bounds.x + 630, bounds.y + 420);
  await page.mouse.down();
  for (let index = 0; index < 30; index++)
    await page.mouse.move(
      bounds.x + 630 + index * 9,
      bounds.y + 420 + Math.sin(index / 5) * 15,
    );
  await page.mouse.up();
  await page.getByRole("button", { name: "Select (V)", exact: true }).click();
  await expect(page.getByTestId("element-count")).toHaveText("5 elements");
  await expect(page.getByTestId("save-status")).toHaveText("Saved");
  if (guide)
    await page.screenshot({ path: join(root, "docs/media/guide-canvas.png") });
  await caption("4. Share your workspace with someone you trust");
  await page
    .getByRole("button", { name: "Workspace sharing", exact: true })
    .click();
  await page.getByLabel("Search users", { exact: true }).fill("sam");
  await page.getByRole("radio", { name: "sam", exact: true }).check();
  await page
    .getByLabel("Sharing access", { exact: true })
    .selectOption("comment");
  await page.getByRole("button", { name: "Add user", exact: true }).click();
  await expect(page.getByLabel("Access for sam")).toHaveValue("comment");
  if (guide)
    await page.screenshot({ path: join(root, "docs/media/guide-sharing.png") });
  await delay(1200);
  await page.getByRole("button", { name: "Close dialog", exact: true }).click();
  await caption("5. Start a conversation right on the canvas");
  await page.getByRole("button", { name: "Add comment", exact: true }).click();
  await page.mouse.click(bounds.x + 540, bounds.y + 360);
  await page
    .getByLabel("Comment text", { exact: true })
    .pressSequentially("Sam, what do you think of this direction?", {
      delay: 55,
    });
  await page.getByRole("button", { name: "Post comment", exact: true }).click();
  const pin = page.getByRole("button", {
    name: "Comment by alex",
    exact: true,
  });
  await pin.hover();
  await expect(
    page.getByText("Sam, what do you think of this direction?", {
      exact: true,
    }),
  ).toBeVisible();
  await delay(1600);
  if (guide) {
    await page.screenshot({
      path: join(root, "docs/media/guide-comments.png"),
    });
    await page
      .getByRole("button", { name: "Back to workspaces", exact: true })
      .click();
    await expect(
      page.getByRole("button", {
        name: "Open A little room for ideas",
        exact: true,
      }),
    ).toBeVisible();
    await page.screenshot({
      path: join(root, "docs/media/guide-workspaces.png"),
    });
  }
  await caption("mindflow — a little space for your thoughts");
  await delay(1800);
  recording = false;
  await capture;
  expect(errors).toEqual([]);
  const entries = manifest.map((frame, index) => ({
    file: frame.file,
    duration:
      index + 1 < manifest.length
        ? Math.max(20, Math.round(manifest[index + 1].time - frame.time))
        : 1500,
  }));
  const manifestPath = join(frames, "frames.json");
  writeFileSync(manifestPath, JSON.stringify(entries));
  await context.close();
  if (!guide) {
    const result = spawnSync(
      join(root, ".cache/demo-venv/bin/python"),
      [
        join(root, "scripts/demo-gif.py"),
        manifestPath,
        join(root, "docs/media/workflow.gif"),
      ],
      { stdio: "inherit" },
    );
    if (result.status !== 0) throw new Error("GIF encoding failed");
    console.log("Raw Playwright video: .cache/demo/video/");
  }
} finally {
  process.removeListener("SIGINT", interrupt);
  process.removeListener("SIGTERM", interrupt);
  recording = false;
  if (capture) await capture.catch(() => {});
  await browser?.close();
  server.kill("SIGTERM");
  await stopped;
  rmSync(data, { recursive: true, force: true });
  rmSync(frames, { recursive: true, force: true });
}
