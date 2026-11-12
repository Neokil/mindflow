package architecture_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestDomainOwnsStorage(t *testing.T) {
	t.Parallel()
	for _, domain := range []string{"authentication", "workspace", "app"} {
		files, err := filepath.Glob(filepath.Join("..", domain, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			checkDomainFile(t, domain, file)
		}
	}
}

func checkDomainFile(t *testing.T, domain, file string) {
	t.Helper()
	if strings.HasSuffix(file, "_test.go") {
		return
	}
	tree, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, dependency := range tree.Imports {
		path, unquoteErr := strconv.Unquote(dependency.Path.Value)
		if unquoteErr != nil {
			t.Fatal(unquoteErr)
		}
		if strings.HasSuffix(path, "/dbquery") && path != "mindflow/internal/"+domain+"/dbquery" {
			t.Errorf("%s imports another domain's storage: %s", file, path)
		}
		if domain == "authentication" && strings.HasPrefix(path, "mindflow/internal/workspace") {
			t.Errorf("authentication depends on workspace internals: %s", file)
		}
		if domain == "app" && filepath.Base(file) != "app.go" && path == "database/sql" {
			t.Errorf("HTTP adapter accesses SQL directly: %s", file)
		}
	}
}
