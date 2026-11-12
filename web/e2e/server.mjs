import { spawn } from "node:child_process";
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { resolve, join } from "node:path";
const root = resolve("..");
const data = mkdtempSync(join(tmpdir(), "mindflow-e2e-"));
mkdirSync(join(root, ".cache/e2e"), { recursive: true });
writeFileSync(join(root, ".cache/e2e/data-dir"), data, { mode: 0o600 });
const child = spawn(join(root, ".cache/e2e/mindflow"), [], {
  env: {
    ...process.env,
    MINDFLOW_DEV: "1",
    MINDFLOW_DATA: data,
    MINDFLOW_ADDR: "127.0.0.1:8087",
  },
  stdio: ["ignore", "pipe", "pipe"],
});
let stderr = "";
child.stderr.on("data", (chunk) => {
  stderr += chunk;
  const match = stderr.match(/credential \(operator only\): ([a-f0-9]+)/);
  if (match)
    writeFileSync(join(root, ".cache/e2e/bootstrap"), match[1], {
      mode: 0o600,
    });
});
child.on("error", (err) => {
  console.error(err.message);
  process.exitCode = 1;
});
child.on("exit", (code) => {
  rmSync(data, { recursive: true, force: true });
  process.exit(code || 0);
});
for (const signal of ["SIGTERM", "SIGINT"])
  process.on(signal, () => child.kill("SIGTERM"));
