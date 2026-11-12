package app

import (
	"fmt"
	"mindflow/internal/workspace"
	"net/http"
	"time"
)

type subscriber struct {
	owner  string
	events chan string
}

func (a *App) publish(owner, kind, resourceID string) {
	a.eventsMu.Lock()
	defer a.eventsMu.Unlock()
	for _, sub := range a.subscribers {
		if sub.owner == owner {
			select {
			case sub.events <- fmt.Sprintf(`{"type":%q,"id":%q}`, kind, resourceID):
			default:
			}
		}
	}
}

func (a *App) events(writer http.ResponseWriter, request *http.Request) (any, error) {
	flusher, ok := writer.(http.Flusher)
	if !ok {
		return nil, problem(http.StatusInternalServerError, "stream_unavailable", "Event stream unavailable.")
	}
	p := who(request)
	key := randomID()
	sub := subscriber{p.ID, make(chan string, eventQueueCapacity)}
	a.eventsMu.Lock()
	a.subscribers[key] = sub
	a.eventsMu.Unlock()
	defer func() { a.eventsMu.Lock(); delete(a.subscribers, key); a.eventsMu.Unlock() }()
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("X-Accel-Buffering", "no")
	_, _ = fmt.Fprint(writer, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()
	tick := time.NewTicker(eventHeartbeatSeconds * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-a.done:
			return responseWritten{}, nil
		case <-request.Context().Done():
			return responseWritten{}, nil
		case data := <-sub.events:
			authenticateErr := a.authenticate(request)
			if authenticateErr != nil {
				_, _ = fmt.Fprint(writer, "event: expired\ndata: {}\n\n")
				flusher.Flush()

				return responseWritten{}, nil
			}
			_, fprintfErr := fmt.Fprintf(writer, "data: %s\n\n", data)
			if fprintfErr != nil {
				return responseWritten{}, nil
			}
			flusher.Flush()
		case <-tick.C:
			authenticateErr2 := a.authenticate(request)
			if authenticateErr2 != nil {
				_, _ = fmt.Fprint(writer, "event: expired\ndata: {}\n\n")
				flusher.Flush()

				return responseWritten{}, nil
			}
			_, fprintErr := fmt.Fprint(writer, ": keepalive\n\n")
			if fprintErr != nil {
				return responseWritten{}, nil
			}
			flusher.Flush()
		}
	}
}

func (a *App) lease(w http.ResponseWriter, request *http.Request) (any, error) {
	var value workspace.LeaseCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.workspace.Lease(request.Context(), who(request), request.PathValue("id"), value)
}

func (a *App) leaseStatus(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.workspace.LeaseStatus(r.Context(), who(r), r.PathValue("id"), r.URL.Query().Get("clientId"))
}
