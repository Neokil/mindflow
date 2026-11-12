package notification

import (
	"context"
	"fmt"
	"mindflow/internal/session"
	"net/http"
	"sync"
)

type subscriber struct {
	owner  string
	events chan string
}

// Authenticator checks whether an event-stream session remains valid.
type Authenticator interface {
	Authenticate(ctx context.Context, cookie string) (session.Principal, error)
}

// Service owns realtime notifications and their active subscribers.
type Service struct {
	done           chan struct{}
	closeOnce      sync.Once
	eventsMu       sync.Mutex
	subscribers    map[string]subscriber
	authentication Authenticator
}

const (
	eventQueueCapacity    = 16
	eventHeartbeatSeconds = 15
)

// NewService constructs the owner-scoped notification broker.
func NewService(authentication Authenticator) *Service {
	return &Service{done: make(chan struct{}), subscribers: make(map[string]subscriber), authentication: authentication}
}

// Publish sends an owner-scoped domain event.
func (a *Service) Publish(owner, kind, resourceID string) {
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

// Close ends active event streams during server shutdown.
func (a *Service) Close() { a.closeOnce.Do(func() { close(a.done) }) }

func (a *Service) authenticate(request *http.Request) error {
	cookie, err := request.Cookie("mindflow_session")
	if err != nil {
		return failure("session_expired", "Please sign in.")
	}
	_, err = a.authentication.Authenticate(request.Context(), cookie.Value)

	return err
}
