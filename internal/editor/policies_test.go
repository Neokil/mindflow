package editor_test

import (
	"errors"
	"mindflow/internal/editor"
	"testing"
)

func TestLeaseTakeoverFencesPreviousWriter(t *testing.T) {
	t.Parallel()

	first, _, err := (editor.LeaseState{}).Advance("session-a", editor.LeaseCommand{ClientID: "client-a", Action: "acquire"}, 100)
	if err != nil {
		t.Fatal(err)
	}
	unchanged, changed, err := first.Advance("session-b", editor.LeaseCommand{ClientID: "client-b", Action: "acquire"}, 110)
	if err != nil || changed || unchanged != first {
		t.Fatal("second editor acquired a live lease")
	}
	second, _, err := first.Advance("session-b", editor.LeaseCommand{ClientID: "client-b", Action: "takeover"}, 110)
	if err != nil {
		t.Fatal(err)
	}
	var problem *editor.Error
	if !errors.As(second.CheckSave("session-a", "client-a", first.Epoch, 1, 1, 111), &problem) || problem.Code != "lease_lost" {
		t.Fatal("stale editor was not fenced")
	}
	err = second.CheckSave("session-b", "client-b", second.Epoch, 1, 1, 111)
	if err != nil {
		t.Fatal(err)
	}
	err = second.CheckSave("session-b", "client-b", second.Epoch, 1, 2, 111)
	if err == nil {
		t.Fatal("stale revision accepted")
	}
	err = second.CheckSave("session-b", "client-b", second.Epoch, 1, 1, 171)
	if err == nil {
		t.Fatal("expired lease accepted")
	}
}
