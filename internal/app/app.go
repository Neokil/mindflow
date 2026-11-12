package app

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	web "mindflow/web"
)

type Config struct {
	Dev       bool
	Bootstrap string
}
type App struct {
	Log       *slog.Logger
	hashMu    sync.Mutex
	limitMu   sync.Mutex
	limits    map[string]rateWindow
	dummyHash string
	DB        *sql.DB
	Config    Config
	Mux       *http.ServeMux
}

func New(db *sql.DB, config Config) *App {
	a := &App{DB: db, Config: config, Mux: http.NewServeMux(), Log: slog.Default(), limits: make(map[string]rateWindow)}
	a.dummyHash, _ = hashPassword("dummy password never used")
	a.identityRoutes()
	a.Mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]bool{"ok": true}) })
	a.Mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		if db.PingContext(r.Context()) != nil {
			writeJSON(w, 503, map[string]bool{"ok": false})
			return
		}
		writeJSON(w, 200, map[string]bool{"ok": true})
	})
	assets, _ := fs.Sub(web.Files, "dist")
	a.Mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(assets, p); err != nil {
			if strings.Contains(p, ".") {
				http.NotFound(w, r)
				return
			}
			r.URL.Path = "/index.html"
		}
		http.FileServer(http.FS(assets)).ServeHTTP(w, r)
	})
	return a
}
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("X-Request-ID", randomID()[:16])
	w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' blob: data:; style-src 'self' 'unsafe-inline'; connect-src 'self'; script-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Cache-Control", "no-store")
	}
	a.Mux.ServeHTTP(w, r)
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
