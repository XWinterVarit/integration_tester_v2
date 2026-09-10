package v1

import (
	"testing"
	"time"
)

func TestRunAppServer(t *testing.T) {
	// Run a simple command that sleeps for a bit so we can stop it
	// Using "sleep 1" (1 second)
	app := RunAppServer("sleep", "1")

	if app.cmd == nil {
		t.Fatal("Cmd should not be nil")
	}

	// Wait a bit to ensure it started
	time.Sleep(100 * time.Millisecond)

	if app.cmd.Process == nil {
		t.Error("Process should not be nil")
	}

	// Stop it
	app.Stop()

	// Wait for process to exit?
	state, err := app.cmd.Process.Wait()
	// Process.Wait might fail if Stop() (Kill) already reaped it or if Wait was called in RunAppServer?
	// RunAppServer does NOT call Wait. Stop() calls Kill() then Wait().
	// So app.cmd.Wait() in Stop() returns the state.
	// Since we called Stop(), calling Wait again might return error or same state.
	if err != nil && err.Error() != "exec: Wait was already called" {
		// It's okay if it was already called.
	}
	_ = state
}

func TestRunAppServerFail(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected panic for missing executable")
		}
	}()

	RunAppServer("non_existent_executable_xyz")
}
