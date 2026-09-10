package v1

import "testing"

func TestRunGUI(t *testing.T) {
	// GUI testing requires a window system and interaction.
	// Skipping actual execution to avoid blocking or failure in headless env.
	t.Skip("Skipping GUI test")
}
