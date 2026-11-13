import { ArrowRight, ChevronLeft, ShieldCheck } from "lucide-react";
import type { FormEvent } from "react";
import { useState } from "react";
import type { Settings, User } from "../../api";
import { api, setCSRF } from "../../api";
import { Brand, ErrorMessage, Spinner } from "../../ui";
import SMTPFields from "./SMTPFields";
import NewPassword from "./NewPassword";
export default function Auth({
  setup = false,
  onLogin,
  overlay = false,
  expectedEmail,
}: {
  setup?: boolean;
  onLogin: (u: User) => void;
  overlay?: boolean;
  expectedEmail?: string;
}) {
  const initialHash = useState(() => {
    const h = location.hash.slice(1);
    if (/^(reset-password|accept-invite)\//.test(h)) {
      history.replaceState(null, "", location.pathname);
      return h;
    }
    return "";
  })[0];
  const [mode, setMode] = useState(
    setup ? "setup" : initialHash ? initialHash.split("/")[0] : "login",
  );
  const [email, setEmail] = useState(expectedEmail || "");
  const [displayName, setDisplayName] = useState("");
  const [password, setPassword] = useState("");
  const [credential, setCredential] = useState("");
  const [settings, setSettings] = useState<Settings>({
    publicURL: location.origin,
    smtp: {
      host: "",
      port: 587,
      mode: "starttls",
      username: "",
      password: "",
      from: "",
      name: "mindflow",
    },
  });
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const title =
    mode === "setup"
      ? "Make yourself at home."
      : mode === "forgot"
        ? "A fresh start."
        : mode === "accept-invite"
          ? "Your space is waiting."
          : mode === "reset-password"
            ? "Choose a new password."
            : overlay
              ? "Welcome back."
              : "Space to think.";
  async function submit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setMessage("");
    setBusy(true);
    try {
      if (mode === "setup") {
        await api("/setup/complete", "POST", {
          credential,
          displayName,
          email,
          password,
          settings,
        });
        setMode("login");
        setMessage("All set. Sign in to your new space.");
      } else if (mode === "forgot") {
        await api("/auth/forgot-password", "POST", { email });
        setMessage(
          "If an active account exists, a reset link will arrive by email.",
        );
      } else if (mode === "accept-invite" || mode === "reset-password") {
        await api(`/auth/${mode}`, "POST", {
          token: initialHash.split("/")[1],
          password,
          ...(mode === "accept-invite" ? { displayName } : {}),
        });
        setMode("login");
        setPassword("");
        setMessage("Your password is set. You can now sign in.");
      } else {
        const result = await api<{ user: User; csrf: string }>(
          "/auth/login",
          "POST",
          { email, password },
        );
        setCSRF(result.csrf);
        onLogin(result.user);
      }
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  async function testEmail() {
    setBusy(true);
    setError("");
    try {
      await api("/setup/test-email", "POST", { credential, email, settings });
      setMessage("Test email sent.");
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <main className={`auth-page ${overlay ? "auth-overlay" : ""}`}>
      <div className="auth-aside">
        <Brand />
        <div className="auth-art" aria-hidden="true">
          <div className="art-note note-one">
            A thought.
            <br />A starting point.
          </div>
          <div className="art-circle" />
          <div className="art-line" />
          <div className="art-note note-two">
            Room for
            <br />
            <em>what’s next.</em>
          </div>
          <span className="art-star">✳</span>
        </div>
        <p className="aside-caption">YOUR IDEAS, WITHOUT EDGES.</p>
      </div>
      <div className="auth-main">
        <div className={`auth-form ${setup ? "setup-form" : ""}`}>
          <span className="eyebrow">
            {mode === "setup"
              ? "WELCOME TO MINDFLOW"
              : "A LITTLE ROOM FOR YOUR THOUGHTS"}
          </span>
          <h1>{title}</h1>
          <p className="muted">
            {mode === "setup"
              ? "Create your administrator account and connect your email server."
              : mode === "forgot"
                ? "We’ll email you a link to reset your password."
                : mode.includes("invite")
                  ? "Set a password to accept your invitation."
                  : mode.includes("reset")
                    ? "Use at least 8 characters. A random passphrase works well."
                    : "Your ideas, sketches, and inspiration. All in one quiet place."}
          </p>
          <form onSubmit={submit}>
            {mode === "setup" && (
              <label>
                Setup credential
                <input
                  required
                  value={credential}
                  onChange={(e) => setCredential(e.target.value)}
                  autoComplete="off"
                  placeholder="From your server startup log"
                />
              </label>
            )}
            {!["accept-invite", "reset-password"].includes(mode) && (
              <label>
                Email address
                <input
                  type="email"
                  autoComplete="username"
                  readOnly={!!expectedEmail}
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="you@example.com"
                />
              </label>
            )}
            {(mode === "setup" || mode === "accept-invite") && (
              <label>
                Display name
                <input
                  autoComplete="name"
                  maxLength={80}
                  value={displayName}
                  onChange={(event) => setDisplayName(event.target.value)}
                  placeholder="How others see you"
                  disabled={busy}
                />
              </label>
            )}
            {mode !== "forgot" &&
              (mode === "login" ? (
                <label>
                  Password
                  <input
                    type="password"
                    required
                    maxLength={128}
                    autoComplete="current-password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="Your password"
                  />
                </label>
              ) : (
                <NewPassword
                  value={password}
                  onChange={setPassword}
                  disabled={busy}
                />
              ))}
            {mode === "setup" && (
              <>
                <hr />
                <h3>Email & server</h3>
                <SMTPFields value={settings} onChange={setSettings} />
                <button
                  type="button"
                  className="secondary"
                  disabled={busy}
                  onClick={testEmail}
                >
                  Send test email
                </button>
              </>
            )}
            <ErrorMessage message={error} />
            {message && (
              <p className="success" role="status">
                {message}
              </p>
            )}
            <button className="primary full" disabled={busy}>
              {busy ? (
                <Spinner />
              ) : (
                <>
                  {mode === "setup"
                    ? "Create mindflow"
                    : mode === "forgot"
                      ? "Send reset link"
                      : mode === "login"
                        ? "Sign in"
                        : "Set password"}
                  <ArrowRight size={17} />
                </>
              )}
            </button>
            {mode === "login" && (
              <button
                type="button"
                className="text-button full"
                onClick={() => {
                  setMode("forgot");
                  setError("");
                  setMessage("");
                }}
              >
                Forgot password?
              </button>
            )}
            {mode === "forgot" && (
              <button
                type="button"
                className="text-button full"
                onClick={() => {
                  setMode("login");
                  setError("");
                  setMessage("");
                }}
              >
                <ChevronLeft size={15} />
                Back to sign in
              </button>
            )}
          </form>
          <div className="auth-footer">
            <ShieldCheck size={15} />
            <span>Private by design. Yours to explore.</span>
          </div>
        </div>
      </div>
    </main>
  );
}
