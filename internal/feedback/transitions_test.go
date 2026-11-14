package feedback_test

import (
	"mindflow/internal/feedback"
	"testing"
)

func TestReportStateMachine(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ state, action, comment, status, version string }{
		{feedback.StateCreated, feedback.ActionAccept, "", feedback.StateAccepted, ""},
		{feedback.StateCreated, feedback.ActionDeny, "Outside the scope of this app.", feedback.StateClosed, ""},
		{feedback.StateAccepted, feedback.ActionCloseComment, "Resolved by configuration.", feedback.StateClosed, ""},
		{feedback.StateAccepted, feedback.ActionCloseRelease, "Fixed in v1.3.0-rc.1.", feedback.StateClosed, "v1.3.0-rc.1"},
		{feedback.StateClosed, feedback.ActionReopen, "", feedback.StateCreated, ""},
	} {
		result, err := feedback.Decide(test.state, feedback.ReviewCommand{Action: test.action, Comment: test.comment})
		if err != nil || result.Status != test.status || result.Release != test.version || result.Comment == "" {
			t.Fatalf("%s/%s: %#v, %v", test.state, test.action, result, err)
		}
	}
	for _, test := range []struct{ state, action, comment string }{
		{feedback.StateCreated, feedback.ActionCloseComment, "premature"},
		{feedback.StateCreated, feedback.ActionCloseRelease, "v1.3.0"},
		{feedback.StateCreated, feedback.ActionDeny, ""},
		{feedback.StateAccepted, feedback.ActionDeny, "too late"},
		{feedback.StateAccepted, feedback.ActionAccept, ""},
		{feedback.StateAccepted, feedback.ActionCloseComment, ""},
		{feedback.StateAccepted, feedback.ActionCloseRelease, "no version"},
		{feedback.StateClosed, feedback.ActionAccept, ""},
		{feedback.StateClosed, feedback.ActionCloseComment, "again"},
		{feedback.StateCreated, feedback.ActionReopen, ""},
	} {
		_, err := feedback.Decide(test.state, feedback.ReviewCommand{Action: test.action, Comment: test.comment})
		if err == nil {
			t.Fatalf("allowed invalid transition %s/%s", test.state, test.action)
		}
	}
}
