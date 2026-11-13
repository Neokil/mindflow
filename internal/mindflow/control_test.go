package mindflow_test

import (
	"encoding/json"
	"mindflow/internal/workspace/page/editor"
	"testing"
)

func requestLease(client *testClient, pageID, clientID, action, requestID string) editor.LeaseResult {
	client.t.Helper()
	response := client.ok("POST", "/api/v1/pages/"+pageID+"/lease", editor.LeaseCommand{ClientID: clientID, Action: action, RequestID: requestID})
	var result editor.LeaseResult
	err := json.Unmarshal(response.Body.Bytes(), &result)
	if err != nil {
		client.t.Fatal(err)
	}

	return result
}

func TestControlRequestDenialAcceptanceAndDeadline(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	_, pageID := owner.workspace()
	first := requestLease(owner, pageID, "first-editor", "acquire", "")
	request := requestLease(owner, pageID, "second-editor", "takeover", "")
	if request.Editable || request.Request == nil || request.Request.Status != "pending" {
		t.Fatal("takeover skipped confirmation")
	}
	duplicate := requestLease(owner, pageID, "second-editor", "takeover", "")
	if duplicate.Request.Deadline != request.Request.Deadline {
		t.Fatal("repeat request extended timeout")
	}
	path := "/api/v1/pages/" + pageID + "/lease"
	if owner.request("POST", path, editor.LeaseCommand{ClientID: "second-editor", Action: "accept", RequestID: request.Request.ID}).Code != 403 {
		t.Fatal("requester accepted their own transfer")
	}
	requestLease(owner, pageID, "first-editor", "deny", request.Request.ID)
	denied := requestLease(owner, pageID, "second-editor", "acquire", "")
	if denied.Editable {
		t.Fatal("denied request transferred control")
	}
	pending := requestLease(owner, pageID, "second-editor", "takeover", "")
	owner.clock.Add(9000)
	before := requestLease(owner, pageID, "first-editor", "acquire", "")
	if !before.Editable {
		t.Fatal("control transferred early")
	}
	owner.clock.Add(2000)
	after := requestLease(owner, pageID, "second-editor", "acquire", "")
	if !after.Editable || after.Epoch <= first.Epoch {
		t.Fatal("deadline did not fence former holder")
	}
	if pending.Request == nil {
		t.Fatal("missing pending request")
	}
	next := requestLease(owner, pageID, "third-editor", "takeover", "")
	requestLease(owner, pageID, "second-editor", "accept", next.Request.ID)
	accepted := requestLease(owner, pageID, "third-editor", "acquire", "")
	if !accepted.Editable {
		t.Fatal("explicit acceptance failed")
	}
}

func TestControlRequestRechecksRevokedWorkspacePermission(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	member, userID := sharedUser(owner, "requester@example.com")
	workspaceID, pageID := owner.workspace()
	owner.ok("PUT", "/api/v1/workspaces/"+workspaceID+"/members", map[string]string{emailField: "requester@example.com", roleField: fullAccessRole})
	requestLease(owner, pageID, "first-editor", "acquire", "")
	requestLease(member, pageID, "second-editor", "takeover", "")
	owner.ok("DELETE", "/api/v1/workspaces/"+workspaceID+"/members/"+userID, nil)
	owner.clock.Add(11000)
	result := requestLease(owner, pageID, "first-editor", "acquire", "")
	if !result.Editable {
		t.Fatal("revoked member received edit control")
	}
}

func TestControlRequestCancelsAfterRequesterSignsOut(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	member, _ := sharedUser(owner, "leaving@example.com")
	workspaceID, pageID := owner.workspace()
	owner.ok("PUT", "/api/v1/workspaces/"+workspaceID+"/members", map[string]string{emailField: "leaving@example.com", roleField: fullAccessRole})
	requestLease(owner, pageID, "first-editor", "acquire", "")
	pending := requestLease(member, pageID, "second-editor", "takeover", "")
	member.ok("POST", "/api/v1/auth/logout", nil)
	owner.clock.Add(11000)
	result := requestLease(owner, pageID, "first-editor", "acquire", "")
	if !result.Editable || pending.Request == nil {
		t.Fatal("signed-out requester received control")
	}
}
