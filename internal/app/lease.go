package app

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

type leaseResult struct {
	Editable  bool  `json:"editable"`
	Epoch     int64 `json:"epoch"`
	ExpiresAt int64 `json:"expiresAt"`
	Revision  int   `json:"revision"`
}

func (a *App) lease(w http.ResponseWriter, r *http.Request) (any, error) {
	var v struct {
		ClientID string `json:"clientId"`
		Action   string `json:"action"`
		Epoch    int64  `json:"epoch"`
	}
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	if len(v.ClientID) < 8 || len(v.ClientID) > 128 {
		return nil, bad("Invalid editor identity.")
	}
	if v.Action != "acquire" && v.Action != "takeover" && v.Action != "renew" && v.Action != "release" {
		return nil, bad("Invalid editor action.")
	}
	p := who(r)
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	page, err := ownedPage(tx, r.PathValue("id"), p.ID)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec("INSERT OR IGNORE INTO page_leases(page_id) VALUES(?)", page.ID); err != nil {
		return nil, err
	}
	var session sql.NullString
	var client string
	var epoch, expires int64
	if err = tx.QueryRow("SELECT session_id,client_id,epoch,expires_at FROM page_leases WHERE page_id=?", page.ID).Scan(&session, &client, &epoch, &expires); err != nil {
		return nil, err
	}
	mine := session.String == p.SessionID && client == v.ClientID && expires > now()
	changed := false
	switch v.Action {
	case "release":
		if mine && epoch == v.Epoch {
			expires = 0
			epoch++
			session = sql.NullString{}
			client = ""
			changed = true
		}
	case "renew":
		if !mine || epoch != v.Epoch {
			return nil, problem(409, "lease_lost", "Editor control expired or was taken by another session.")
		}
		expires = now() + 60
		changed = true
	case "takeover":
		epoch++
		expires = now() + 60
		session = sql.NullString{String: p.SessionID, Valid: true}
		client = v.ClientID
		mine = true
		changed = true
	case "acquire":
		if !mine && (expires <= now() || !session.Valid) {
			epoch++
			expires = now() + 60
			session = sql.NullString{String: p.SessionID, Valid: true}
			client = v.ClientID
			mine = true
			changed = true
		} else if mine {
			expires = now() + 60
			changed = true
		}
	}
	if changed {
		if _, err = tx.Exec("UPDATE page_leases SET session_id=?,client_id=?,epoch=?,expires_at=? WHERE page_id=?", session, client, epoch, expires, page.ID); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	if changed && v.Action != "renew" {
		a.publish(p.ID, "lease", page.ID)
	}
	return leaseResult{Editable: session.String == p.SessionID && client == v.ClientID && expires > now(), Epoch: epoch, ExpiresAt: expires, Revision: page.Revision}, nil
}
func (a *App) leaseStatus(w http.ResponseWriter, r *http.Request) (any, error) {
	var rev int
	if err := a.DB.QueryRow("SELECT p.revision FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?", r.PathValue("id"), who(r).ID).Scan(&rev); err != nil {
		return nil, err
	}
	var session sql.NullString
	var client string
	var epoch, expires int64
	err := a.DB.QueryRow("SELECT session_id,client_id,epoch,expires_at FROM page_leases WHERE page_id=?", r.PathValue("id")).Scan(&session, &client, &epoch, &expires)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return leaseResult{Editable: session.String == who(r).SessionID && client == r.URL.Query().Get("clientId") && expires > now(), Epoch: epoch, ExpiresAt: expires, Revision: rev}, nil
}
func (a *App) publish(owner, kind, id string) {
	a.eventsMu.Lock()
	defer a.eventsMu.Unlock()
	for _, sub := range a.subscribers {
		if sub.owner == owner {
			select {
			case sub.events <- fmt.Sprintf(`{"type":%q,"id":%q}`, kind, id):
			default:
			}
		}
	}
}

type subscriber struct {
	owner  string
	events chan string
}

func (a *App) events(w http.ResponseWriter, r *http.Request) (any, error) {
	f, ok := w.(http.Flusher)
	if !ok {
		return nil, problem(500, "stream_unavailable", "Event stream unavailable.")
	}
	p := who(r)
	key := randomID()
	sub := subscriber{p.ID, make(chan string, 16)}
	a.eventsMu.Lock()
	a.subscribers[key] = sub
	a.eventsMu.Unlock()
	defer func() { a.eventsMu.Lock(); delete(a.subscribers, key); a.eventsMu.Unlock() }()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	_, _ = fmt.Fprint(w, "data: {\"type\":\"connected\"}\n\n")
	f.Flush()
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-a.done:
			return nil, nil
		case <-r.Context().Done():
			return nil, nil
		case data := <-sub.events:
			if _, err := a.authenticate(r); err != nil {
				_, _ = fmt.Fprint(w, "event: expired\ndata: {}\n\n")
				f.Flush()
				return nil, nil
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return nil, nil
			}
			f.Flush()
		case <-tick.C:
			if _, err := a.authenticate(r); err != nil {
				_, _ = fmt.Fprint(w, "event: expired\ndata: {}\n\n")
				f.Flush()
				return nil, nil
			}
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return nil, nil
			}
			f.Flush()
		}
	}
}
