package feedback

import (
	"mindflow/internal/account"
	"net/http"
)

// HTTP supplies authenticated request handling and per-user abuse limits.
type HTTP interface {
	account.HTTP
	Limited(request *http.Request, key string) bool
}

// RegisterHTTP exposes private reports, discussion and explicit screenshot attachments.
func (a *Service) RegisterHTTP(routes HTTP) {
	routes.Endpoint("GET /api/v1/feedback", true, false, func(_ http.ResponseWriter, request *http.Request) (any, error) {
		return a.List(request.Context(), routes.CurrentUser(request))
	})
	routes.Endpoint("POST /api/v1/feedback", true, false, func(writer http.ResponseWriter, request *http.Request) (any, error) {
		actor := routes.CurrentUser(request)
		if routes.Limited(request, "feedback-create:"+actor.ID) {
			return nil, &Error{code: "rate_limited", message: "Please wait before sending another report."}
		}
		var command CreateCommand
		err := routes.Decode(writer, request, &command)
		if err != nil {
			return nil, err
		}

		return a.Create(request.Context(), actor, command)
	})
	routes.Endpoint("GET /api/v1/feedback/{id}", true, false, func(_ http.ResponseWriter, request *http.Request) (any, error) {
		return a.Find(request.Context(), routes.CurrentUser(request), request.PathValue("id"))
	})
	routes.Endpoint("POST /api/v1/feedback/{id}/comments", true, false, func(writer http.ResponseWriter, request *http.Request) (any, error) {
		actor := routes.CurrentUser(request)
		if routes.Limited(request, "feedback-comment:"+actor.ID) {
			return nil, &Error{code: "rate_limited", message: "Please wait before sending another comment."}
		}
		var command struct {
			Body string `json:"body"`
		}
		err := routes.Decode(writer, request, &command)
		if err != nil {
			return nil, err
		}

		return a.Comment(request.Context(), actor, request.PathValue("id"), command.Body)
	})
	routes.Endpoint("PATCH /api/v1/feedback/{id}", true, true, func(writer http.ResponseWriter, request *http.Request) (any, error) {
		var command ReviewCommand
		err := routes.Decode(writer, request, &command)
		if err != nil {
			return nil, err
		}

		return a.Review(request.Context(), routes.CurrentUser(request), request.PathValue("id"), command)
	})
	routes.Endpoint("GET /api/v1/feedback/{id}/screenshot", true, false, func(writer http.ResponseWriter, request *http.Request) (any, error) {
		content, err := a.Screenshot(request.Context(), routes.CurrentUser(request), request.PathValue("id"))
		if err != nil {
			return nil, err
		}
		writer.Header().Set("Content-Type", "image/png")
		_, err = writer.Write(content) //nolint:gosec // G705: stored screenshot bytes are decoded and re-encoded as PNG, served with image/png and nosniff.

		return nil, wrap("write feedback screenshot", err)
	})
}
