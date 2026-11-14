import { useState } from "react";
import { api } from "../../api";
import { ErrorMessage } from "../../ui";
import { captureView } from "./capture";
import type { Report } from "./types";
export default function ReportForm({
  created,
  cancel,
}: {
  created: (report: Report) => void;
  cancel: () => void;
}) {
  const [kind, setKind] = useState<"bug" | "feature">(),
    [title, setTitle] = useState(""),
    [body, setBody] = useState(""),
    [screenshot, setScreenshot] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  async function capture() {
    setBusy(true);
    setError("");
    try {
      setScreenshot(await captureView());
    } catch (error) {
      setError((error as Error).message);
    } finally {
      setBusy(false);
    }
  }
  async function upload(file?: File) {
    if (!file) return;
    setError("");
    if (file.type !== "image/png" || file.size > 5 * 1024 * 1024) {
      setError("Choose a PNG up to 5 MB.");
      return;
    }
    const reader = new FileReader();
    reader.onload = () => setScreenshot(String(reader.result).split(",")[1]);
    reader.onerror = () => setError("The screenshot could not be read.");
    reader.readAsDataURL(file);
  }
  async function submit() {
    setBusy(true);
    setError("");
    try {
      created(
        await api<Report>("/feedback", "POST", {
          kind,
          title,
          body,
          screenshot,
        }),
      );
    } catch (error) {
      setError((error as Error).message);
    } finally {
      setBusy(false);
    }
  }
  if (!kind)
    return (
      <section>
        <h3>What would you like to share?</h3>
        <div className="form-actions">
          <button className="secondary" onClick={() => setKind("bug")}>
            Report a bug
          </button>
          <button className="secondary" onClick={() => setKind("feature")}>
            Request a feature
          </button>
          <button className="secondary" onClick={cancel}>
            Cancel
          </button>
        </div>
      </section>
    );
  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        void submit();
      }}
    >
      <h3>{kind === "bug" ? "Report a bug" : "Request a feature"}</h3>
      <p className="muted">
        Private between you and the administrators of this mindflow instance.
      </p>
      <label>
        Title
        <input
          required
          maxLength={150}
          value={title}
          onChange={(event) => setTitle(event.target.value)}
        />
      </label>
      <label>
        Details
        <textarea
          required
          maxLength={10000}
          rows={5}
          value={body}
          onChange={(event) => setBody(event.target.value)}
          placeholder={
            kind === "bug"
              ? "What happened? What did you expect? How can we reproduce it?"
              : "What would you like to do, and how would it help?"
          }
        />
      </label>
      <div className="form-actions">
        <button
          type="button"
          className="secondary"
          disabled={busy}
          onClick={() => void capture()}
        >
          Capture current view
        </button>
        <label className="secondary screenshot-upload">
          Upload PNG
          <input
            aria-label="Upload feedback screenshot"
            type="file"
            accept="image/png"
            disabled={busy}
            onChange={(event) => {
              void upload(event.target.files?.[0]);
              event.target.value = "";
            }}
          />
        </label>
      </div>
      <p className="muted">
        Review the screenshot before sending. It may contain private content.
        Dialogs are excluded from automatic capture.
      </p>
      {screenshot && (
        <figure className="feedback-preview">
          <img
            alt="Screenshot preview"
            src={`data:image/png;base64,${screenshot}`}
          />
          <button
            type="button"
            className="secondary"
            onClick={() => setScreenshot("")}
          >
            Remove screenshot
          </button>
        </figure>
      )}
      <ErrorMessage message={error} />
      <div className="form-actions">
        <button
          type="button"
          className="secondary"
          disabled={busy}
          onClick={cancel}
        >
          Cancel
        </button>
        <button className="primary" disabled={busy}>
          {busy ? "Please wait…" : "Send report"}
        </button>
      </div>
    </form>
  );
}
