import { useState } from "react";
import { api } from "../../api";
import type { User } from "../../api";
import { ErrorMessage, Modal } from "../../ui";
import Avatar from "./Avatar";
export default function Profile({
  user,
  changed,
  close,
}: {
  user: User;
  changed: (user: User) => void;
  close: () => void;
}) {
  const [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  async function update(file?: File) {
    setBusy(true);
    setError("");
    try {
      if (file) {
        if (file.size > 2 * 1024 * 1024)
          throw new Error("Choose an image up to 2 MB.");
        const form = new FormData();
        form.append("file", file);
        await api("/auth/avatar", "POST", form);
      } else await api("/auth/avatar", "DELETE");
      const result = await api<{ user: User }>("/auth/me");
      changed(result.user);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal title="Your profile" close={close}>
      <div className="profile-identity">
        <Avatar id={user.id} email={user.email} version={user.avatarVersion} />
        <strong>{user.email}</strong>
      </div>
      <label>
        Avatar
        <input
          aria-label="Upload avatar"
          type="file"
          accept="image/png,image/jpeg,image/webp"
          disabled={busy}
          onChange={(event) => {
            const file = event.target.files?.[0];
            if (file) void update(file);
            event.target.value = "";
          }}
        />
      </label>
      <p className="muted">
        PNG, JPEG or WebP, up to 2 MB. Your image is cropped to a square.
      </p>
      <button
        className="secondary"
        disabled={busy || !user.avatarVersion}
        onClick={() => void update()}
      >
        Remove avatar
      </button>
      <ErrorMessage message={error} />
    </Modal>
  );
}
