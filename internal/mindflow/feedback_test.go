package mindflow_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"mindflow/internal/account"
	"mindflow/internal/feedback"
	"net/http"
	"strings"
	"testing"
)

func TestFeedbackIsPrivateAndSupportsDiscussionReviewAndDeletion(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	author, userID := sharedUser(owner, "reporter@example.com")
	stranger, _ := sharedUser(owner, "stranger@example.com")
	var screenshot bytes.Buffer
	err := png.Encode(&screenshot, image.NewRGBA(image.Rect(0, 0, 4, 4)))
	if err != nil {
		t.Fatal(err)
	}
	response := author.ok("POST", "/api/v1/feedback", feedback.CreateCommand{Kind: "bug", Title: "A visible problem", Body: "How to reproduce it", Screenshot: base64.StdEncoding.EncodeToString(screenshot.Bytes())})
	var report feedback.Report
	err = json.Unmarshal(response.Body.Bytes(), &report)
	if err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/feedback/" + report.ID
	for _, resource := range []string{path, path + "/screenshot"} {
		if stranger.request("GET", resource, nil).Code != http.StatusNotFound {
			t.Fatal("another user could read private report", resource)
		}
	}
	if strings.Contains(stranger.ok("GET", "/api/v1/feedback", nil).Body.String(), report.Title) {
		t.Fatal("private report listed")
	}
	if stranger.request("POST", path+"/comments", map[string]string{commentBodyField: "intrusion"}).Code != http.StatusNotFound {
		t.Fatal("another user could comment")
	}
	if author.request("PATCH", path, feedback.ReviewCommand{Action: feedback.ActionAccept}).Code != http.StatusForbidden {
		t.Fatal("author reviewed own report")
	}
	author.ok("GET", path+"/screenshot", nil)
	owner.ok("GET", path+"/screenshot", nil)
	owner.ok("POST", path+"/comments", map[string]string{commentBodyField: "Thank you, investigating."})
	author.ok("POST", path+"/comments", map[string]string{commentBodyField: "Here is more information."})
	if owner.request("PATCH", path, feedback.ReviewCommand{Action: feedback.ActionCloseRelease, Comment: "https://evil.example"}).Code != http.StatusBadRequest {
		t.Fatal("closed bug without a release")
	}
	owner.ok("PATCH", path, feedback.ReviewCommand{Action: feedback.ActionAccept})
	response = owner.ok("PATCH", path, feedback.ReviewCommand{Action: feedback.ActionCloseRelease, Comment: "Fixed in " + releaseVersionForTest})
	err = json.Unmarshal(response.Body.Bytes(), &report)
	if err != nil {
		t.Fatal(err)
	}
	if report.ReleaseURL != "https://github.com/Neokil/mindflow/releases/tag/v1.2.0" || len(report.Comments) != 4 {
		t.Fatal("incorrect discussion or release", report)
	}
	owner.ok("DELETE", "/api/v1/admin/users/"+userID, map[string]bool{"confirm": true})
	if owner.request("GET", path, nil).Code != http.StatusNotFound {
		t.Fatal("deleted author's report retained")
	}
	if owner.request("GET", path+"/screenshot", nil).Code != http.StatusNotFound {
		t.Fatal("deleted author's screenshot retained")
	}
}

func TestFeedbackRejectsInvalidAttachmentsAndPrematureClosure(t *testing.T) {
	t.Parallel()
	client := fixture(t)
	client.setup()
	invalid := feedback.CreateCommand{Kind: "bug", Title: "Broken", Body: "Details", Screenshot: base64.StdEncoding.EncodeToString([]byte("<svg/>"))}
	if client.request("POST", "/api/v1/feedback", invalid).Code != http.StatusBadRequest {
		t.Fatal("invalid screenshot accepted")
	}
	response := client.ok("POST", "/api/v1/feedback", feedback.CreateCommand{Kind: "feature", Title: "A new tool", Body: "Useful for sketching"})
	var report feedback.Report
	err := json.Unmarshal(response.Body.Bytes(), &report)
	if err != nil {
		t.Fatal(err)
	}
	if client.request("PATCH", "/api/v1/feedback/"+report.ID, feedback.ReviewCommand{Action: feedback.ActionCloseRelease, Comment: "Fixed in " + releaseVersionForTest}).Code != http.StatusBadRequest {
		t.Fatal("feature closed before acceptance")
	}
	client.ok("PATCH", "/api/v1/feedback/"+report.ID, feedback.ReviewCommand{Action: feedback.ActionDeny, Comment: "Not planned."})
}

