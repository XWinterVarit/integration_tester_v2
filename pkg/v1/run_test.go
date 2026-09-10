package v1

import "testing"

func TestCurrentModeFromEnv(t *testing.T) {
	t.Setenv("INTEGRATION_TESTER_MODE", "CLI")
	t.Setenv("IT_MODE", "")
	if got := CurrentMode(); got != ModeCLI {
		t.Fatalf("expected %q, got %q", ModeCLI, got)
	}
}

func TestCurrentModeDefault(t *testing.T) {
	t.Setenv("INTEGRATION_TESTER_MODE", "")
	t.Setenv("IT_MODE", "")
	if got := CurrentMode(); got != ModeGUI {
		t.Fatalf("expected default %q, got %q", ModeGUI, got)
	}
}
