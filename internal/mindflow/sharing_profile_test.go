package mindflow_test

import (
	"mindflow/internal/workspace/sharing"
	"net/http"
	"strings"
	"testing"
)

func TestDisplayNamesInSharingCommentsAndControl(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	member, userID := sharedUser(owner, "named-collaborator@example.com")
	member.ok("PATCH", "/api/v1/auth/profile", map[string]string{profileNameField: profileTestName})
	workspaceID, pageID := owner.workspace()
	membersPath := "/api/v1/workspaces/" + workspaceID + "/members"
	owner.ok("PUT", membersPath, sharing.Command{UserID: userID, Role: fullAccessRole})
	if !strings.Contains(owner.ok("GET", membersPath, nil).Body.String(), profileTestName) {
		t.Fatal("member display name missing")
	}
	if member.request("PUT", membersPath, sharing.Command{UserID: userID, Role: fullAccessRole}).Code != http.StatusForbidden {
		t.Fatal("member changed sharing by account ID")
	}
	if owner.request("PUT", membersPath, map[string]string{"userId": userID, emailField: "named-collaborator@example.com", roleField: fullAccessRole}).Code != http.StatusBadRequest {
		t.Fatal("ambiguous account identities accepted")
	}
	commentsPath := "/api/v1/pages/" + pageID + "/comments"
	member.ok("POST", commentsPath, map[string]any{commentBodyField: "Review this", "x": 100, "y": 100})
	if !strings.Contains(owner.ok("GET", commentsPath, nil).Body.String(), `"authorDisplayName":"Ada Lovelace"`) {
		t.Fatal("comment did not resolve the current display name")
	}
	member.ok("PATCH", "/api/v1/auth/profile", map[string]string{profileNameField: "Grace Hopper"})
	if !strings.Contains(owner.ok("GET", commentsPath, nil).Body.String(), `"authorDisplayName":"Grace Hopper"`) {
		t.Fatal("comment retained a stale author name")
	}
	owner.acquire(pageID, "named-owner-client", "acquire")
	request := requestLease(member, pageID, "named-reader-client", "takeover", "")
	if request.Request == nil || request.Request.RequesterDisplayName != "Grace Hopper" {
		t.Fatal("control request omitted the display name")
	}
}
