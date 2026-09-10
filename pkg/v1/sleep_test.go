package v1

import (
	"testing"
	"time"
)

func TestSleepDryRunSkipsDelay(t *testing.T) {
	tester := NewTester()
	tester.Stage("SleepStage", func() {
		Sleep(200 * time.Millisecond)
	})

	start := time.Now()
	tester.DryRunAll()
	if time.Since(start) > 50*time.Millisecond {
		t.Fatalf("Sleep should be skipped in dry-run")
	}

	acts := GetStageActions("SleepStage")
	if len(acts) != 1 || acts[0].Summary == "" {
		t.Fatalf("expected recorded sleep action, got %#v", acts)
	}
}

func TestSleepRealDelayAndRecording(t *testing.T) {
	tester := NewTester()
	tester.Stage("SleepStage", func() {
		Sleep(100 * time.Millisecond)
	})

	start := time.Now()
	if err := tester.RunStageByName("SleepStage"); err != nil {
		t.Fatalf("stage failed: %v", err)
	}
	if time.Since(start) < 90*time.Millisecond {
		t.Fatalf("Sleep did not wait long enough")
	}

	acts := GetStageActions("SleepStage")
	if len(acts) != 1 || acts[0].Summary == "" {
		t.Fatalf("expected recorded sleep action, got %#v", acts)
	}
}
