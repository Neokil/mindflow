import { LogOut, Settings, SunMoon } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import type { ServerEvent, User } from "./api";
import { api, APIError, setCSRF } from "./api";
import Admin from "./features/accounts/Admin";
import Auth from "./features/accounts/Auth";
import Dashboard from "./features/workspaces/Dashboard";
import WorkspaceEditor from "./features/workspaces/WorkspaceEditor";
import { clearMedia, invalidateMedia } from "./media";
import type { Workspace } from "./model";
import { localImages } from "./session";
import { Brand, ErrorMessage, Spinner } from "./ui";
export default function App() {
  const [serverEvent, setServerEvent] = useState<ServerEvent>({
    type: "connected",
    sequence: 0,
  });
  const [authenticationGeneration, setAuthenticationGeneration] = useState(0);
  const requireAuthentication = useCallback(() => setReauth(true), []);
  const updateDirty = useCallback((value: boolean) => {
    dirty.current = value;
  }, []);
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
  const appearancePreference = user?.appearance || "system";
  const userID = user?.id;
  useEffect(() => {
    const query = matchMedia("(prefers-color-scheme: dark)");
    const update = () => {
      document.documentElement.dataset.theme =
        appearancePreference === "system"
          ? query.matches
            ? "dark"
            : "light"
          : appearancePreference;
    };
    update();
    query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, [appearancePreference]);
  useEffect(() => {
    const source = userID ? new EventSource("/api/v1/events") : undefined;
    if (source) {
      source.onmessage = (event) => {
        try {
          const detail = JSON.parse(event.data);
          if (detail.type === "asset") invalidateMedia(detail.id);
          if (typeof detail.type === "string")
            setServerEvent((previous) => ({
              type: detail.type,
              id: typeof detail.id === "string" ? detail.id : undefined,
              sequence: previous.sequence + 1,
            }));
        } catch {
          /* Reconnect polling remains authoritative. */
        }
      };
      source.addEventListener("expired", () => setReauth(true));
    }
    return () => {
      source?.close();
    };
  }, [userID, authenticationGeneration]);
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
            dirtyChanged={updateDirty}
            onAuthenticationRequired={requireAuthentication}
            serverEvent={serverEvent}
          />
        ) : (
          <Dashboard open={setWorkspace} serverEvent={serverEvent} />
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
            setAuthenticationGeneration((generation) => generation + 1);
            setServerEvent((previous) => ({
              type: "connected",
              sequence: previous.sequence + 1,
            }));
          }}
        />
      )}
    </div>
  );
}
