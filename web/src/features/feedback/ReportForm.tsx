import { useRef, useState } from "react";
import { api } from "../../api";
import { ErrorMessage } from "../../ui";
import { imageAttachment } from "./imageAttachment";
import { captureView } from "./capture";
import type { Report } from "./types";
export default function ReportForm({
  created,
  cancel,
}: {
  created: (report: Report) => void;
  cancel: () => void;
}) {
  const [kind, setKind] = useState<"bug" | "feature">("bug"),
    [title, setTitle] = useState(""),
    [body, setBody] = useState(""),
    [screenshot, setScreenshot] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const fileInput = useRef<HTMLInputElement>(null);
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
    setBusy(true);
    setError("");
    try {
      setScreenshot(await imageAttachment(file));
    } catch (error) {
      setError((error as Error).message);
    } finally {
      setBusy(false);
    }
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
  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        void submit();
      }}
    >
      <label>
        Report type
        <select
          aria-label="Report type"
          value={kind}
          disabled={busy}
          onChange={(event) => setKind(event.target.value as "bug" | "feature")}
        >
          <option value="bug">Bug</option>
          <option value="feature">Feature request</option>
        </select>
      </label>
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
      <div className="feedback-attachment-actions">
        <button
          type="button"
          className="secondary"
          disabled={busy}
          onClick={() => void capture()}
        >
          Capture current view
        </button>
        <button
          type="button"
          className="secondary"
          disabled={busy}
          onClick={() => fileInput.current?.click()}
        >
          Upload image
        </button>
        <input
          ref={fileInput}
          className="feedback-attachment-input"
          aria-label="Upload feedback image"
          type="file"
          accept="image/*"
          disabled={busy}
          onChange={(event) => {
            void upload(event.target.files?.[0]);
            event.target.value = "";
          }}
        />
      </div>
      <p className="muted">
        Review the image before sending. It may contain private content. Dialogs
        are excluded from automatic capture.
      </p>
      {screenshot && (
        <figure className="feedback-preview">
          <img
            alt="Image preview"
            src={`data:image/png;base64,${screenshot}`}
          />
          <button
            type="button"
            className="secondary"
            onClick={() => setScreenshot("")}
          >
            Remove image
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
