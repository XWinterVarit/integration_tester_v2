package v1

import (
	"net/http"
	"testing"
	"time"
)

func TestRunGUI(t *testing.T) {
	// The GUI now runs the Electron + React frontend against the HTTP API.
	// Launching a window is not possible in a headless test, but we can verify
	// the backing server starts and serves state.
	tester := NewTester()
	tester.Stage("Example", func() {})

	srv := NewUIServer(tester)
	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start UI server: %v", err)
	}
	defer srv.Stop()

	client := &http.Client{Timeout: time.Second}
	resp, err := client.Get(srv.URL() + "/api/state")
	if err != nil {
		t.Fatalf("state request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
