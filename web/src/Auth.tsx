import { useState } from "react";
import type { FormEvent } from "react";
import { ArrowRight, ChevronLeft, ShieldCheck } from "lucide-react";
import { api, setCSRF } from "./api";
import type { User, Settings } from "./api";
import { Brand, Spinner, ErrorMessage } from "./ui";
export function SMTPFields({
  value,
  onChange,
}: {
  value: Settings;
  onChange: (s: Settings) => void;
}) {
  const update = (key: string, v: string | number) =>
    onChange({ ...value, smtp: { ...value.smtp, [key]: v } });
  return (
    <>
      <label>
        Public application URL
        <input
          type="url"
          required
          value={value.publicURL}
          onChange={(e) => onChange({ ...value, publicURL: e.target.value })}
          placeholder="https://mindflow.example.com"
        />
      </label>
      <div className="form-row">
        <label>
          SMTP host
          <input
            required
            value={value.smtp.host}
            onChange={(e) => update("host", e.target.value)}
            placeholder="smtp.example.com"
          />
        </label>
        <label className="short">
          Port
          <input
            type="number"
            min="1"
            max="65535"
            required
            value={value.smtp.port}
            onChange={(e) => update("port", +e.target.value)}
          />
        </label>
      </div>
      <label>
        Connection security
        <select
          value={value.smtp.mode}
          onChange={(e) => update("mode", e.target.value)}
        >
          <option value="starttls">STARTTLS</option>
          <option value="tls">Implicit TLS</option>
        </select>
      </label>
      <div className="form-row">
        <label>
          SMTP username
          <input
            autoComplete="off"
            value={value.smtp.username}
            onChange={(e) => update("username", e.target.value)}
          />
        </label>
        <label>
          SMTP password
          <input
            type="password"
            autoComplete="new-password"
            value={value.smtp.password || ""}
            onChange={(e) => update("password", e.target.value)}
            placeholder="Leave blank to keep existing"
          />
        </label>
      </div>
      <div className="form-row">
        <label>
          Sender email
          <input
            type="email"
            required
            value={value.smtp.from}
            onChange={(e) => update("from", e.target.value)}
          />
        </label>
        <label>
          Sender name
          <input
            value={value.smtp.name}
            onChange={(e) => update("name", e.target.value)}
          />
        </label>
      </div>
    </>
  );
}
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
                    ? "Use at least 15 characters. A passphrase works well."
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
            {mode !== "forgot" && (
              <label>
                Password
                <input
                  type="password"
                  required
                  minLength={mode === "login" ? undefined : 15}
                  maxLength={128}
                  autoComplete={
                    mode === "login" ? "current-password" : "new-password"
                  }
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder={
                    mode === "login"
                      ? "Your password"
                      : "At least 15 characters"
                  }
                />
              </label>
            )}
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
