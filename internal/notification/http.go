package notification

import (
	"fmt"
	"mindflow/internal/account/session"
	"net/http"
	"time"
)

// HTTP is the request boundary needed by this domain's routes.
type HTTP interface {
	Endpoint(pattern string, authenticated bool, administrator bool, handler func(http.ResponseWriter, *http.Request) (any, error))
	CurrentSession(request *http.Request) session.Principal
}
type httpHandler struct {
	service *Service
	routes  HTTP
}

// RegisterHTTP registers routes owned by this domain.
func (a *Service) RegisterHTTP(routes HTTP) {
	handler := &httpHandler{service: a, routes: routes}
	routes.Endpoint("GET /api/v1/events", true, false, handler.events)
}

func (a *httpHandler) events(writer http.ResponseWriter, request *http.Request) (any, error) {
	flusher, ok := writer.(http.Flusher)
	if !ok {
		return nil, failure("stream_unavailable", "Event stream unavailable.")
	}
	p := a.routes.CurrentSession(request)
	key := newID()
	sub := subscriber{p.ID, make(chan string, eventQueueCapacity)}
	a.service.eventsMu.Lock()
	a.service.subscribers[key] = sub
	a.service.eventsMu.Unlock()
	defer func() { a.service.eventsMu.Lock(); delete(a.service.subscribers, key); a.service.eventsMu.Unlock() }()
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("X-Accel-Buffering", "no")
	_, _ = fmt.Fprint(writer, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()
	tick := time.NewTicker(eventHeartbeatSeconds * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-a.service.done:
			return responseWritten{}, nil
		case <-request.Context().Done():
			return responseWritten{}, nil
		case data := <-sub.events:
			authenticateErr := a.service.authenticate(request)
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
			authenticateErr2 := a.service.authenticate(request)
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

type responseWritten struct{}

// ResponseWritten marks a body already written by the domain handler.
func (responseWritten) ResponseWritten() {}
