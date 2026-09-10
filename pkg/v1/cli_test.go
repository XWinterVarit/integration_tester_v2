package v1

import (
	"bytes"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunCLICommandReportsProgressAndBusy(t *testing.T) {
	tester := NewTester()
	tester.Stage("Slow", func() { time.Sleep(80 * time.Millisecond) })

	var out bytes.Buffer
	in := strings.NewReader("run Slow\nrun Slow\nstatus\nexit\n")
	RunCLICommandIO(tester, in, &out, CLICommandOptions{
		Heartbeat: 10 * time.Millisecond,
		SlowAfter: 20 * time.Millisecond,
	})

	got := out.String()
	for _, want := range []string{
		"RUNNING Slow",
		"BUSY running=Slow",         // second run rejected, no overlap
		"STATUS RUNNING stage=Slow", // status available while running
		"SLOW Slow",                 // flagged as taking too long
		"PASSED Slow duration=",     // final result with duration
		"Bye.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n--- output ---\n%s", want, got)
		}
	}
}

func TestRunCLICommandListAndUnknown(t *testing.T) {
	tester := NewTester()
	tester.Stage("Setup: DB", func() {})

	var out bytes.Buffer
	in := strings.NewReader("list\nrun Nope\nexit\n")
	RunCLICommandIO(tester, in, &out, CLICommandOptions{Heartbeat: -1})

	got := out.String()
	if !strings.Contains(got, "  Setup: DB") {
		t.Errorf("list missing stage name\n%s", got)
	}
	if !strings.Contains(got, "ERROR unknown stage: Nope") {
		t.Errorf("missing unknown-stage error\n%s", got)
	}
}

func TestRunCLICommandFailedStage(t *testing.T) {
	tester := NewTester()
	tester.Stage("Boom", func() { Fail("kaboom") })

	var out bytes.Buffer
	in := strings.NewReader("run Boom\nexit\n")
	RunCLICommandIO(tester, in, &out, CLICommandOptions{Heartbeat: -1})

	got := out.String()
	if !strings.Contains(got, "FAILED Boom duration=") || !strings.Contains(got, "kaboom") {
		t.Errorf("missing failure report\n%s", got)
	}
}

func TestRunStageByNameSerialized(t *testing.T) {
	tester := NewTester()

	var running int32
	var maxRunning int32
	tester.Stage("A", func() {
		n := atomic.AddInt32(&running, 1)
		for {
			m := atomic.LoadInt32(&maxRunning)
			if n <= m || atomic.CompareAndSwapInt32(&maxRunning, m, n) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		atomic.AddInt32(&running, -1)
	})

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = tester.RunStageByName("A")
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&maxRunning); got != 1 {
		t.Fatalf("expected stages to be serialized (max concurrency 1), got %d", got)
	}
}

func TestTryRunStageByNameBusy(t *testing.T) {
	tester := NewTester()
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	tester.Stage("Hold", func() {
		close(started)
		<-release
	})

	go func() {
		_ = tester.RunStageByName("Hold")
		close(finished)
	}()

	<-started
	if err := tester.TryRunStageByName("Hold"); err != ErrStageRunning {
		t.Fatalf("expected ErrStageRunning, got %v", err)
	}
	if got := tester.RunningStage(); got != "Hold" {
		t.Fatalf("RunningStage() = %q, want %q", got, "Hold")
	}

	close(release)
	<-finished
	if got := tester.RunningStage(); got != "" {
		t.Fatalf("RunningStage() after finish = %q, want empty", got)
	}
}
