import { defineConfig, devices } from "@playwright/test";
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  workers: 1,
  timeout: 45000,
  expect: { timeout: 10000 },
  reporter: "list",
  use: {
    baseURL: "http://127.0.0.1:8087",
    viewport: { width: 1440, height: 1000 },
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    {
      name: "chromium",
      testIgnore: ["**/http.spec.ts", "**/tablet.spec.ts"],
      use: {
        ...devices["Desktop Chrome"],
        viewport: { width: 1440, height: 1000 },
      },
    },
    {
      name: "firefox",
      // Firefox can pause timers/painting on macOS under browser-test load.
      expect: { timeout: 20000 },
      testIgnore: ["**/http.spec.ts", "**/tablet.spec.ts"],
      use: {
        ...devices["Desktop Firefox"],
        deviceScaleFactor: 2,
        viewport: { width: 1440, height: 1000 },
      },
    },
    {
      name: "ipad-webkit",
      testMatch: [
        "**/tablet.spec.ts",
        "**/focus-mode.spec.ts",
        "**/canvas-tools.spec.ts",
        "**/canvas-refinements.spec.ts",
        "**/ink.spec.ts",
        "**/pdf.spec.ts",
        "**/transfer.spec.ts",
        "**/profiles.spec.ts",
        "**/workspace-previews.spec.ts",
        "**/element-links.spec.ts",
        "**/trash.spec.ts",
        "**/ownership.spec.ts",
      ],
      use: { ...devices["iPad Pro 11"], browserName: "webkit" },
    },
    {
      name: "tablet-chromium",
      testMatch: [
        "**/tablet.spec.ts",
        "**/focus-mode.spec.ts",
        "**/canvas-tools.spec.ts",
        "**/canvas-refinements.spec.ts",
        "**/ink.spec.ts",
        "**/pdf.spec.ts",
        "**/transfer.spec.ts",
        "**/profiles.spec.ts",
        "**/workspace-previews.spec.ts",
        "**/element-links.spec.ts",
        "**/trash.spec.ts",
        "**/ownership.spec.ts",
      ],
      use: { ...devices["iPad Pro 11"], browserName: "chromium" },
    },
    {
      name: "chromium-http",
      testMatch: "**/http.spec.ts",
      use: {
        ...devices["Desktop Chrome"],
        baseURL: "http://mindflow.test:8087",
        viewport: { width: 1440, height: 1000 },
        launchOptions: {
          args: [
            "--host-resolver-rules=MAP mindflow.test 127.0.0.1",
            "--no-proxy-server",
          ],
        },
      },
    },
  ],
  webServer: {
    command: "node e2e/server.mjs",
    url: "http://127.0.0.1:8087/health/ready",
    reuseExistingServer: false,
    timeout: 30000,
  },
});
