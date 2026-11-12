import { useEffect, useRef, useState } from "react";
import { LogOut, Settings, SunMoon } from "lucide-react";
import { api, setCSRF, APIError } from "./api";
import type { User } from "./api";
import Auth from "./Auth";
import { Dashboard, WorkspaceEditor } from "./Workspaces";
import type { Workspace } from "./model";
import Admin from "./Admin";
import { clearMedia, invalidateMedia } from "./media";
import { localImages } from "./session";
import { Brand, Spinner, ErrorMessage } from "./ui";
export default function App() {
  const [workspace, setWorkspace] = useState<Workspace>();
  const dirty = useRef(false);
  const [reauth, setReauth] = useState(false);
  const [user, setUser] = useState<User>();
  const [initialized, setInitialized] = useState<boolean>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [admin, setAdmin] = useState(false);
  useEffect(() => {
    (async () => {
      try {
        const status = await api<{ initialized: boolean }>("/setup/status");
        setInitialized(status.initialized);
        if (
          status.initialized &&
          !location.hash.match(/^#(accept-invite|reset-password)\//)
        ) {
          try {
            const result = await api<{ user: User; csrf: string }>("/auth/me");
            setCSRF(result.csrf);
            setUser(result.user);
          } catch (e) {
            if (!(e instanceof APIError && e.status === 401)) throw e;
          }
        }
      } catch (e) {
        setError((e as Error).message);
      } finally {
        setLoading(false);
      }
    })();
  }, []);
  useEffect(() => {
    const query = matchMedia("(prefers-color-scheme: dark)");
    const update = () => {
      document.documentElement.dataset.theme =
        user?.appearance === "system" || !user
          ? query.matches
            ? "dark"
            : "light"
          : user.appearance;
    };
    update();
    query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, [user?.appearance]);
  useEffect(() => {
    const source = user ? new EventSource("/api/v1/events") : undefined;
    if (source) {
      source.onmessage = (event) => {
        try {
          const detail = JSON.parse(event.data);
          if (detail.type === "asset") invalidateMedia(detail.id);
          window.dispatchEvent(new CustomEvent("mindflow:event", { detail }));
        } catch {
          /* Reconnect polling remains authoritative. */
        }
      };
      source.addEventListener("expired", () => setReauth(true));
    }
    const onReauth = () => setReauth(true);
    window.addEventListener("mindflow:reauth", onReauth);
    return () => {
      source?.close();
      window.removeEventListener("mindflow:reauth", onReauth);
    };
  }, [user?.id]);
  async function appearance() {
    if (!user) return;
    const next =
      user.appearance === "system"
        ? "light"
        : user.appearance === "light"
          ? "dark"
          : "system";
    try {
      await api("/auth/preferences", "PATCH", { appearance: next });
      setUser({ ...user, appearance: next });
    } catch (e) {
      setError((e as Error).message);
    }
  }
  if (loading)
    return (
      <div className="loading-screen">
        <Brand />
        <Spinner />
      </div>
    );
  if (!user)
    return (
      <>
        <ErrorMessage message={error} />
        <Auth
          setup={initialized === false}
          onLogin={(u) => {
            setUser(u);
            setInitialized(true);
          }}
        />
      </>
    );
  return (
    <div className="app">
      <header className="app-header">
        <Brand small />
        <div className="header-actions">
          <span className="account-email">{user.email}</span>
          <button
            className="icon-button"
            title={`Appearance: ${user.appearance}`}
            aria-label="Change appearance"
            onClick={appearance}
          >
            <SunMoon size={18} />
          </button>
          {user.role === "admin" && (
            <button
              className="icon-button"
              aria-label="Administration"
              onClick={() => setAdmin(true)}
            >
              <Settings size={18} />
            </button>
          )}
          <button
            className="icon-button"
            aria-label="Sign out"
            onClick={async () => {
              if (
                dirty.current &&
                !confirm("Discard unsaved changes and sign out?")
              )
                return;
              try {
                await api("/auth/logout", "POST", {});
                setWorkspace(undefined);
                setUser(undefined);
                setCSRF("");
                dirty.current = false;
                setAdmin(false);
                setReauth(false);
                for (const image of localImages.values())
                  URL.revokeObjectURL(image.url);
                localImages.clear();
                clearMedia();
              } catch (e) {
                setError((e as Error).message);
              }
            }}
          >
            <LogOut size={18} />
          </button>
        </div>
      </header>
      <ErrorMessage message={error} />
      <>
        {workspace ? (
          <WorkspaceEditor
            key={workspace.id}
            workspace={workspace}
            back={() => setWorkspace(undefined)}
            dirtyChanged={(value) => {
              dirty.current = value;
            }}
          />
        ) : (
          <Dashboard open={setWorkspace} />
        )}
      </>
      {admin && <Admin close={() => setAdmin(false)} self={user} />}
      {reauth && (
        <Auth
          overlay
          expectedEmail={user.email}
          onLogin={(u) => {
            if (u.id !== user.id) {
              setWorkspace(undefined);
              dirty.current = false;
              for (const image of localImages.values())
                URL.revokeObjectURL(image.url);
              localImages.clear();
              clearMedia();
            }
            setUser(u);
            setReauth(false);
            window.dispatchEvent(new Event("mindflow:event"));
          }}
        />
      )}
    </div>
  );
}
