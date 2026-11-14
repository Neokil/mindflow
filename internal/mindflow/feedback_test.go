package mindflow_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
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
	if author.request("PATCH", path, feedback.ReviewCommand{Status: "accepted"}).Code != http.StatusForbidden {
		t.Fatal("author reviewed own report")
	}
	author.ok("GET", path+"/screenshot", nil)
	owner.ok("GET", path+"/screenshot", nil)
	owner.ok("POST", path+"/comments", map[string]string{commentBodyField: "Thank you, investigating."})
	author.ok("POST", path+"/comments", map[string]string{commentBodyField: "Here is more information."})
	if owner.request("PATCH", path, feedback.ReviewCommand{Status: feedbackClosedStatus, Release: "https://evil.example"}).Code != http.StatusBadRequest {
		t.Fatal("closed bug without a release")
	}
	owner.ok("PATCH", path, feedback.ReviewCommand{Status: "accepted"})
	response = owner.ok("PATCH", path, feedback.ReviewCommand{Status: feedbackClosedStatus, Release: releaseVersionForTest})
	err = json.Unmarshal(response.Body.Bytes(), &report)
	if err != nil {
		t.Fatal(err)
	}
	if report.ReleaseURL != "https://github.com/Neokil/mindflow/releases/tag/v1.2.0" || len(report.Comments) != 2 {
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

func TestFeedbackRejectsInvalidAttachmentsAndFeatureClosure(t *testing.T) {
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
	if client.request("PATCH", "/api/v1/feedback/"+report.ID, feedback.ReviewCommand{Status: feedbackClosedStatus, Release: releaseVersionForTest}).Code != http.StatusBadRequest {
		t.Fatal("feature closed like a bug")
	}
	client.ok("PATCH", "/api/v1/feedback/"+report.ID, feedback.ReviewCommand{Status: "denied"})
}

const feedbackClosedStatus = "closed"
