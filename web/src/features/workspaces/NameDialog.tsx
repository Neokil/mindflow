import type { FormEvent } from "react";
import { useState } from "react";
import { ErrorMessage, Modal, Spinner } from "../../ui";
export default function NameDialog({
  title,
  initial = "",
  action,
  close,
}: {
  title: string;
  initial?: string;
  action: (name: string) => Promise<void>;
  close: () => void;
}) {
  const [name, setName] = useState(initial),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await action(name.trim());
      close();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal title={title} close={close}>
      <form onSubmit={submit}>
        <label>
          Name
          <input
            aria-label="Name"
            autoFocus
            required
            maxLength={200}
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="A name for your ideas"
          />
        </label>
        <ErrorMessage message={error} />
        <div className="form-actions">
          <button className="secondary" type="button" onClick={close}>
            Cancel
          </button>
          <button className="primary" disabled={busy || !name.trim()}>
            {busy ? <Spinner /> : "Save"}
          </button>
        </div>
      </form>
    </Modal>
  );
}
