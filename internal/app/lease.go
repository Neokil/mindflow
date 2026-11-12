package app

import (
	"fmt"
	"net/http"
	"time"

	"mindflow/internal/domain"
)

type subscriber struct {
	owner  string
	events chan string
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
func (a *App) lease(w http.ResponseWriter, r *http.Request) (any, error) {
	var v domain.LeaseCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.Lease(r.Context(), who(r), r.PathValue("id"), v)
}
func (a *App) leaseStatus(w http.ResponseWriter, r *http.Request) (any, error) {

	return a.service.LeaseStatus(r.Context(), who(r), r.PathValue("id"), r.URL.Query().Get("clientId"))
}
