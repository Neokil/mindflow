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
async function request(
  path: string,
  method = "GET",
  body?: unknown,
): Promise<Response> {
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
  if (!response.ok) {
    const data = await response.json().catch(() => ({}));
    throw new APIError(
      response.status,
      data.code || "request_failed",
      data.message || "The request failed. Please retry.",
    );
  }
  return response;
}
export async function api<T = unknown>(
  path: string,
  method = "GET",
  body?: unknown,
): Promise<T> {
  return (await request(path, method, body)).json().catch(() => ({}));
}
export async function apiFile(
  path: string,
  body: unknown,
  onNotes?: (notes: string[]) => void,
): Promise<Blob> {
  const response = await request(path, "POST", body);
  const notes = response.headers.get("X-Mindflow-Conversion-Notes");
  if (notes && onNotes) onNotes(JSON.parse(notes));
  return response.blob();
}
export type User = {
  displayName: string;
  avatarVersion?: string;
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

export type ServerEvent = { type: string; id?: string; sequence: number };

export type DirectoryUser = Pick<User, "id" | "displayName" | "avatarVersion">;
