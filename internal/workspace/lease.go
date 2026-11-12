package workspace

import "mindflow/internal/fault"

// LeaseSeconds is the lifetime renewed by an active editor.
const LeaseSeconds int64 = 60

// LeaseState tracks the session, fencing epoch and expiry of editor control.
type LeaseState struct {
	SessionID, ClientID string
	Epoch, ExpiresAt    int64
}

// LeaseResult reports control eligibility and the current page revision.
type LeaseResult struct {
	Editable  bool  `json:"editable"`
	Epoch     int64 `json:"epoch"`
	ExpiresAt int64 `json:"expiresAt"`
	Revision  int   `json:"revision"`
}

// CanEdit checks whether a session still holds a live editor lease.
func (l LeaseState) CanEdit(sessionID, clientID string, at int64) bool {
	return l.SessionID != "" && l.SessionID == sessionID && l.ClientID == clientID && l.ExpiresAt > at
}

// Advance applies a user's explicit control action. Persistence must save the
// resulting state atomically with its ownership check.
func (l LeaseState) Advance(sessionID string, command LeaseCommand, atValue int64) (LeaseState, bool, error) {
	if len(command.ClientID) < 8 || len(command.ClientID) > maxIdentifierLength {
		return l, false, fault.Invalid("Invalid editor identity.")
	}
	mine := l.CanEdit(sessionID, command.ClientID, atValue)
	switch command.Action {
	case "release":
		if mine && l.Epoch == command.Epoch {
			l.SessionID = ""
			l.ClientID = ""
			l.ExpiresAt = 0
			l.Epoch++

			return l, true, nil
		}
	case leaseRenew:
		if !mine || l.Epoch != command.Epoch {
			return l, false, fault.Fault("lease_lost", "Editor control expired or was taken by another session.")
		}
		l.ExpiresAt = atValue + LeaseSeconds

		return l, true, nil
	case "takeover":
		l.Epoch++
		l.SessionID = sessionID
		l.ClientID = command.ClientID
		l.ExpiresAt = atValue + LeaseSeconds

		return l, true, nil
	case "acquire":
		if !mine && (l.ExpiresAt <= atValue || l.SessionID == "") {
			l.Epoch++
			l.SessionID = sessionID
			l.ClientID = command.ClientID
			l.ExpiresAt = atValue + LeaseSeconds

			return l, true, nil
		}
		if mine {
			l.ExpiresAt = atValue + LeaseSeconds

			return l, true, nil
		}
	default:
		return l, false, fault.Invalid("Invalid editor action.")
	}

	return l, false, nil
}

// CheckSave rejects expired control, stale fencing epochs and changed revisions.
func (l LeaseState) CheckSave(sessionID, clientID string, epoch int64, expectedRevision, revision int, at int64) error {
	if !l.CanEdit(sessionID, clientID, at) || l.Epoch != epoch {
		return fault.Fault("lease_lost", "This page is identifier.Now read-only. Save your local work as a separate page.")
	}
	if revision != expectedRevision {
		return fault.Fault("revision_conflict", "The page changed in another session. Save your local work as a separate page.")
	}

	return nil
}
