package email_test

import (
	"mindflow/internal/email"
	"testing"
)

func TestDeliveryRejectsPlaintextAndHeaderInjectionBeforeConnecting(t *testing.T) {
	t.Parallel()
	for _, config := range []email.Config{{Mode: "plain"}, {Mode: "tls", Name: "mindflow\r\nBcc: stolen@example.invalid"}} {
		if email.Send(t.Context(), config, "person@example.invalid", "Test", "Body") == nil {
			t.Fatal("unsafe SMTP configuration accepted")
		}
	}
}
