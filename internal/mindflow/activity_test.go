package mindflow_test

import (
	"testing"
	"time"
)

func TestAuthenticatedUseCancelsPendingDeletionWithoutAnotherLogin(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	member, userID := sharedUser(owner, "active-again@example.com")
	_, err := owner.db.ExecContext(t.Context(), "UPDATE account_activity SET last_seen=?,cycle='pending',deadline=1,final_deadline=1 WHERE user_id=?", time.Now().Add(-100*24*time.Hour).Unix(), userID)
	if err != nil {
		t.Fatal(err)
	}
	member.ok("GET", "/api/v1/auth/me", nil)
	var cycle string
	var deadline, lastSeen int64
	err = owner.db.QueryRowContext(t.Context(), "SELECT cycle,final_deadline,last_seen FROM account_activity WHERE user_id=?", userID).Scan(&cycle, &deadline, &lastSeen)
	if err != nil {
		t.Fatal(err)
	}
	if cycle != "" || deadline != 0 || lastSeen < time.Now().Add(-time.Minute).Unix() {
		t.Fatal("authenticated use did not cancel deletion", cycle, deadline, lastSeen)
	}
}
