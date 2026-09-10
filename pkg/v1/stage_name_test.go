package v1

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestStageNameWithColon verifies stage names containing ":" work everywhere.
func TestStageNameWithColon(t *testing.T) {
	const name = "Setup: DB"

	tester := NewTester()
	ran := false
	actionRan := false
	tester.Stage(name, func() {
		ran = true
		RecordAction("create: table", func() { actionRan = true })
	})

	if err := tester.RunStageByName(name); err != nil {
		t.Fatalf("RunStageByName(%q) failed: %v", name, err)
	}
	if !ran {
		t.Fatalf("stage %q did not run", name)
	}
	if got := len(GetStageActions(name)); got != 1 {
		t.Fatalf("expected 1 action for %q, got %d", name, got)
	}

	srv := startUITestServer(t, tester)

	// State must expose the full name.
	resp, err := http.Get(srv.URL() + "/api/state")
	if err != nil {
		t.Fatalf("state request failed: %v", err)
	}
	defer resp.Body.Close()
	var state statePayload
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	if len(state.Stages) != 1 || state.Stages[0].Name != name {
		t.Fatalf("unexpected stages: %+v", state.Stages)
	}

	// Running the stage via the API must match the full name.
	postJSON(t, srv.URL()+"/api/stage/run", map[string]any{"name": name}).Body.Close()
	if !waitFor(t, 2*time.Second, func() bool { return srv.getStatus(name) == "PASSED" }) {
		t.Fatalf("stage %q did not pass via API, status=%q", name, srv.getStatus(name))
	}

	// Running the action via the API must resolve the stage by its full name.
	postJSON(t, srv.URL()+"/api/action/run", map[string]any{"stage": name, "index": 0}).Body.Close()
	if !waitFor(t, 2*time.Second, func() bool { return actionRan }) {
		t.Fatalf("action for stage %q was not executed", name)
	}
}
