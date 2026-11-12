package store

import "context"

func (a *Repositories) cleanup(ctx context.Context) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	for _, q := range []string{
		"DELETE FROM jobs WHERE asset_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM assets WHERE assets.id=jobs.asset_id)",
		"DELETE FROM jobs WHERE kind='mail' AND NOT EXISTS(SELECT 1 FROM account_tokens WHERE account_tokens.user_id=jobs.user_id AND account_tokens.expires_at>unixepoch())",
		"DELETE FROM account_tokens WHERE expires_at<=unixepoch()",
		"DELETE FROM sessions WHERE expires_at<=unixepoch()",
		"DELETE FROM save_receipts WHERE expires_at<=unixepoch()",
		"DELETE FROM assets WHERE created_at<unixepoch()-86400 AND NOT EXISTS(SELECT 1 FROM page_assets WHERE page_assets.asset_id=assets.id)",
	} {
		if _, err = tx.ExecContext(ctx, q); err != nil {
			return
		}
	}
	_ = tx.Commit()
}
