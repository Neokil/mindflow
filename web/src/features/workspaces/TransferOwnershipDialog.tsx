import { useState } from "react";
import type { DirectoryUser } from "../../api";
import { api } from "../../api";
import type { Workspace } from "../../model";
import { Modal, ErrorMessage, Spinner } from "../../ui";
import SharingUserPicker from "./SharingUserPicker";
export default function TransferOwnershipDialog({
  workspace,
  users,
  close,
  transferred,
  beforeTransfer,
}: {
  workspace: Workspace;
  users: DirectoryUser[];
  close: () => void;
  transferred: () => Promise<void>;
  beforeTransfer?: () => Promise<void>;
}) {
  const [query, setQuery] = useState(""),
    [selected, setSelected] = useState(""),
    [confirmed, setConfirmed] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  async function submit() {
    setBusy(true);
    setError("");
    try {
      await beforeTransfer?.();
      await api(`/workspaces/${workspace.id}/ownership`, "POST", {
        userId: selected,
        revision: workspace.revision,
        confirm: confirmed,
      });
      await transferred();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal
      title="Transfer ownership"
      close={() => {
        if (!busy) close();
      }}
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          void submit();
        }}
      >
        <p className="muted">
          Choose the new owner of “{workspace.name}”. They will manage sharing
          and deletion. You will keep full access. Existing editor sessions will
          be released.
        </p>
        <SharingUserPicker
          users={users}
          query={query}
          onQuery={(value) => {
            setQuery(value);
            setSelected("");
            setConfirmed(false);
          }}
          selected={selected}
          onSelect={(value) => {
            setSelected(value);
            setConfirmed(false);
          }}
          disabled={busy}
          loading={false}
        />
        <label className="check-label">
          <input
            type="checkbox"
            aria-label="Confirm ownership transfer"
            checked={confirmed}
            disabled={busy || !selected}
            onChange={(e) => setConfirmed(e.target.checked)}
          />
          I understand that I will become a full-access member.
        </label>
        <ErrorMessage message={error} />
        <div className="form-actions">
          <button
            type="button"
            className="secondary"
            disabled={busy}
            onClick={close}
          >
            Cancel
          </button>
          <button
            className="primary"
            disabled={busy || !selected || !confirmed}
          >
            {busy ? <Spinner /> : null}Transfer ownership
          </button>
        </div>
      </form>
    </Modal>
  );
}
