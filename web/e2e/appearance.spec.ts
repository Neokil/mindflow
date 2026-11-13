import { expect, test } from "@playwright/test";
import { login } from "./helpers";

test("system appearance follows live changes and GitHub aligns with the brand", async ({
  page,
}, info) => {
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await login(page);
  await expect(page.locator(".brand-mark img")).toHaveAttribute(
    "src",
    "/icon.png",
  );
  await expect(page.locator('link[rel="icon"]')).toHaveAttribute(
    "href",
    "/icon.png",
  );
  await expect
    .poll(() =>
      page
        .locator(".brand-mark img")
        .evaluate((image) => (image as HTMLImageElement).naturalWidth),
    )
    .toBeGreaterThan(0);
  const primary = page.getByRole("button", {
    name: "New workspace",
    exact: true,
  });
  await expect(primary).toHaveCSS("background-color", "rgb(164, 214, 94)");
  await expect(primary).toHaveCSS("color", "rgb(27, 94, 59)");
  await page.screenshot({
    path: `../.cache/e2e/branding-${info.project.name}-dark.png`,
  });
  const appearance = page.getByRole("button", { name: "Change appearance" });
  await expect(appearance).toHaveAttribute("title", "Appearance: system");
  await expect(page.locator("html")).toHaveCSS("color-scheme", "dark");
  await page.emulateMedia({ colorScheme: "light" });
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await expect(page.locator("html")).toHaveCSS("color-scheme", "light");
  await expect(primary).toHaveCSS("background-color", "rgb(27, 94, 59)");
  await expect(primary).toHaveCSS("color", "rgb(255, 255, 255)");
  await page.screenshot({
    path: `../.cache/e2e/branding-${info.project.name}-light.png`,
  });
  await page.emulateMedia({ colorScheme: "dark" });
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");

  const brand = (await page.locator(".app-brand .brand").boundingBox())!;
  const icon = (await page
    .getByRole("link", { name: "mindflow on GitHub" })
    .locator("svg")
    .boundingBox())!;
  expect(
    Math.abs(icon.y + icon.height / 2 - brand.y - brand.height / 2),
  ).toBeLessThan(1);

  // Explicit choices remain available; returning to System resumes automatic changes.
  try {
    await appearance.click();
    await expect(appearance).toHaveAttribute("title", "Appearance: light");
    await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
    await appearance.click();
    await expect(appearance).toHaveAttribute("title", "Appearance: dark");
    await page.emulateMedia({ colorScheme: "light" });
    await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
    await appearance.click();
    await expect(appearance).toHaveAttribute("title", "Appearance: system");
    await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
    await page.reload();
    await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
    await page.emulateMedia({ colorScheme: "dark" });
    await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  } finally {
    const me = await (await page.request.get("/api/v1/auth/me")).json();
    await page.request.patch("/api/v1/auth/preferences", {
      headers: { "X-CSRF-Token": me.csrf },
      data: { appearance: "system" },
    });
  }
});
