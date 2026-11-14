import { useEffect, useState } from "react";
import { api } from "../../api";
type Counts = { reports: number; replies: number };
const changedEvent = "mindflow-feedback-changed";
export function feedbackChanged() {
  window.dispatchEvent(new Event(changedEvent));
}
export default function useFeedbackUnread(userID?: string) {
  const [counts, setCounts] = useState<Counts & { reader?: string }>({
    reports: 0,
    replies: 0,
  });
  useEffect(() => {
    if (!userID) return;
    let disposed = false,
      sequence = 0;
    const refresh = () => {
      if (document.visibilityState === "hidden") return;
      const request = ++sequence;
      void api<Counts>("/feedback/unread")
        .then((value) => {
          if (!disposed && request === sequence)
            setCounts({ ...value, reader: userID });
        })
        .catch(() => {});
    };
    refresh();
    const timer = setInterval(refresh, 10000);
    window.addEventListener(changedEvent, refresh);
    window.addEventListener("focus", refresh);
    document.addEventListener("visibilitychange", refresh);
    return () => {
      disposed = true;
      clearInterval(timer);
      window.removeEventListener(changedEvent, refresh);
      window.removeEventListener("focus", refresh);
      document.removeEventListener("visibilitychange", refresh);
    };
  }, [userID]);
  return userID && counts.reader === userID
    ? counts
    : { reports: 0, replies: 0 };
}
