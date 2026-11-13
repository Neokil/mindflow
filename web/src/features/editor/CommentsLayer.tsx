import Avatar from "../accounts/Avatar";
import { useCallback, useEffect, useState } from "react";
import { Check, MessageCircle, Trash2, X } from "lucide-react";
import { api } from "../../api";
import type { Point } from "../../model";
import type { Camera } from "./types";
import { ErrorMessage } from "../../ui";

type Message = {
  id: string;
  authorEmail: string;
  authorDisplayName: string;
  authorId: string;
  avatarVersion?: string;
  body: string;
  createdAt: number;
};
type Thread = {
  id: string;
  initials: string;
  authorEmail: string;
  authorDisplayName: string;
  authorId: string;
  avatarVersion?: string;
  x: number;
  y: number;
  resolved: boolean;
  canManage: boolean;
  messages: Message[];
};
export default function CommentsLayer({
  pageId,
  camera,
  size,
  canComment,
  refresh,
  draftPoint,
  closeDraft,
}: {
  pageId: string;
  camera: Camera;
  size: { width: number; height: number };
  canComment: boolean;
  refresh: number;
  draftPoint?: Point;
  closeDraft: () => void;
}) {
  const [threads, setThreads] = useState<Thread[]>([]),
    [open, setOpen] = useState<string>(),
    [showResolved, setShowResolved] = useState(false),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const load = useCallback(async () => {
    setThreads(await api<Thread[]>(`/pages/${pageId}/comments`));
  }, [pageId]);
  useEffect(() => {
    let canceled = false;
    const refreshThreads = () =>
      void api<Thread[]>(`/pages/${pageId}/comments`)
        .then((value) => {
          if (!canceled) setThreads(value);
        })
        .catch(() => {
          if (!canceled) setThreads([]);
        });
    refreshThreads();
    const timer = setInterval(refreshThreads, 5000);
    return () => {
      canceled = true;
      clearInterval(timer);
    };
  }, [pageId, refresh]);
  async function mutate(action: () => Promise<unknown>) {
    setError("");
    setBusy(true);
    try {
      await action();
      await load();
      return true;
    } catch (err) {
      setError((err as Error).message);
      return false;
    } finally {
      setBusy(false);
    }
  }
  const position = (point: Point) => ({
    left: point.x * camera.zoom + camera.x,
    top: point.y * camera.zoom + camera.y,
  });
  const popupPosition = (point: Point) => ({
    left: point.x * camera.zoom + camera.x + 340 > size.width ? -298 : 28,
    top: Math.min(0, size.height - 390 - (point.y * camera.zoom + camera.y)),
  });
  return (
    <div className="comments-layer">
      {threads.some((thread) => thread.resolved) && (
        <button
          className="resolved-comments secondary compact"
          aria-pressed={showResolved}
          onClick={() => setShowResolved(!showResolved)}
        >
          {showResolved ? "Hide resolved comments" : "Show resolved comments"}
        </button>
      )}
      {threads
        .filter((thread) => !thread.resolved || showResolved)
        .map((thread) => (
          <div
            className="comment-anchor"
            key={thread.id}
            style={position(thread)}
            onMouseEnter={() => setOpen(thread.id)}
            onMouseLeave={(event) => {
              if (!event.currentTarget.contains(document.activeElement))
                setOpen(undefined);
            }}
          >
            <button
              className={`comment-pin ${thread.resolved ? "resolved" : ""}`}
              aria-label={`Comment by ${thread.authorDisplayName}`}
              aria-expanded={open === thread.id}
              onFocus={() => setOpen(thread.id)}
              onClick={() => setOpen(thread.id)}
            >
              {thread.avatarVersion ? (
                <Avatar
                  id={thread.authorId}
                  email={thread.authorEmail}
                  name={thread.authorDisplayName}
                  version={thread.avatarVersion}
                />
              ) : (
                thread.initials
              )}
            </button>
            {open === thread.id && (
              <section
                className="comment-popover"
                style={popupPosition(thread)}
                role="dialog"
                aria-label={`Comment thread by ${thread.authorEmail}`}
              >
                <div className="comment-heading">
                  <strong>
                    {thread.resolved ? "Resolved comment" : "Comment"}
                  </strong>
                  <button
                    className="icon-button"
                    aria-label="Close comment"
                    onClick={() => setOpen(undefined)}
                  >
                    <X size={14} />
                  </button>
                </div>
                <div className="comment-messages">
                  {thread.messages.map((message) => (
                    <div className="comment-message" key={message.id}>
                      <strong className="profile-identity">
                        <Avatar
                          id={message.authorId}
                          email={message.authorDisplayName}
                          name={message.authorDisplayName}
                          version={message.avatarVersion}
                        />
                        {message.authorDisplayName}
                      </strong>
                      <p>{message.body}</p>
                    </div>
                  ))}
                </div>
                {canComment && !thread.resolved && (
                  <CommentForm
                    label="Reply to comment"
                    submit="Reply"
                    busy={busy}
                    onSubmit={(body) =>
                      mutate(() =>
                        api(`/comments/${thread.id}/replies`, "POST", { body }),
                      )
                    }
                  />
                )}
                {thread.canManage && (
                  <div className="comment-actions">
                    <button
                      className="text-button"
                      disabled={busy}
                      onClick={() =>
                        void mutate(() =>
                          api(`/comments/${thread.id}`, "PATCH", {
                            resolved: !thread.resolved,
                          }),
                        )
                      }
                    >
                      <Check size={14} />
                      {thread.resolved ? "Reopen" : "Resolve"}
                    </button>
                    <button
                      className="text-button danger"
                      disabled={busy}
                      aria-label="Delete comment thread"
                      onClick={() => {
                        if (confirm("Delete this comment and all its replies?"))
                          void mutate(() =>
                            api(`/comments/${thread.id}`, "DELETE", {}),
                          );
                      }}
                    >
                      <Trash2 size={14} />
                      Delete
                    </button>
                  </div>
                )}
                <ErrorMessage message={error} />
              </section>
            )}
          </div>
        ))}
      {draftPoint && canComment && (
        <div className="comment-anchor" style={position(draftPoint)}>
          <span className="comment-pin">
            <MessageCircle size={16} />
          </span>
          <section
            className="comment-popover"
            style={popupPosition(draftPoint)}
            role="dialog"
            aria-label="New comment"
          >
            <div className="comment-heading">
              <strong>New comment</strong>
              <button
                className="icon-button"
                aria-label="Cancel comment"
                onClick={closeDraft}
              >
                <X size={14} />
              </button>
            </div>
            <CommentForm
              label="Comment text"
              submit="Post comment"
              busy={busy}
              onSubmit={async (body) => {
                const saved = await mutate(() =>
                  api(`/pages/${pageId}/comments`, "POST", {
                    ...draftPoint,
                    body,
                  }),
                );
                if (saved) closeDraft();
                return saved;
              }}
            />
            <ErrorMessage message={error} />
          </section>
        </div>
      )}
    </div>
  );
}
function CommentForm({
  label,
  submit,
  busy,
  onSubmit,
}: {
  label: string;
  submit: string;
  busy: boolean;
  onSubmit: (body: string) => Promise<boolean>;
}) {
  const [body, setBody] = useState("");
  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        void onSubmit(body).then((saved) => {
          if (saved) setBody("");
        });
      }}
    >
      <textarea
        autoFocus={label === "Comment text"}
        aria-label={label}
        placeholder={label}
        maxLength={10000}
        value={body}
        onChange={(event) => setBody(event.target.value)}
        disabled={busy}
        required
      />
      <button className="primary compact" disabled={busy || !body.trim()}>
        {submit}
      </button>
    </form>
  );
}
