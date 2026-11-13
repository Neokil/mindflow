import { useEffect, useState } from "react";
export default function UpdateBanner() {
  const [available, setAvailable] = useState(false);
  useEffect(() => {
    const current = document.querySelector<HTMLMetaElement>(
      'meta[name="mindflow-version"]',
    )?.content;
    if (!current) return;
    let stopped = false,
      busy = false;
    const check = async () => {
      if (busy || stopped || document.visibilityState === "hidden") return;
      busy = true;
      try {
        const response = await fetch("/api/v1/version", {
          cache: "no-store",
          credentials: "same-origin",
        });
        if (response.ok) {
          const latest = await response.json();
          if (
            !stopped &&
            typeof latest.version === "string" &&
            latest.version !== current
          )
            setAvailable(true);
        }
      } catch {
        /* Retry on the next poll or foreground event. */
      } finally {
        busy = false;
      }
    };
    void check();
    const timer = setInterval(() => void check(), 60000);
    const foreground = () => void check();
    window.addEventListener("focus", foreground);
    document.addEventListener("visibilitychange", foreground);
    return () => {
      stopped = true;
      clearInterval(timer);
      window.removeEventListener("focus", foreground);
      document.removeEventListener("visibilitychange", foreground);
    };
  }, []);
  return available ? (
    <div className="update-banner" role="status">
      <span>A new version of mindflow is available. Reload to update.</span>
      <button className="secondary compact" onClick={() => location.reload()}>
        Reload
      </button>
    </div>
  ) : null;
}
