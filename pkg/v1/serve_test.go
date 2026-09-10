package v1

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func startUITestServer(t *testing.T, tester *Tester) *UIServer {
	t.Helper()
	srv := NewUIServer(tester)
	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start UI server: %v", err)
	}
	t.Cleanup(srv.Stop)
	return srv
}

func postJSON(t *testing.T, url string, body interface{}) *http.Response {
	t.Helper()
	data, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("POST %s failed: %v", url, err)
	}
	return resp
}

func waitFor(t *testing.T, timeout time.Duration, check func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func TestUIServerState(t *testing.T) {
	tester := NewTester()
	tester.Stage("Setup", func() {})
	tester.Stage("Run", func() {})

	srv := startUITestServer(t, tester)

	resp, err := http.Get(srv.URL() + "/api/state")
	if err != nil {
		t.Fatalf("state request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var state statePayload
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	if len(state.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(state.Stages))
	}
	if state.Stages[0].Name != "Setup" || state.Stages[0].Status != "Not Run" {
		t.Fatalf("unexpected first stage: %+v", state.Stages[0])
	}
}

func TestUIServerDiscoverAndRunStage(t *testing.T) {
	tester := NewTester()
	ran := false
	tester.Stage("Setup", func() {
		ran = true
		RecordAction("noop", func() {})
	})

	srv := startUITestServer(t, tester)

	postJSON(t, srv.URL()+"/api/discover", map[string]any{}).Body.Close()
	if !waitFor(t, 2*time.Second, func() bool {
		return len(GetStageActions("Setup")) == 1
	}) {
		t.Fatal("discover did not record actions")
	}

	postJSON(t, srv.URL()+"/api/stage/run", map[string]any{"name": "Setup"}).Body.Close()
	if !waitFor(t, 2*time.Second, func() bool {
		return srv.getStatus("Setup") == "PASSED"
	}) {
		t.Fatalf("stage did not pass, status=%q", srv.getStatus("Setup"))
	}
	if !ran {
		t.Fatal("stage function was not executed")
	}
}

func TestUIServerRunAction(t *testing.T) {
	tester := NewTester()
	executed := make(chan struct{}, 1)
	tester.Stage("Setup", func() {
		RecordAction("do it", func() { executed <- struct{}{} })
	})

	srv := startUITestServer(t, tester)

	// Record the action without executing it.
	if err := tester.RunStageByName("Setup"); err != nil {
		t.Fatalf("run stage: %v", err)
	}

	postJSON(t, srv.URL()+"/api/action/run", map[string]any{"stage": "Setup", "index": 0}).Body.Close()

	select {
	case <-executed:
	case <-time.After(2 * time.Second):
		t.Fatal("action was not executed")
	}
}

func TestUIServerEventsStreamsState(t *testing.T) {
	tester := NewTester()
	tester.Stage("Setup", func() {})

	srv := startUITestServer(t, tester)

	req, _ := http.NewRequest(http.MethodGet, srv.URL()+"/api/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("events request failed: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("unexpected content type: %s", ct)
	}

	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read event: %v", err)
	}
	if !strings.HasPrefix(line, "data: ") {
		t.Fatalf("unexpected SSE line: %q", line)
	}

	var payload eventPayload
	if err := json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(line), "data: ")), &payload); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if payload.Type != "state" || payload.State == nil || len(payload.State.Stages) != 1 {
		t.Fatalf("unexpected initial event: %+v", payload)
	}
}
