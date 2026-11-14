package mindflow_test

import (
	"encoding/json"
	"mindflow/internal/feedback"
	feedbackstorage "mindflow/internal/feedback/storage"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func feedbackResponse(t *testing.T, response *httptest.ResponseRecorder) feedback.Report {
	t.Helper()
	var report feedback.Report
	err := json.Unmarshal(response.Body.Bytes(), &report)
	if err != nil {
		t.Fatal(err)
	}

	return report
}

func TestClosedReportsExpireFromClosureAndReopeningRestartsRetention(t *testing.T) {
	t.Parallel()
	client := fixture(t)
	client.setup()
	create := func(kind, title string) feedback.Report {
		t.Helper()

		return feedbackResponse(t, client.ok("POST", "/api/v1/feedback", feedback.CreateCommand{Kind: kind, Title: title, Body: "Retention test"}))
	}
	bug := create(feedback.KindBug, "Bug retention")
	feature := create(feedback.KindFeature, "Feature retention")
	open := create(feedback.KindBug, "Open report")
	path := "/api/v1/feedback/" + bug.ID
	client.ok("PATCH", path, feedback.ReviewCommand{Action: feedback.ActionAccept})
	bug = feedbackResponse(t, client.ok("PATCH", path, feedback.ReviewCommand{Action: feedback.ActionCloseComment, Comment: "Resolved."}))
	feature = feedbackResponse(t, client.ok("PATCH", "/api/v1/feedback/"+feature.ID, feedback.ReviewCommand{Action: feedback.ActionDeny, Comment: "Not planned."}))
	if bug.PurgeAt-bug.ClosedAt != int64(feedbackTestRetention/time.Second) {
		t.Fatal("wrong retention period")
	}
	client.ok("POST", path+"/read", map[string]int64{feedbackReadThroughField: bug.ReadThrough})
	client.clock.Store(int64(29 * 24 * time.Hour / time.Millisecond))
	commented := feedbackResponse(t, client.ok("POST", path+"/comments", map[string]string{commentBodyField: "One more detail."}))
	if commented.PurgeAt != bug.PurgeAt {
		t.Fatal("comment extended retention")
	}
	client.db.SetMaxOpenConns(1)
	_, err := client.db.ExecContext(t.Context(), "PRAGMA foreign_keys=ON")
	if err != nil {
		t.Fatal(err)
	}
	cleanupNow := time.Unix(bug.PurgeAt-1, 0)
	cleanup := feedback.NewService(feedbackstorage.NewRepository(client.db), nil, func() time.Time { return cleanupNow })
	err = cleanup.Cleanup(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	client.ok("GET", path, nil)
	cleanupNow = time.Unix(max(bug.PurgeAt, feature.PurgeAt), 0)
	err = cleanup.Cleanup(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{bug.ID, feature.ID} {
		if client.request("GET", "/api/v1/feedback/"+id, nil).Code != http.StatusNotFound {
			t.Fatal("closed report retained after deadline")
		}
		var remaining int
		err = client.db.QueryRowContext(t.Context(), "SELECT (SELECT count(*) FROM feedback_comments WHERE report_id=?)+(SELECT count(*) FROM feedback_events WHERE report_id=?)+(SELECT count(*) FROM feedback_reads WHERE report_id=?)", id, id, id).Scan(&remaining)
		if err != nil || remaining != 0 {
			t.Fatal("dependent report records retained", remaining, err)
		}
	}
	client.ok("GET", "/api/v1/feedback/"+open.ID, nil)
	again := create(feedback.KindBug, "Reopened report")
	againPath := "/api/v1/feedback/" + again.ID
	again = feedbackResponse(t, client.ok("PATCH", againPath, feedback.ReviewCommand{Action: feedback.ActionDeny, Comment: "Closed for now."}))
	originalDeadline := again.PurgeAt
	reopened := feedbackResponse(t, client.ok("PATCH", againPath, feedback.ReviewCommand{Action: feedback.ActionReopen}))
	if reopened.ClosedAt != 0 || reopened.PurgeAt != 0 {
		t.Fatal("reopening retained deletion deadline")
	}
	cleanupNow = time.Unix(originalDeadline+1, 0)
	err = cleanup.Cleanup(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	client.ok("GET", againPath, nil)
	client.clock.Store(int64(70 * 24 * time.Hour / time.Millisecond))
	again = feedbackResponse(t, client.ok("PATCH", againPath, feedback.ReviewCommand{Action: feedback.ActionDeny, Comment: "Closed again."}))
	if again.PurgeAt <= originalDeadline {
		t.Fatal("reclosing did not restart retention")
	}
	cleanupNow = time.Unix(again.PurgeAt-1, 0)
	err = cleanup.Cleanup(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	client.ok("GET", againPath, nil)
	cleanupNow = cleanupNow.Add(time.Second)
	err = cleanup.Cleanup(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if client.request("GET", againPath, nil).Code != http.StatusNotFound {
		t.Fatal("reclosed report did not expire")
	}
}

func TestFeedbackSearchCoversTitleBodyAndCommentsWithoutCrossingPermissions(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	author, _ := sharedUser(owner, "search-author@example.com")
	stranger, _ := sharedUser(owner, "search-stranger@example.com")
	report := feedbackResponse(t, author.ok("POST", "/api/v1/feedback", feedback.CreateCommand{Kind: feedback.KindBug, Title: "Canvas drift", Body: "Zooming causes misalignment."}))
	author.ok("POST", "/api/v1/feedback/"+report.ID+"/comments", map[string]string{commentBodyField: "Reproduces when drawing with a stylus."})
	owner.ok("POST", "/api/v1/feedback", feedback.CreateCommand{Kind: feedback.KindFeature, Title: "Something else", Body: "An unrelated improvement."})
	for _, query := range []string{"CANVAS", "misalignment", "STYLUS"} {
		result := owner.ok("GET", "/api/v1/feedback?scope=all&q="+url.QueryEscape(query), nil)
		var reports []feedback.Report
		err := json.Unmarshal(result.Body.Bytes(), &reports)
		if err != nil {
			t.Fatal(err)
		}
		if len(reports) != 1 || reports[0].ID != report.ID {
			t.Fatal("search did not cover report content", query, result.Body.String())
		}
		if strings.Contains(stranger.ok("GET", "/api/v1/feedback?q="+url.QueryEscape(query), nil).Body.String(), report.ID) {
			t.Fatal("search leaked a private report")
		}
	}
	for _, query := range []string{"no match", "%", "_"} {
		response := owner.ok("GET", "/api/v1/feedback?scope=all&q="+url.QueryEscape(query), nil)
		if strings.TrimSpace(response.Body.String()) != "[]" {
			t.Fatal("search treated a literal query as a wildcard", query)
		}
	}
}

const feedbackTestRetention = 30 * 24 * time.Hour
