import Avatar from "./Avatar";
import {
  Mail,
  Plus,
  RotateCcw,
  Settings as SettingsIcon,
  Trash2,
  Users,
} from "lucide-react";
import { useEffect, useState } from "react";
import type { Settings, User } from "../../api";
import { api } from "../../api";
import { ErrorMessage, Modal, Spinner } from "../../ui";
import SMTPFields from "./SMTPFields";
export default function Admin({
  close,
  self,
}: {
  close: () => void;
  self: User;
}) {
  const [tab, setTab] = useState("users");
  const [users, setUsers] = useState<{ user: User; mailError: string }[]>([]);
  const [settings, setSettings] = useState<Settings>();
  const [email, setEmail] = useState("");
  const [role, setRole] = useState("user");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  async function load() {
    setUsers(await api("/admin/users"));
    const result = await api<{ settings: Settings }>("/admin/settings");
    setSettings(result.settings);
  }
  useEffect(() => {
    load().catch((e) => setError(e.message));
  }, []);
  useEffect(() => {
    if (tab !== "users") return;
    const timer = setInterval(() => {
      api<{ user: User; mailError: string }[]>("/admin/users")
        .then(setUsers)
        .catch((e) => setError(e.message));
    }, 5000);
    return () => clearInterval(timer);
  }, [tab]);
  async function act(fn: () => Promise<unknown>, success = "") {
    setError("");
    setMessage("");
    setBusy(true);
    try {
      await fn();
      await load();
      setMessage(success);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal title="Administration" close={close} wide>
      <div className="segmented">
        <button
          className={tab === "users" ? "active" : ""}
          onClick={() => setTab("users")}
        >
          <Users size={16} />
          Users
        </button>
        <button
          className={tab === "settings" ? "active" : ""}
          onClick={() => setTab("settings")}
        >
          <SettingsIcon size={16} />
          Email & server
        </button>
      </div>
      <ErrorMessage message={error} />
      {message && (
        <p className="success" role="status">
          {message}
        </p>
      )}
      {tab === "users" ? (
        <>
          <p className="muted">
            Manage accounts. Workspaces and files remain private to their
            owners.
          </p>
          <form
            className="invite-form"
            onSubmit={(e) => {
              e.preventDefault();
              act(
                () => api("/admin/users", "POST", { email, role }),
                "Invitation queued.",
              ).then(() => setEmail(""));
            }}
          >
            <label>
              Email address
              <input
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="friend@example.com"
              />
            </label>
            <label>
              Role
              <select value={role} onChange={(e) => setRole(e.target.value)}>
                <option value="user">User</option>
                <option value="admin">Administrator</option>
              </select>
            </label>
            <button className="primary" disabled={busy}>
              <Plus size={16} />
              Invite
            </button>
          </form>
          <div className="user-list">
            {users.map(({ user: u, mailError }) => (
              <div className="user-row" key={u.id}>
                <Avatar
                  id={u.id}
                  email={u.email}
                  name={u.displayName}
                  version={u.avatarVersion}
                />
                <div className="user-info">
                  <strong>
                    {u.displayName}
                    {u.id === self.id ? " (you)" : ""}
                  </strong>
                  <small>{u.email}</small>
                  <span>
                    {u.role === "admin" ? "Administrator" : "User"} · {u.status}
                  </span>
                  {mailError && <small className="error">{mailError}</small>}
                </div>
                <div className="row-actions">
                  <select
                    aria-label={`Role for ${u.email}`}
                    disabled={busy}
                    value={u.role}
                    onChange={(e) =>
                      act(() =>
                        api(`/admin/users/${u.id}`, "PATCH", {
                          role: e.target.value,
                        }),
                      )
                    }
                  >
                    <option value="user">User</option>
                    <option value="admin">Admin</option>
                  </select>
                  {u.status === "invited" ? (
                    <button
                      className="icon-button"
                      data-tooltip="Resend invitation"
                      aria-label={`Resend invitation to ${u.email}`}
                      disabled={busy}
                      onClick={() =>
                        act(
                          () => api(`/admin/users/${u.id}/invite`, "POST", {}),
                          "Invitation queued.",
                        )
                      }
                    >
                      <Mail size={17} />
                    </button>
                  ) : (
                    <>
                      <button
                        className="secondary compact"
                        disabled={busy}
                        onClick={() =>
                          act(() =>
                            api(`/admin/users/${u.id}`, "PATCH", {
                              status:
                                u.status === "active" ? "inactive" : "active",
                            }),
                          )
                        }
                      >
                        {u.status === "active" ? "Deactivate" : "Activate"}
                      </button>
                      <button
                        className="icon-button"
                        data-tooltip="Send password reset"
                        aria-label={`Send password reset to ${u.email}`}
                        disabled={busy || u.status !== "active"}
                        onClick={() =>
                          act(
                            () => api(`/admin/users/${u.id}/reset`, "POST", {}),
                            "Password reset queued.",
                          )
                        }
                      >
                        <RotateCcw size={17} />
                      </button>
                    </>
                  )}
                  <button
                    className="icon-button danger"
                    aria-label={`Delete ${u.email}`}
                    disabled={busy}
                    onClick={() => {
                      if (
                        confirm(
                          `Permanently delete ${u.email} and all their workspaces and images?`,
                        )
                      )
                        act(() =>
                          api(`/admin/users/${u.id}`, "DELETE", {
                            confirm: true,
                          }),
                        );
                    }}
                    data-tooltip={`Delete ${u.email}`}
                  >
                    <Trash2 size={17} />
                  </button>
                </div>
              </div>
            ))}
          </div>
        </>
      ) : settings ? (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            act(
              () => api("/admin/settings", "PUT", settings),
              "Settings saved.",
            );
          }}
        >
          <SMTPFields value={settings} onChange={setSettings} />
          <div className="form-actions">
            <button
              type="button"
              className="secondary"
              disabled={busy}
              onClick={() =>
                act(
                  () => api("/admin/settings/test-email", "POST", {}),
                  "Test email sent to your address.",
                )
              }
            >
              <Mail size={16} />
              Test saved settings
            </button>
            <button className="primary" disabled={busy}>
              {busy ? <Spinner /> : "Save settings"}
            </button>
          </div>
        </form>
      ) : (
        <Spinner />
      )}
    </Modal>
  );
}
