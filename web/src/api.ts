export class APIError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
  }
}
let csrf = "";
export const setCSRF = (value: string) => {
  csrf = value;
};
export async function api<T = unknown>(
  path: string,
  method = "GET",
  body?: unknown,
): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    method,
    credentials: "same-origin",
    headers: {
      ...(body instanceof FormData
        ? {}
        : { "Content-Type": "application/json" }),
      ...(method !== "GET" ? { "X-CSRF-Token": csrf } : {}),
    },
    body:
      body === undefined
        ? undefined
        : body instanceof FormData
          ? body
          : JSON.stringify(body),
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok)
    throw new APIError(
      response.status,
      data.code || "request_failed",
      data.message || "The request failed. Please retry.",
    );
  return data as T;
}
export type User = {
  id: string;
  email: string;
  role: "admin" | "user";
  status: string;
  appearance: string;
};
export type SMTP = {
  host: string;
  port: number;
  mode: string;
  username: string;
  password?: string;
  from: string;
  name: string;
};
export type Settings = { publicURL: string; smtp: SMTP };
