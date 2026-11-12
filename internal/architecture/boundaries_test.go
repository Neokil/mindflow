package architecture_test

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
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

func TestDomainsOwnQueriesAndSchemas(t *testing.T) {
	t.Parallel()
	for _, domain := range []string{"account", "session", "invitation", passwordResetDomain, "email", "installation", "workspace", "page", canvasDomain, "editor", imageDomain} {
		for _, name := range []string{"storage/queries/queries.sql", "storage/queries.sql.go", "storage/sqlc_db.go", "storage/migrations/00001_initial.sql", "storage/migrations/migrations.go"} {
			_, err := os.Stat(filepath.Join("..", domain, name))
			if err != nil {
				t.Errorf("%s does not own %s: %v", domain, name, err)
			}
		}
	}
	files, err := filepath.Glob("../*/*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		checkQueryVisibility(t, file)
	}
}

func checkQueryVisibility(t *testing.T, file string) {
	t.Helper()
	if strings.HasSuffix(file, "_test.go") || strings.HasSuffix(file, ".sql.go") || strings.Contains(file, "sqlc_") {
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
		if path == "database/sql" || strings.HasPrefix(path, "github.com/pressly/goose") {
			t.Errorf("database dependency outside storage: %s imports %s", file, path)
		}
		if !strings.HasPrefix(path, "mindflow/internal/") {
			continue
		}
		name := filepath.Base(path)
		if dependency.Name != nil {
			name = dependency.Name.Name
		}
		domain := filepath.Base(filepath.Dir(file))
		if domain != "mindflow" && strings.Contains(path, "/storage") && !strings.HasPrefix(path, "mindflow/internal/"+domain+"/storage") {
			t.Errorf("%s imports another domain's storage: %s", file, path)
		}
		imports[name] = path
	}
	ast.Inspect(tree, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
			if name, ok := selector.X.(*ast.Ident); ok && imports[name.Name] != "" && selector.Sel.Name == "New" {
				t.Errorf("%s constructs queries outside storage", file)
			}
		}

		return true
	})
}

func TestSQLWritesStayWithTheirDomain(t *testing.T) {
	t.Parallel()
	ownership := map[string][]string{
		"account": {"users"}, "session": {"sessions"}, "invitation": {accountTokensTable}, passwordResetDomain: {accountTokensTable}, "email": {jobsTable}, "installation": {"app_settings"},
		"workspace": {"workspaces"}, "page": {pagesTable}, canvasDomain: {pagesTable, "page_assets", "save_receipts"}, "editor": {"page_leases"}, imageDomain: {"assets", "asset_variants", jobsTable},
	}
	for domain, tables := range ownership {
		checkDomainWrites(t, domain, tables)
	}
}

func checkDomainWrites(t *testing.T, domain string, tables []string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", domain, "storage", "queries", "queries.sql")) //nolint:gosec // G304: domain is selected from the fixed ownership map in this test.
	if err != nil {
		t.Fatal(err)
	}
	mutation := regexp.MustCompile(`(?i)^(?:INSERT(?: OR IGNORE)? INTO|UPDATE|DELETE FROM)\s+(\w+)`)
	for _, statement := range regexp.MustCompile(`(?m)^-- name: `).Split(string(data), -1) {
		lines := strings.Split(strings.TrimSpace(statement), "\n")
		body := strings.Join(lines[1:], "\n")
		match := mutation.FindStringSubmatch(strings.TrimSpace(body))
		if len(match) == 0 {
			continue
		}
		if !contains(tables, match[1]) {
			t.Errorf("%s writes %s owned by another domain", domain, match[1])
		}
		checkPartition(t, domain, match[1], body)
	}
}

func checkPartition(t *testing.T, domain, table, body string) {
	t.Helper()
	if table == accountTokensTable {
		purpose := "'invite'"
		if domain == passwordResetDomain {
			purpose = "'reset'"
		}
		if !strings.Contains(body, purpose) {
			t.Errorf("%s mutates tokens without its purpose guard", domain)
		}
	}
	if table == jobsTable {
		kind := "'mail'"
		if domain == imageDomain {
			kind = "'variant'"
		}
		if !strings.Contains(body, kind) {
			t.Errorf("%s mutates jobs without its kind guard", domain)
		}
	}
	if domain == canvasDomain && table == pagesTable && !strings.HasPrefix(strings.TrimSpace(body), "UPDATE pages SET document=") {
		t.Error("canvas mutates page metadata instead of its document fields")
	}
}

func contains(values []string, value string) bool {
	return slices.Contains(values, value)
}

const (
	canvasDomain       = "canvas"
	accountTokensTable = "account_tokens"
	pagesTable         = "pages"
)

const (
	imageDomain = "image"
	jobsTable   = "jobs"
)

const passwordResetDomain = "passwordreset"
