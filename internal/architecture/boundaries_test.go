package architecture

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Keep the domain and application contracts independent of their adapters.
// The HTTP composition root may receive *sql.DB, but handlers cannot use SQL.
func TestDependencyDirection(t *testing.T) {
	for _, layer := range []string{"domain", "application", "app"} {
		files, err := filepath.Glob(filepath.Join("..", layer, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			tree, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, imp := range tree.Imports {
				path, _ := strconv.Unquote(imp.Path.Value)
				if layer != "app" && (path == "net/http" || path == "database/sql" || strings.HasPrefix(path, "mindflow/internal/store") || strings.HasPrefix(path, "mindflow/internal/app")) {
					t.Errorf("%s imports an outer adapter: %s", file, path)
				}
				if layer == "domain" && strings.HasPrefix(path, "mindflow/internal/application") {
					t.Errorf("domain imports application: %s", file)
				}
			}
			if layer == "app" && filepath.Base(file) != "app.go" {
				for _, imp := range tree.Imports {
					path, _ := strconv.Unquote(imp.Path.Value)
					if path == "database/sql" || path == "mindflow/internal/store" {
						t.Errorf("HTTP handler imports persistence: %s (%s)", file, path)
					}
				}
			}
		}
	}
}
