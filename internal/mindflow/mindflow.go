package mindflow

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"mindflow/internal/account"
	"mindflow/internal/account/authentication"
	"mindflow/internal/account/avatar"
	"mindflow/internal/account/invitation"
	"mindflow/internal/account/passwordreset"
	"mindflow/internal/account/session"
	"mindflow/internal/email"
	"mindflow/internal/installation"
	"mindflow/internal/mindflow/storage"
	"mindflow/internal/notification"
	"mindflow/internal/workspace"
	"mindflow/internal/workspace/page"
	"mindflow/internal/workspace/page/canvas"
	"mindflow/internal/workspace/page/comment"
	"mindflow/internal/workspace/page/editor"
	"mindflow/internal/workspace/sharing"
	"net/http"
	"strings"
	"sync"
	"time"

	imageasset "mindflow/internal/workspace/image"

	web "mindflow/web"
)

// Config supplies development mode and the one-time setup credential.
type Config struct {
	Now       func() time.Time
	Dev       bool
	Bootstrap string
}

// App serves the HTTP API and coordinates domain services.
type App struct {
	Log            *slog.Logger
	limitMu        sync.Mutex
	limits         map[string]rateWindow
	authentication *authentication.Service
	notifications  *notification.Service
	accounts       *account.Service
	sessions       *session.Service
	installation   *installation.Service
	invitations    *invitation.Service
	passwordResets *passwordreset.Service
	mail           *email.Service
	workspace      *workspace.Service
	sharing        *sharing.Service
	comments       *comment.Service
	pages          *page.Service
	canvases       *canvas.Service
	editors        *editor.Service
	images         *imageasset.Service
	Config         Config
	Mux            *http.ServeMux
}

// New composes domain services and registers HTTP routes.
func New(ctx context.Context, data *storage.Database, config Config) (*App, error) {
	service := &App{Config: config, Mux: http.NewServeMux(), Log: slog.Default(), limits: make(map[string]rateWindow)}
	accounts, err := account.NewService(data.Accounts, data.Email, data.Workspaces)
	if err != nil {
		return nil, err
	}
	service.accounts = accounts
	service.sessions = session.NewService(data.Accounts)
	service.installation = installation.NewService(data.Installation, accounts, installation.Config{Dev: config.Dev, Bootstrap: config.Bootstrap})
	service.mail = email.NewService(data.Email, service.installation, linkAuthorization{data: data})
	service.invitations = invitation.NewService(data.Accounts, accounts, service.mail, service.installation)
	service.passwordResets = passwordreset.NewService(data.Accounts, accounts, service.mail, service.installation)
	service.authentication = authentication.NewService(accounts, service.sessions, config.Dev)
	service.notifications = notification.NewService(service.authentication)
	publish := func(userID, kind, resourceID string) {
		service.notifications.Publish(userID, kind, resourceID)
		recipients, audienceErr := data.Workspaces.EventAudience(ctx, kind, resourceID)
		if audienceErr != nil {
			service.Log.Error("resolve workspace event recipients", "error", audienceErr)

			return
		}
		for _, recipient := range recipients {
			if recipient != userID {
				service.notifications.Publish(recipient, kind, resourceID)
			}
		}
	}
	service.sharing = sharing.NewService(data.Workspaces, accounts, publish)
	service.comments = comment.NewService(data.Workspaces, accounts, publish)
	service.images = imageasset.NewService(data.Workspaces, publish)
	service.editors, err = editor.NewService(ctx, data.Workspaces, publish, service.sessions, config.Now)
	if err != nil {
		return nil, err
	}
	service.canvases = canvas.NewService(data.Workspaces, service.images, service.editors, data.Workspaces, publish)
	service.pages = page.NewService(data.Workspaces, service.canvases, service.images, publish)
	service.workspace = workspace.NewService(data.Workspaces, service.pages, service.images, publish)
	avatar.NewService(data.Accounts).RegisterHTTP(service)
	service.accounts.RegisterHTTP(service)
	service.authentication.RegisterHTTP(service)
	service.installation.RegisterHTTP(service)
	service.invitations.RegisterHTTP(service)
	service.passwordResets.RegisterHTTP(service)
	service.mail.RegisterHTTP(service)
	service.workspace.RegisterHTTP(service)
	service.sharing.RegisterHTTP(service)
	service.comments.RegisterHTTP(service)
	service.pages.RegisterHTTP(service)
	service.canvases.RegisterHTTP(service)
	service.editors.RegisterHTTP(service)
	service.images.RegisterHTTP(service)
	service.notifications.RegisterHTTP(service)

	service.Mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, map[string]bool{"ok": true}) })
	service.Mux.HandleFunc("GET /health/ready", func(writer http.ResponseWriter, r *http.Request) {
		if data.Ping(r.Context()) != nil {
			writeJSON(writer, http.StatusServiceUnavailable, map[string]bool{"ok": false})

			return
		}
		writeJSON(writer, http.StatusOK, map[string]bool{"ok": true})
	})
	assets, _ := fs.Sub(web.Files, "dist")
	service.Mux.HandleFunc("GET /", func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/api/") {
			http.NotFound(writer, request)

			return
		}
		assetPath := strings.TrimPrefix(request.URL.Path, "/")
		if assetPath == "" {
			assetPath = "index.html"
		}
		_, statErr := fs.Stat(assets, assetPath)
		if statErr != nil {
			if strings.Contains(assetPath, ".") {
				http.NotFound(writer, request)

				return
			}
			request.URL.Path = "/index.html"
		}
		http.FileServer(http.FS(assets)).ServeHTTP(writer, request)
	})

	return service, nil
}

