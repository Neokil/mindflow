package architecture_test

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestNoHorizontalPackages(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"store", "database", "domain", "application", "repository", "fault", "identifier", "result", "httpapi"} {
		_, err := os.Stat(filepath.Join("..", name))
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("horizontal package %s still exists: %v", name, err)
		}
	}
}

func TestTopLevelDomainsOwnStorage(t *testing.T) {
	t.Parallel()
	for _, domain := range []string{"account", "workspace", "email", "installation", "release", "feedback"} {
		for _, name := range []string{"storage/queries/queries.sql", "storage/queries.sql.go", "storage/sqlc_db.go", "storage/migrations/00001_initial.sql", "storage/migrations/migrations.go"} {
			_, err := os.Stat(filepath.Join("..", domain, name))
			if err != nil {
				t.Errorf("%s does not own %s: %v", domain, name, err)
			}
		}
	}
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative := strings.TrimPrefix(filepath.ToSlash(path), "../")
		parts := strings.Split(relative, "/")
		if entry.IsDir() && entry.Name() == "storage" && len(parts) != 2 {
			t.Errorf("subdomain defines separate storage: %s", path)
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".go") {
			checkQueryVisibility(t, path, parts[0])
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func checkQueryVisibility(t *testing.T, file, domain string) {
	t.Helper()
	if strings.HasSuffix(file, "_test.go") {
		return
	}
	tree, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	imports := map[string]string{}
	for _, dependency := range tree.Imports {
		path, unquoteErr := strconv.Unquote(dependency.Path.Value)
		if unquoteErr != nil {
			t.Fatal(unquoteErr)
		}
		if !strings.Contains(filepath.ToSlash(file), "/storage/") && (path == "database/sql" || strings.HasPrefix(path, "github.com/pressly/goose")) {
			t.Errorf("database dependency outside storage: %s imports %s", file, path)
		}
		name := filepath.Base(path)
		if dependency.Name != nil {
			name = dependency.Name.Name
		}
		if strings.Contains(path, "/storage") && strings.HasPrefix(path, "mindflow/internal/") && domain != "mindflow" && path != "mindflow/internal/mindflow/storage/transaction" && path != "mindflow/internal/"+domain+"/storage" {
			t.Errorf("%s imports another domain's storage: %s", file, path)
		}
		imports[name] = path
	}
	ast.Inspect(tree, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if selector, selectorOK := call.Fun.(*ast.SelectorExpr); selectorOK {
			if name, nameOK := selector.X.(*ast.Ident); nameOK && strings.HasSuffix(imports[name.Name], "/storage") && selector.Sel.Name == "New" && !strings.Contains(filepath.ToSlash(file), "/storage/") {
				t.Errorf("%s constructs queries outside storage", file)
			}
		}

		return true
	})
}

func TestSQLReadsWritesAndForeignKeysStayWithinDomain(t *testing.T) {
	t.Parallel()
	ownership := map[string][]string{
		"account":      {"users", "sessions", "account_tokens", "avatars", "account_activity", "account_update_preferences"},
		"workspace":    {"workspaces", "pages", "page_assets", "save_receipts", "page_leases", "assets", "asset_variants", "image_jobs", "workspace_members", "active_workspaces", "active_pages", "active_assets", "workspace_access", "asset_access", "comment_threads", "comment_messages", "editor_requests", "workspace_templates", "template_assets"},
		"email":        {"mail_jobs", "mail_deliveries"},
		"installation": {"app_settings"},
		"release":      {"release_state"},
		"feedback":     {"feedback_reports", "feedback_comments", "feedback_events", "feedback_reads"},
	}
	tableReference := regexp.MustCompile(`(?i)\b(?:FROM|JOIN|UPDATE|INTO|REFERENCES|CREATE TABLE(?: IF NOT EXISTS)?)\s+(\w+)`)
	for domain, tables := range ownership {
		files, globErr := filepath.Glob(filepath.Join("..", domain, "storage", "*", "*.sql"))
		if globErr != nil {
			t.Fatal(globErr)
		}
		for _, file := range files {
			data, err := os.ReadFile(file) //nolint:gosec // G304: SQL paths are enumerated under the fixed domain storage directories in this test.
			if err != nil {
				t.Fatal(err)
			}
			sql := regexp.MustCompile(`(?m)^--.*$`).ReplaceAllString(string(data), "")
			for _, match := range tableReference.FindAllStringSubmatch(sql, -1) {
				if !strings.EqualFold(match[1], "SET") && !slices.Contains(tables, match[1]) {
					t.Errorf("%s accesses foreign table %s in %s", domain, match[1], file)
				}
			}
		}
	}
}
