import { expect, type Page } from "@playwright/test";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { DatabaseSync } from "node:sqlite";
export async function invite(page: Page, email: string) {
  const status = await page.evaluate(async (address) => {
    const me = await (await fetch("/api/v1/auth/me")).json();
    return (
      await fetch("/api/v1/admin/users", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-CSRF-Token": me.csrf,
        },
        body: JSON.stringify({ email: address, role: "user" }),
      })
    ).status;
  }, email);
  expect(status).toBe(200);
  // Read only the disposable fixture's captured email job; never add a production token endpoint.
  const directory = readFileSync("../.cache/e2e/data-dir", "utf8");
  const database = new DatabaseSync(join(directory, "mindflow.sqlite"), {
    readOnly: true,
  });
  try {
    const row = database
      .prepare(
        "SELECT payload FROM mail_jobs WHERE user_id=(SELECT id FROM users WHERE email=?)",
      )
      .get(email);
    const body = JSON.parse(String(row?.payload)).body as string;
    return body.split("/").at(-1)!.split("\n")[0];
  } finally {
    database.close();
  }
}
