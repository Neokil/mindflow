package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"io/fs"
	"log/slog"
	"mindflow/internal/authentication"
	"mindflow/internal/workspace"
	"net/http"
	"strings"
	"sync"

	web "mindflow/web"
)

// Config supplies development mode and the one-time setup credential.
type Config struct {
	Dev       bool
	Bootstrap string
}

// App serves the HTTP API and coordinates domain services.
type App struct {
	done           chan struct{}
	closeOnce      sync.Once
	Log            *slog.Logger
	limitMu        sync.Mutex
	limits         map[string]rateWindow
	eventsMu       sync.Mutex
	subscribers    map[string]subscriber
	authentication *authentication.Service
	workspace      *workspace.Service
	Config         Config
	Mux            *http.ServeMux
}

// New composes domain services and registers HTTP routes.
func New(ctx context.Context, connection *sql.DB, config Config) (*App, error) {
	service := &App{Config: config, Mux: http.NewServeMux(), Log: slog.Default(), limits: make(map[string]rateWindow), subscribers: make(map[string]subscriber), done: make(chan struct{})}
	auth, err := authentication.New(connection, authentication.Config{Dev: config.Dev, Bootstrap: config.Bootstrap})
	if err != nil {
		return nil, err
	}
	service.authentication = auth
	workspaces, err := workspace.New(ctx, connection, service.publish)
	if err != nil {
		return nil, err
	}
	service.workspace = workspaces
	service.identityRoutes()
	service.workspaceRoutes()

	service.Mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, map[string]bool{"ok": true}) })
	service.Mux.HandleFunc("GET /health/ready", func(writer http.ResponseWriter, r *http.Request) {
		if connection.PingContext(r.Context()) != nil {
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
	request.Header.Set("X-Request-ID", randomID()[:eventQueueCapacity])
	writer.Header().Set("X-Request-ID", request.Header.Get("X-Request-ID"))
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("Referrer-Policy", "no-referrer")
	writer.Header().Set("X-Frame-Options", "DENY")
	writer.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' blob: data:; style-src 'self' 'unsafe-inline'; connect-src 'self'; script-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
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
func (a *App) Close() { a.closeOnce.Do(func() { close(a.done) }) }

// RunWorkers processes account mail, image variants and expired records.
func (a *App) RunWorkers(ctx context.Context) {
	go a.authentication.RunWorkers(ctx)
	a.workspace.RunWorkers(ctx)
}
