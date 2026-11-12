import { useEffect, useState } from "react";
import { api } from "../../api";
import type { Workspace } from "../../model";
import { Modal, ErrorMessage } from "../../ui";

type Member = { userId: string; email: string; role: string };
const roles = (
  <>
    <option value="read">Read-only</option>
    <option value="comment">Comment</option>
    <option value="full">Full access</option>
  </>
);
export default function SharingDialog({
  workspace,
  close,
  changed,
}: {
  workspace: Workspace;
  close: () => void;
  changed: () => Promise<void>;
}) {
  const owner = !workspace.role || workspace.role === "owner";
  const [members, setMembers] = useState<Member[]>([]),
    [email, setEmail] = useState(""),
    [role, setRole] = useState("read"),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  useEffect(() => {
    let canceled = false;
    if (owner)
      void api<Member[]>(`/workspaces/${workspace.id}/members`)
        .then((value) => {
          if (!canceled) setMembers(value);
        })
        .catch((err) => {
          if (!canceled) setError(err.message);
        });
    return () => {
      canceled = true;
    };
  }, [workspace.id, owner]);
  async function update(action: () => Promise<unknown>) {
    setBusy(true);
    setError("");
    try {
      await action();
      setMembers(await api(`/workspaces/${workspace.id}/members`));
      await changed();
      setEmail("");
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  const grant = (address: string, permission: string) =>
    api(`/workspaces/${workspace.id}/members`, "PUT", {
      email: address,
      role: permission,
    });
  return (
    <Modal title="Workspace sharing" close={close}>
      {owner ? (
        <>
          <p className="muted">
            Only people you add can open this workspace. Add an existing active
            mindflow user by email.
          </p>
          <form
            onSubmit={(event) => {
              event.preventDefault();
              void update(() => grant(email, role));
            }}
          >
            <label>
              Email address
              <input
                type="email"
                aria-label="Share with email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                required
                disabled={busy}
              />
            </label>
            <label>
              Access
              <select
                aria-label="Sharing access"
                value={role}
                onChange={(event) => setRole(event.target.value)}
                disabled={busy}
              >
                {roles}
              </select>
            </label>
            <button className="primary full" disabled={busy}>
              Add user
            </button>
          </form>
          <div className="sharing-members">
            {members.length === 0 ? (
              <p className="muted">Private — only you have access.</p>
            ) : (
              members.map((member) => (
                <div className="sharing-member" key={member.userId}>
                  <span>{member.email}</span>
                  <select
                    aria-label={`Access for ${member.email}`}
                    value={member.role}
                    disabled={busy}
                    onChange={(event) =>
                      void update(() => grant(member.email, event.target.value))
                    }
                  >
                    {roles}
                  </select>
                  <button
                    className="text-button danger"
                    aria-label={`Remove ${member.email}`}
                    disabled={busy}
                    onClick={() =>
                      void update(() =>
                        api(
                          `/workspaces/${workspace.id}/members/${member.userId}`,
                          "DELETE",
                          {},
                        ),
                      )
                    }
                  >
                    Remove
                  </button>
                </div>
              ))
            )}
          </div>
        </>
      ) : (
        <p>
          You have{" "}
          {workspace.role === "full"
            ? "full"
            : workspace.role === "comment"
              ? "comment"
              : "read-only"}{" "}
          access. Only the workspace owner can manage sharing.
        </p>
      )}
      <ErrorMessage message={error} />
    </Modal>
  );
}
