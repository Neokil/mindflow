import Avatar from "../accounts/Avatar";
import { useEffect, useState } from "react";
import type { PageSession } from "../../session";
import { Modal } from "../../ui";
export default function ControlRequestDialog({
  session,
}: {
  session: PageSession;
}) {
  const [now, setNow] = useState(Date.now()),
    [busy, setBusy] = useState(false);
  const request = session.control;
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 100);
    return () => clearInterval(timer);
  }, []);
  if (
    !request ||
    request.status === "accepted" ||
    request.status === "cancelled"
  )
    return null;
  const seconds = Math.max(
    0,
    Math.ceil((session.controlDeadline - now) / 1000),
  );
  async function decide(action: "accept" | "deny" | "cancel") {
    setBusy(true);
    try {
      await session.decideControl(action);
    } finally {
      setBusy(false);
    }
  }
  if (request.status === "denied")
    return (
      <Modal
        title="Control request denied"
        close={() => session.dismissControl()}
      >
        <p>
          The current editor declined your request. You can try again later.
        </p>
        <button
          className="primary full"
          onClick={() => session.dismissControl()}
        >
          OK
        </button>
      </Modal>
    );
  return (
    <Modal
      title={
        request.incoming ? (
          <>
            <Avatar
              id={request.requesterId}
              email={request.requesterEmail}
              name={request.requesterDisplayName}
              version={request.avatarVersion}
            />{" "}
            {request.requesterDisplayName} is requesting session control
          </>
        ) : (
          "Requesting session control"
        )
      }
      close={() => void decide(request.incoming ? "deny" : "cancel")}
    >
      <progress
        className="control-progress"
        aria-label="Control request countdown"
        max={10}
        value={Math.min(
          10,
          Math.max(0, 10 - (session.controlDeadline - now) / 1000),
        )}
      />
      <p>
        {request.incoming
          ? "Accept to make this session read-only, or deny to keep editing. This request is automatically accepted after 10 seconds."
          : "The current editor has been asked to confirm the operation. Control transfers automatically if they do not respond within 10 seconds."}
      </p>
      <p role="timer">
        {seconds > 0
          ? `${seconds} seconds remaining`
          : "Waiting for server confirmation…"}
      </p>
      <div className="dialog-actions">
        {request.incoming ? (
          <>
            <button
              className="secondary"
              disabled={busy || seconds === 0}
              onClick={() => void decide("deny")}
            >
              Deny
            </button>
            <button
              className="primary"
              disabled={busy || seconds === 0}
              onClick={() => void decide("accept")}
            >
              Accept ({seconds})
            </button>
          </>
        ) : (
          <button
            className="secondary"
            disabled={busy}
            onClick={() => void decide("cancel")}
          >
            Cancel request
          </button>
        )}
      </div>
    </Modal>
  );
}
