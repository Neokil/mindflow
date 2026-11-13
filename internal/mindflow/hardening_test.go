package mindflow_test

import (
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestRateLimiterFailsClosedAtItsMemoryBound(t *testing.T) {
	t.Parallel()
	app := fixture(t).a
	request := httptest.NewRequestWithContext(t.Context(), "POST", "/", nil)
	const capacity = 10000
	for index := range capacity {
		if app.Limited(request, "client-"+strconv.Itoa(index)) {
			t.Fatal("fresh rate key blocked before capacity")
		}
	}
	if !app.Limited(request, "overflow") {
		t.Fatal("rate cache admitted an unbounded new key")
	}
	if app.Limited(request, "client-0") {
		t.Fatal("bounded cache rejected an existing legitimate key")
	}
}

func TestSecurityResponseHeaders(t *testing.T) {
	t.Parallel()
	client := fixture(t)
	response := client.request("GET", "/health/live", nil)
	for name, want := range map[string]string{"X-Content-Type-Options": "nosniff", "Cross-Origin-Resource-Policy": "same-origin", "Cross-Origin-Opener-Policy": "same-origin"} {
		if response.Header().Get(name) != want {
			t.Errorf("%s: %q", name, response.Header().Get(name))
		}
	}
}
