import BuildVersion from "./BuildVersion";
import UpdateBanner from "./UpdateBanner";
import { useCanvasFullscreen } from "./useCanvasFullscreen";
import { useWorkspaceRoute } from "./navigation";
import { Github, LogOut, Bug, Settings, SunMoon } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import type { ServerEvent, User } from "./api";
import { api, APIError, setCSRF } from "./api";
import Feedback from "./features/feedback/Reports";
import useFeedbackUnread from "./features/feedback/useFeedbackUnread";
import Profile from "./features/accounts/Profile";
import Avatar from "./features/accounts/Avatar";
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
  const { route, navigate } = useWorkspaceRoute(dirty);
  const [opening, setOpening] = useState(false);
  const [reauth, setReauth] = useState(false);
  const [user, setUser] = useState<User>();
  const unreadFeedback = useFeedbackUnread(user?.id);
  const [initialized, setInitialized] = useState<boolean>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [feedback, setFeedback] = useState(false);
  const [profile, setProfile] = useState(false);
  const [admin, setAdmin] = useState(false);
  const [headerCollapsed, setHeaderCollapsed] = useState(false);
  const editorOpen =
    !opening && workspace?.id === route.workspaceId && !!workspace;
  const fullscreen = useCanvasFullscreen(editorOpen);
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
  const routeWorkspace = route.workspaceId;
  useEffect(() => {
    let cancelled = false;
    setWorkspace(undefined);
    if (!userID || !routeWorkspace) {
      setOpening(false);
      return;
    }
    setOpening(true);
    setError("");
    void api<Workspace[]>("/workspaces")
      .then((list) => {
        if (cancelled) return;
        const target = list.find((item) => item.id === routeWorkspace);
        if (target) setWorkspace(target);
        else
          setError(
            "This workspace is unavailable or has not been shared with you.",
          );
      })
      .catch((err) => {
        if (!cancelled) setError(err.message);
      })
      .finally(() => {
        if (!cancelled) setOpening(false);
      });
    return () => {
      cancelled = true;
    };
  }, [userID, routeWorkspace]);
  const choosePage = useCallback(
    (pageId?: string, replace = false) => {
      navigate({ workspaceId: routeWorkspace, pageId }, replace);
    },
    [navigate, routeWorkspace],
  );

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
        <UpdateBanner />
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
    <div
      className={`app${editorOpen ? " editor-open" : ""}${editorOpen && headerCollapsed ? " header-collapsed" : ""}${editorOpen && fullscreen.active ? " canvas-fullscreen" : ""}`}
    >
      <UpdateBanner />
      <div className="app-content">
        <header className="app-header" id="app-header">
          <div className="app-brand">
            <div className="brand-version">
              <Brand small />
              <BuildVersion />
            </div>
            <a
              className="icon-button"
              href="https://github.com/Neokil/mindflow"
              target="_blank"
              rel="noopener noreferrer"
              aria-label="mindflow on GitHub"
              data-tooltip="View on GitHub"
            >
              <Github size={18} />
            </a>
          </div>
          <div className="header-actions">
            <button
              className="account-profile"
              aria-label="Your profile"
              onClick={() => setProfile(true)}
              data-tooltip="Your profile"
            >
              <Avatar
                id={user.id}
                email={user.email}
                name={user.displayName}
                version={user.avatarVersion}
              />
              <span className="account-email" title={user.displayName}>
                {user.displayName}
              </span>
            </button>
            <button
              className="icon-button"
              data-tooltip={`Appearance: ${user.appearance}`}
              aria-label="Change appearance"
              onClick={appearance}
            >
              <SunMoon size={18} />
            </button>
            {user.role === "admin" && (
              <button
                className="icon-button notification-button"
                aria-label="Administration"
                aria-description={
                  unreadFeedback.reports > 0
                    ? `${unreadFeedback.reports} unread reports and comments`
                    : undefined
                }
                onClick={() => setAdmin(true)}
                data-tooltip="Administration"
              >
                <Settings size={18} />
                {unreadFeedback.reports > 0 && (
                  <span
                    className="notification-badge"
                    data-testid="admin-feedback-unread"
                    aria-label={`${unreadFeedback.reports} unread reports and comments`}
                  >
                    {unreadFeedback.reports}
                  </span>
                )}
              </button>
            )}
            <button
              className="icon-button notification-button"
              aria-label="Feedback & Bugreport"
              aria-description={
                unreadFeedback.replies > 0
                  ? `${unreadFeedback.replies} unread replies`
                  : undefined
              }
              data-tooltip="Feedback & Bugreport"
              onClick={() => setFeedback(true)}
            >
              <Bug size={18} />
              {unreadFeedback.replies > 0 && (
                <span
                  className="notification-badge"
                  data-testid="feedback-unread"
                  aria-label={`${unreadFeedback.replies} unread replies`}
                >
                  {unreadFeedback.replies}
                </span>
              )}
            </button>
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
              data-tooltip="Sign out"
            >
              <LogOut size={18} />
            </button>
          </div>
        </header>
        <ErrorMessage message={error} />
        <>
          {opening ? (
            <div className="editor-loading">
              <Spinner />
            </div>
          ) : workspace && workspace.id === route.workspaceId ? (
            <WorkspaceEditor
              key={workspace.id}
              workspace={workspace}
              headerCollapsed={headerCollapsed}
              onToggleHeader={() => setHeaderCollapsed((value) => !value)}
              fullscreen={fullscreen}
              activePageId={route.pageId}
              onPageChange={choosePage}
              back={() => navigate({})}
              dirtyChanged={updateDirty}
              onAuthenticationRequired={requireAuthentication}
              serverEvent={serverEvent}
            />
          ) : (
            <Dashboard
              open={(value) => navigate({ workspaceId: value.id })}
              serverEvent={serverEvent}
            />
          )}
        </>
      </div>
      {feedback && <Feedback close={() => setFeedback(false)} />}
      {profile && (
        <Profile
          user={user}
          changed={setUser}
          close={() => setProfile(false)}
        />
      )}
      {admin && (
        <Admin
          close={() => setAdmin(false)}
          self={user}
          unread={unreadFeedback.reports}
        />
      )}
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
