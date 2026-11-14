import { LockKeyhole } from "lucide-react";
import type { PageSession } from "../../session";

export default function PageControlStatus({
  session,
}: {
  session: PageSession;
}) {
  return (
    <>
      {session.loaded && !session.editable && (
        <div
          className="page-control-status"
          role="status"
          aria-label="Page access"
        >
          <LockKeyhole size={14} />
          <span>
            {session.canWrite
              ? "Page is read-only. Another session may have control."
              : `You have ${session.permission === "comment" ? "comment" : "read-only"} access to this workspace.`}
          </span>
          <button
            className="secondary compact"
            disabled={
              !!session.recovery ||
              !session.canWrite ||
              session.control?.status === "pending"
            }
            hidden={!session.canWrite}
            data-tooltip={
              session.recovery
                ? "Save or discard the local recovery copy before taking control."
                : undefined
            }
            onClick={() => void session.takeover()}
          >
            Take control
          </button>
          {session.permission === "owner" && (
            <button
              className="secondary compact"
              disabled={!!session.recovery}
              onClick={() => {
                if (
                  confirm(
                    "Force control immediately? The current editor will become read-only.",
                  )
                )
                  void session.takeover(true);
              }}
            >
              Force control
            </button>
          )}
        </div>
      )}
    </>
  );
}
