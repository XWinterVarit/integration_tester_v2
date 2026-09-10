package v1

import (
	"strings"
	"testing"
)

func TestTester(t *testing.T) {
	tester := NewTester()

	// Test adding stages
	tester.Stage("Stage1", func() {})
	tester.Stage("Stage2", func() {})

	if len(tester.Stages) != 2 {
		t.Errorf("Expected 2 stages, got %d", len(tester.Stages))
	}

	// Test running stage success
	err := tester.RunStageByName("Stage1")
	if err != nil {
		t.Errorf("Stage1 failed: %v", err)
	}

	// Test stage not found
	err = tester.RunStageByName("StageX")
	if err == nil {
		t.Error("Expected error for missing stage")
	}

	// Test stage failure
	tester.Stage("FailStage", func() {
		Fail("Explicit fail")
	})

	err = tester.RunStageByName("FailStage")
	if err == nil {
		t.Error("Expected error for FailStage")
	}
	if !strings.Contains(err.Error(), "Explicit fail") {
		t.Errorf("Expected error message 'Explicit fail', got %v", err)
	}

	// Test stage panic
	tester.Stage("PanicStage", func() {
		panic("Something bad happened")
	})

	err = tester.RunStageByName("PanicStage")
	if err == nil {
		t.Error("Expected error for PanicStage")
	}
	if !strings.Contains(err.Error(), "panic: Something bad happened") {
		t.Errorf("Expected error message 'panic: Something bad happened', got %v", err)
	}
}

func TestDryRun(t *testing.T) {
	tester := NewTester()
	tester.Stage("DryRunStage", func() {
		// This should be recorded
		RecordAction("My Action", func() {})
	})

	tester.DryRunAll()

	actions := GetStageActions("DryRunStage")
	if len(actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(actions))
	}
	if actions[0].Summary != "My Action" {
		t.Errorf("Expected action summary 'My Action', got '%s'", actions[0].Summary)
	}

	// Test IsDryRun
	tester.Stage("CheckDryRun", func() {
		if !IsDryRun() {
			panic("Expected IsDryRun to be true")
		}
	})
	tester.DryRunStage(tester.Stages[1])
}

func TestDryRunWithMockClient(t *testing.T) {
	tester := NewTester()

	tester.Stage("MockStage", func() {
		client := NewDynamicMockClient("http://localhost:9001")
		err := client.RegisterRoute(9002, "GET", "/mock-test", nil)
		AssertNoError(err)

		resp := SendRequest("http://localhost:8080/mock-test")
		ExpectStatusCode(resp, 200)
	})

	tester.DryRunAll()

	actions := GetStageActions("MockStage")
	if len(actions) == 0 {
		t.Fatalf("expected actions recorded during dry-run")
	}
}
