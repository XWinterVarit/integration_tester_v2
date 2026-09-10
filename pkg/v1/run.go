package v1

import (
	"flag"
	"log"
	"os"
	"strings"
)

// Run modes accepted by Run and RunWithMode.
const (
	ModeGUI        = "gui"         // Electron + React desktop UI
	ModeCLI        = "cli"         // run every stage sequentially, then exit
	ModeCLICommand = "cli-command" // interactive stdin command session
	ModeServer     = "server"      // HTTP API + web UI in the browser
)

const modeFlagName = "mode"

// RegisterModeFlag registers a "-mode" flag on flag.CommandLine, defaulting to
// the INTEGRATION_TESTER_MODE / IT_MODE environment variable (or "gui"). Call it
// before flag.Parse() so the app can be started with, for example:
//
//	go run . -mode cli
//
// It is safe to call more than once; the first call wins.
func RegisterModeFlag(usage string) {
	if flag.Lookup(modeFlagName) != nil {
		return
	}
	if usage == "" {
		usage = "run mode: gui | cli | cli-command | server"
	}
	flag.String(modeFlagName, envMode(), usage)
}

// CurrentMode resolves the requested run mode from the "-mode" flag (when
// registered) or from the INTEGRATION_TESTER_MODE / IT_MODE environment
// variables, defaulting to ModeGUI.
func CurrentMode() string {
	if f := flag.Lookup(modeFlagName); f != nil {
		if v := strings.TrimSpace(f.Value.String()); v != "" {
			return strings.ToLower(v)
		}
	}
	if v := envMode(); v != "" {
		return strings.ToLower(v)
	}
	return ModeGUI
}

// Run starts the tester in the mode selected by CurrentMode. Apps that embed
// the tester can call Run and let the operator pick the mode at launch time
// instead of hard-coding RunGUI.
//
// Usage:
//
//	go run . -mode cli        # or cli-command / server / gui
//	INTEGRATION_TESTER_MODE=cli go run .
func Run(t *Tester) {
	RunWithMode(t, CurrentMode())
}

// RunWithMode starts the tester in an explicit mode.
func RunWithMode(t *Tester, mode string) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", ModeGUI:
		RunGUI(t)
	case ModeCLI:
		RunCLI(t)
	case ModeCLICommand, "command", "cli_command":
		RunCLICommand(t)
	case ModeServer, "web":
		RunServer(t)
	default:
		log.Fatalf("unknown run mode %q (expected: %s | %s | %s | %s)",
			mode, ModeGUI, ModeCLI, ModeCLICommand, ModeServer)
	}
}

func envMode() string {
	for _, key := range []string{"INTEGRATION_TESTER_MODE", "IT_MODE"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}
