import { useState } from "react";
import { api } from "../../api";
import { ErrorMessage } from "../../ui";
import { actionLabels, statusLabel } from "./types";
import type { Report, ReviewAction } from "./types";
export default function ReportThread({
  report,
  admin,
  changed,
}: {
  report: Report;
  admin: boolean;
  changed: (report: Report) => void;
}) {
  const [comment, setComment] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  async function act(action?: ReviewAction) {
    setBusy(true);
    setError("");
    try {
      const updated = await api<Report>(
        `/feedback/${report.id}${action ? "" : "/comments"}`,
        action ? "PATCH" : "POST",
        action ? { action, comment } : { body: comment },
      );
      changed(updated);
      setComment("");
    } catch (error) {
      setError((error as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="feedback-thread">
      <p className="muted">
        {report.kind === "bug" ? "Bug" : "Feature request"} ·{" "}
        {statusLabel(report.status)} · {report.authorName}
      </p>
      {report.status === "closed" && report.purgeAt > 0 && (
        <p className="muted">
          This report will be deleted on{" "}
          {new Date(report.purgeAt * 1000).toLocaleDateString()}.
        </p>
      )}
      {report.releaseURL && (
        <a href={report.releaseURL} target="_blank" rel="noopener noreferrer">
          Fixed in {report.release}
        </a>
      )}
      <p className="feedback-body">{report.body}</p>
      {report.hasScreenshot && (
        <a
          href={`/api/v1/feedback/${report.id}/screenshot`}
          target="_blank"
          rel="noopener noreferrer"
        >
          <img
            className="feedback-screenshot"
            src={`/api/v1/feedback/${report.id}/screenshot`}
            alt="Report image"
          />
        </a>
      )}
      <h4>Discussion</h4>
      {report.comments.map((message) => (
        <article className="feedback-message" key={message.id}>
          <strong>{message.authorName}</strong>
          <small>{new Date(message.createdAt * 1000).toLocaleString()}</small>
          <p className="feedback-body">{message.body}</p>
        </article>
      ))}
      <form
        onSubmit={(event) => {
          event.preventDefault();
          void act();
        }}
      >
        <label>
          Add a comment
          <textarea
            disabled={busy}
            maxLength={5000}
            rows={3}
            value={comment}
            onChange={(event) => setComment(event.target.value)}
            placeholder={
              admin
                ? "Explain your decision, or include a release tag such as v1.3.0 when closing with a release."
                : "Add more information or reply to an administrator."
            }
          />
        </label>
        <div className="feedback-comment-actions">
          {admin &&
            (report.actions || []).map((action) => (
              <button
                key={action}
                type="button"
                className="secondary"
                disabled={
                  busy ||
                  (!comment.trim() && !["accept", "reopen"].includes(action))
                }
                onClick={() => void act(action)}
              >
                {actionLabels[action]}
              </button>
            ))}
          <button className="primary" disabled={busy || !comment.trim()}>
            Send comment
          </button>
        </div>
      </form>
      <ErrorMessage message={error} />
    </section>
  );
}