func TestFeedbackUnreadIsPrivatePerReaderAndUsesDisplayedSnapshot(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	author, _ := sharedUser(owner, "unread-author@example.com")
	stranger, _ := sharedUser(owner, "unread-stranger@example.com")
	moderator, moderatorID := sharedUser(owner, "unread-admin@example.com")
	owner.ok("PATCH", "/api/v1/admin/users/"+moderatorID, map[string]string{roleField: account.RoleAdmin})
	created := author.ok("POST", "/api/v1/feedback", feedback.CreateCommand{Kind: "feature", Title: "Unread test", Body: "Please add this feature."})
	var report feedback.Report
	err := json.Unmarshal(created.Body.Bytes(), &report)
	if err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/feedback/" + report.ID
	counts := func(client *testClient, reports, replies int64) {
		t.Helper()
		var unread feedback.UnreadCounts
		err = json.Unmarshal(client.ok("GET", "/api/v1/feedback/unread", nil).Body.Bytes(), &unread)
		if err != nil {
			t.Fatal(err)
		}
		if unread.Reports != reports || unread.Replies != replies {
			t.Fatalf("counts %#v, want %d/%d", unread, reports, replies)
		}
	}
	counts(owner, 1, 0)
	counts(moderator, 1, 0)
	counts(author, 0, 0)
	counts(stranger, 0, 0)
	initial := report.ReadThrough
	owner.ok("POST", path+"/read", map[string]int64{feedbackReadThroughField: initial})
	counts(owner, 0, 0)
	counts(moderator, 1, 0)
	owner.ok("POST", path+"/comments", map[string]string{commentBodyField: "Can you explain the use case?"})
	counts(author, 0, 1)
	counts(owner, 0, 0)
	counts(moderator, 2, 0)
	// A stale acknowledgement must not consume a reply the client has not seen.
	author.ok("POST", path+"/read", map[string]int64{feedbackReadThroughField: initial})
	counts(author, 0, 1)
	err = json.Unmarshal(author.ok("GET", path, nil).Body.Bytes(), &report)
	if err != nil {
		t.Fatal(err)
	}
	author.ok("POST", path+"/read", map[string]int64{feedbackReadThroughField: report.ReadThrough})
	counts(author, 0, 0)
	author.ok("POST", path+"/comments", map[string]string{commentBodyField: "It would save a few clicks."})
	counts(owner, 1, 0)
	counts(moderator, 3, 0)
	counts(author, 0, 0)
	if stranger.request("POST", path+"/read", map[string]int64{feedbackReadThroughField: report.ReadThrough}).Code != http.StatusNotFound {
		t.Fatal("another reader marked a private report")
	}
	if stranger.request("GET", "/api/v1/feedback?scope=all", nil).Code != http.StatusForbidden {
		t.Fatal("non-admin opened all reports")
	}
	if owner.request("POST", path+"/read", map[string]int64{feedbackReadThroughField: report.ReadThrough + 100}).Code != http.StatusBadRequest {
		t.Fatal("future read position accepted")
	}
	owner.ok("PATCH", path, feedback.ReviewCommand{Action: feedback.ActionDeny, Comment: "This is outside the scope."})
	err = json.Unmarshal(author.ok("GET", path, nil).Body.Bytes(), &report)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != feedback.StateClosed || report.Comments[len(report.Comments)-1].Body != "This is outside the scope." {
		t.Fatal("denial explanation was not recorded")
	}
	if owner.request("PATCH", path, feedback.ReviewCommand{Action: feedback.ActionAccept}).Code != http.StatusBadRequest {
		t.Fatal("closed report skipped reopen")
	}
	owner.ok("PATCH", path, feedback.ReviewCommand{Action: feedback.ActionReopen})
	owner.ok("PATCH", path, feedback.ReviewCommand{Action: feedback.ActionAccept})
	owner.ok("PATCH", path, feedback.ReviewCommand{Action: feedback.ActionCloseComment, Comment: "Implemented without a release."})
}

const feedbackReadThroughField = "through"
