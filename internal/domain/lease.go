package domain

const LeaseSeconds int64 = 60

type LeaseState struct {
	SessionID, ClientID string
	Epoch, ExpiresAt    int64
}
type LeaseResult struct {
	Editable  bool  `json:"editable"`
	Epoch     int64 `json:"epoch"`
	ExpiresAt int64 `json:"expiresAt"`
	Revision  int   `json:"revision"`
}

func (l LeaseState) CanEdit(sessionID, clientID string, at int64) bool {
	return l.SessionID != "" && l.SessionID == sessionID && l.ClientID == clientID && l.ExpiresAt > at
}

// Advance applies a user's explicit control action. Persistence must save the
// resulting state atomically with its ownership check.
func (l LeaseState) Advance(sessionID string, command LeaseCommand, at int64) (LeaseState, bool, error) {
	if len(command.ClientID) < 8 || len(command.ClientID) > 128 {
		return l, false, Invalid("Invalid editor identity.")
	}
	mine := l.CanEdit(sessionID, command.ClientID, at)
	switch command.Action {
	case "release":
		if mine && l.Epoch == command.Epoch {
			l.SessionID = ""
			l.ClientID = ""
			l.ExpiresAt = 0
			l.Epoch++
			return l, true, nil
		}
	case "renew":
		if !mine || l.Epoch != command.Epoch {
			return l, false, Fault("lease_lost", "Editor control expired or was taken by another session.")
		}
		l.ExpiresAt = at + LeaseSeconds
		return l, true, nil
	case "takeover":
		l.Epoch++
		l.SessionID = sessionID
		l.ClientID = command.ClientID
		l.ExpiresAt = at + LeaseSeconds
		return l, true, nil
	case "acquire":
		if !mine && (l.ExpiresAt <= at || l.SessionID == "") {
			l.Epoch++
			l.SessionID = sessionID
			l.ClientID = command.ClientID
			l.ExpiresAt = at + LeaseSeconds
			return l, true, nil
		}
		if mine {
			l.ExpiresAt = at + LeaseSeconds
			return l, true, nil
		}
	default:
		return l, false, Invalid("Invalid editor action.")
	}
	return l, false, nil
}
func (l LeaseState) CheckSave(sessionID, clientID string, epoch int64, expectedRevision, revision int, at int64) error {
	if !l.CanEdit(sessionID, clientID, at) || l.Epoch != epoch {
		return Fault("lease_lost", "This page is now read-only. Save your local work as a separate page.")
	}
	if revision != expectedRevision {
		return Fault("revision_conflict", "The page changed in another session. Save your local work as a separate page.")
	}
	return nil
}
