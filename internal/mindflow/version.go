package mindflow

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

// registerAssets ties the running browser document and the version endpoint to the same build.
func (a *App) registerAssets(assets fs.FS) error {
	index, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		return fmt.Errorf("read frontend build: %w", err)
	}
	digest := sha256.Sum256(index)
	version := hex.EncodeToString(digest[:])
	if a.Config.Version != "" {
		version = a.Config.Version
	}
	index = bytes.Replace(index, []byte("</head>"), []byte(`<meta name="mindflow-version" content="`+version+`"/></head>`), 1)
	a.Mux.HandleFunc("GET /api/v1/version", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Cache-Control", "no-store")
		writeJSON(writer, http.StatusOK, map[string]string{"version": version})
	})
	a.Mux.HandleFunc("GET /", func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/api/") {
			http.NotFound(writer, request)

			return
		}
		path := strings.TrimPrefix(request.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		_, statErr := fs.Stat(assets, path)
		if statErr != nil && strings.Contains(path, ".") {
			http.NotFound(writer, request)

			return
		}
		if path == "index.html" || statErr != nil {
			writer.Header().Set("Cache-Control", "no-cache")
			writer.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = writer.Write(index)

			return
		}
		http.FileServer(http.FS(assets)).ServeHTTP(writer, request)
	})

	return nil
}
