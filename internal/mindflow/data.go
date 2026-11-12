package mindflow

import (
	"context"
	"database/sql"
	"mindflow/internal/account"
	"mindflow/internal/canvas"
	"mindflow/internal/editor"
	"mindflow/internal/email"
	imageasset "mindflow/internal/image"
	"mindflow/internal/installation"
	"mindflow/internal/invitation"
	"mindflow/internal/page"
	"mindflow/internal/passwordreset"
	"mindflow/internal/session"
	"mindflow/internal/workspace"
)

// OpenData installs domain-owned schemas using the application's SQLite durability settings.
func OpenData(ctx context.Context, path string) (*sql.DB, error) {
	return installation.Open(ctx, path, account.Schema, session.Schema, invitation.Schema, passwordreset.Schema, email.Schema, workspace.Schema, page.Schema, canvas.Schema, editor.Schema, imageasset.Schema)
}
