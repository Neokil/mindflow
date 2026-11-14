import { useState } from "react";
import { api } from "../../api";
import { ErrorMessage } from "../../ui";
import type { Report } from "./types";
export default function ReportThread({
  report,
  admin,
  changed,
  back,
}: {
  report: Report;
  admin: boolean;
  changed: (report: Report) => void;
  back: () => void;
}) {
  const [comment, setComment] = useState(""),
    [release, setRelease] = useState(report.release),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  async function act(path: string, method: string, body: unknown) {
    setBusy(true);
    setError("");
    try {
      changed(await api<Report>(path, method, body));
      setComment("");
    } catch (error) {
      setError((error as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="feedback-thread">
      <button className="secondary" onClick={back}>
        Back to reports
      </button>
      <h3>{report.title}</h3>
      <p className="muted">
        {report.kind === "bug" ? "Bug" : "Feature request"} · {report.status} ·{" "}
        {report.authorName}
      </p>
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
            alt="Report screenshot"
          />
        </a>
      )}
      {admin && (
        <div className="feedback-review">
          <h4>Review report</h4>
          <div className="form-actions">
            {["accepted", "denied", "submitted"].map((status) => (
              <button
                key={status}
                className="secondary"
                disabled={busy || report.status === status}
                onClick={() =>
                  void act(`/feedback/${report.id}`, "PATCH", {
                    status,
                    release: "",
                  })
                }
              >
                {status === "accepted"
                  ? "Accept"
                  : status === "denied"
                    ? "Deny"
                    : "Reopen"}
              </button>
            ))}
          </div>
          {report.kind === "bug" && (
            <div className="form-actions">
              <label>
                Fixed in release
                <input
                  placeholder="v1.2.0"
                  value={release}
                  onChange={(event) => setRelease(event.target.value)}
                />
              </label>
              <button
                className="secondary"
                disabled={busy || !release}
                onClick={() =>
                  void act(`/feedback/${report.id}`, "PATCH", {
                    status: "closed",
                    release,
                  })
                }
              >
                Close with release
              </button>
            </div>
          )}
        </div>
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
          void act(`/feedback/${report.id}/comments`, "POST", {
            body: comment,
          });
        }}
      >
        <label>
          Add a comment
          <textarea
            required
            maxLength={5000}
            rows={3}
            value={comment}
            onChange={(event) => setComment(event.target.value)}
          />
        </label>
        <button className="primary" disabled={busy || !comment.trim()}>
          Send comment
        </button>
      </form>
      <ErrorMessage message={error} />
    </section>
  );
}
