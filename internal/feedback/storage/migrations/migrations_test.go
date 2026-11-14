package migrations_test

import (
	"database/sql"
	"mindflow/internal/feedback/storage/migrations"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // Register SQLite for real upgrade verification.
)

func TestLifecycleUpgradePreservesReportsCommentsAndMedia(t *testing.T) {
	t.Parallel()
	database, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "feedback.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	_, err = database.ExecContext(t.Context(), "PRAGMA foreign_keys=ON")
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, database, migrations.Files(), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}
	_, err = provider.UpTo(t.Context(), 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"submitted", acceptedState, "denied", closedState} {
		_, err = database.ExecContext(t.Context(), "INSERT INTO feedback_reports(id,author_id,kind,title,body,status,screenshot,created_at,updated_at) VALUES(?,?,'bug','Title','Body',?,X'89504E47',1,1)", state, "author", state)
		if err != nil {
			t.Fatal(err)
		}
		_, err = database.ExecContext(t.Context(), "INSERT INTO feedback_comments(id,report_id,author_id,body,created_at) VALUES(?,?,?,'Reply',1)", "reply-"+state, state, "admin")
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = provider.Up(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for old, next := range map[string]string{"submitted": "created", acceptedState: acceptedState, "denied": closedState, closedState: closedState} {
		var state string
		var media int
		err = database.QueryRowContext(t.Context(), "SELECT status,length(screenshot) FROM feedback_reports WHERE id=?", old).Scan(&state, &media)
		if err != nil {
			t.Fatal(err)
		}
		if state != next || media != 4 {
			t.Fatal("upgrade lost report state or media", old, state, media)
		}
		var events int
		err = database.QueryRowContext(t.Context(), "SELECT count(*) FROM feedback_events WHERE report_id=?", old).Scan(&events)
		if err != nil {
			t.Fatal(err)
		}
		if events != 2 {
			t.Fatal("upgrade lost comment activity", old, events)
		}
	}
	var comments int
	err = database.QueryRowContext(t.Context(), "SELECT count(*) FROM feedback_comments").Scan(&comments)
	if err != nil {
		t.Fatal(err)
	}
	if comments != 4 {
		t.Fatal("upgrade lost comments", comments)
	}
}

const acceptedState = "accepted"

const closedState = "closed"
