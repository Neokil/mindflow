export type Report = {
  id: string;
  authorId: string;
  authorName: string;
  kind: "bug" | "feature";
  title: string;
  body?: string;
  status: "created" | "accepted" | "closed";
  unread: number;
  readThrough: number;
  actions: ReviewAction[];
  release: string;
  releaseURL?: string;
  hasScreenshot: boolean;
  createdAt: number;
  updatedAt: number;
  closedAt: number;
  purgeAt: number;
  comments: {
    id: string;
    authorId: string;
    authorName: string;
    body: string;
    createdAt: number;
  }[];
};

export type ReviewAction =
  "accept" | "deny" | "close_release" | "close_comment" | "reopen";
export const actionLabels: Record<ReviewAction, string> = {
  accept: "Accept",
  deny: "Deny",
  close_release: "Close with release",
  close_comment: "Close with comment",
  reopen: "Reopen",
};
export function statusLabel(status: Report["status"]) {
  return status === "accepted"
    ? "Accepted, to do"
    : status === "created"
      ? "Created"
      : "Closed";
}
