import TransferOwnershipDialog from "./TransferOwnershipDialog";
import SharingUserPicker from "./SharingUserPicker";
import type { DirectoryUser } from "../../api";
import Avatar from "../accounts/Avatar";
import { useEffect, useState } from "react";
import { api } from "../../api";
import type { Workspace } from "../../model";
import { Modal, ErrorMessage } from "../../ui";

type Member = {
  userId: string;
  displayName: string;
  role: string;
  avatarVersion?: string;
};
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
  beforeTransfer,
}: {
  workspace: Workspace;
  close: () => void;
  changed: () => Promise<void>;
  beforeTransfer?: () => Promise<void>;
}) {
  const [transferring, setTransferring] = useState(false);
  const owner = !workspace.role || workspace.role === "owner";
  const [workspaceOwner, setWorkspaceOwner] = useState<Member>();
  const [members, setMembers] = useState<Member[]>([]),
    [query, setQuery] = useState(""),
    [selected, setSelected] = useState(""),
    [users, setUsers] = useState<DirectoryUser[]>([]),
    [loadingUsers, setLoadingUsers] = useState(true),
    [role, setRole] = useState("read"),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  useEffect(() => {
    let canceled = false;
    void api<Member>(`/workspaces/${workspace.id}/sharing-info`)
      .then((value) => {
        if (!canceled) setWorkspaceOwner(value);
      })
      .catch((err) => {
        if (!canceled) setError(err.message);
      });
    if (owner)
      void Promise.all([
        api<Member[]>(`/workspaces/${workspace.id}/members`),
        api<DirectoryUser[]>("/accounts/directory"),
      ])
        .then(([members, users]) => {
          if (!canceled) {
            setMembers(members);
            setUsers(users);
          }
        })
        .catch((err) => {
          if (!canceled) setError(err.message);
        })
        .finally(() => {
          if (!canceled) setLoadingUsers(false);
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
      setQuery("");
      setSelected("");
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  const grant = (userId: string, permission: string) =>
    api(`/workspaces/${workspace.id}/members`, "PUT", {
      userId,
      role: permission,
    });
  return (
    <Modal title="Workspace sharing" close={close}>
      {workspaceOwner && (
        <div className="profile-identity">
          <Avatar
            id={workspaceOwner.userId}
            name={workspaceOwner.displayName}
            version={workspaceOwner.avatarVersion}
          />
          <span>
            <strong>{workspaceOwner.displayName}</strong>
            <br />
            <small>Workspace owner</small>
          </span>
        </div>
      )}
      {owner ? (
        <>
          <p className="muted">
            Only people you add can open this workspace. Choose a registered
            user by display name.
          </p>
          <form
            onSubmit={(event) => {
              event.preventDefault();
              if (selected) void update(() => grant(selected, role));
            }}
          >
            <SharingUserPicker
              users={users.filter(
                (user) =>
                  user.id !== workspaceOwner?.userId &&
                  !members.some((member) => member.userId === user.id),
              )}
              query={query}
              onQuery={(value) => {
                setQuery(value);
                setSelected("");
              }}
              selected={selected}
              onSelect={setSelected}
              disabled={busy}
              loading={loadingUsers || !workspaceOwner}
            />
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
            <button className="primary full" disabled={busy || !selected}>
              Add user
            </button>
          </form>
          <div className="sharing-members">
            {members.length === 0 ? (
              <p className="muted">Private — only you have access.</p>
            ) : (
              members.map((member) => (
                <div className="sharing-member" key={member.userId}>
                  <span className="profile-identity">
                    <Avatar
                      id={member.userId}
                      email={member.displayName}
                      name={member.displayName}
                      version={member.avatarVersion}
                    />
                    {member.displayName}
                  </span>
                  <select
                    aria-label={`Access for ${member.displayName}`}
                    value={member.role}
                    disabled={busy}
                    onChange={(event) =>
                      void update(() =>
                        grant(member.userId, event.target.value),
                      )
                    }
                  >
                    {roles}
                  </select>
                  <button
                    className="text-button danger"
                    aria-label={`Remove ${member.displayName}`}
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
                    data-tooltip={`Remove ${member.displayName}`}
                  >
                    Remove
                  </button>
                </div>
              ))
            )}
          </div>
          <section className="ownership-section">
            <h3>Ownership</h3>
            <p className="muted">
              Make another registered user the owner and keep full access.
            </p>
            <button
              className="secondary"
              disabled={busy || loadingUsers}
              onClick={() => setTransferring(true)}
            >
              Transfer ownership
            </button>
          </section>
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
      {transferring && (
        <TransferOwnershipDialog
          workspace={workspace}
          users={users.filter((user) => user.id !== workspaceOwner?.userId)}
          beforeTransfer={beforeTransfer}
          close={() => setTransferring(false)}
          transferred={async () => {
            await changed();
            close();
          }}
        />
      )}
      <ErrorMessage message={error} />
    </Modal>
  );
}