// ServeHTTP sets common security headers before routing the request.
func (a *App) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// Network writers support deadlines; in-memory test writers may not.
	_ = http.NewResponseController(writer).SetWriteDeadline(time.Now().Add(time.Minute))
	request.Header.Set("X-Request-ID", randomID()[:eventQueueCapacity])
	writer.Header().Set("X-Request-ID", request.Header.Get("X-Request-ID"))
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("Referrer-Policy", "no-referrer")
	writer.Header().Set("X-Frame-Options", "DENY")
	writer.Header().Set("Content-Security-Policy", "object-src 'none'; default-src 'self'; img-src 'self' blob: data:; style-src 'self' 'unsafe-inline'; connect-src 'self'; script-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	writer.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	writer.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
	if !a.Config.Dev {
		writer.Header().Set("Strict-Transport-Security", "max-age=31536000")
	}
	if request.URL.Path == "/" || request.URL.Path == "/index.html" {
		writer.Header().Set("Cache-Control", "no-cache")
	}
	if strings.HasPrefix(request.URL.Path, "/api/") {
		writer.Header().Set("Cache-Control", "no-store")
	}
	a.Mux.ServeHTTP(writer, request)
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		slog.Error("encode HTTP response", "error", err)
		http.Error(writer, "Response unavailable", http.StatusInternalServerError)

		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_, err = writer.Write(append(payload, '\n'))
	if err != nil {
		slog.Debug("write HTTP response", "error", err)
	}
}

// Close stops long-lived event streams during graceful shutdown.
func (a *App) Close() { a.notifications.Close() }

// RunWorkers processes account mail, image variants and expired records.
func (a *App) RunWorkers(ctx context.Context) {
	ticker := time.NewTicker(workerIntervalSeconds * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mail.Process(ctx)
			a.images.RunVariant(ctx)
			_ = a.sessions.Cleanup(ctx)
			_ = a.invitations.Cleanup(ctx)
			_ = a.passwordResets.Cleanup(ctx)
			_ = a.images.Cleanup(ctx)
			_ = a.canvases.Cleanup(ctx)
		}
	}
}

const workerIntervalSeconds = 2
