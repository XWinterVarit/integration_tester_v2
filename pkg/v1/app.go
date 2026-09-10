package v1

import (
	"fmt"
	"os"
	"os/exec"
)

// AppServer represents a running application server.
type AppServer struct {
	cmd *exec.Cmd
}

// RunAppServer runs the application server.
func RunAppServer(path string, args ...string) *AppServer {
	RecordAction(fmt.Sprintf("App Run: %s", path), func() { RunAppServer(path, args...) })
	if IsDryRun() {
		return &AppServer{}
	}
	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	Logf(LogTypeApp, "Starting Server: %s %v", path, args)
	if err := cmd.Start(); err != nil {
		Fail("Failed to start server: %v", err)
	}

	return &AppServer{cmd: cmd}
}

// Stop stops the application server.
func (s *AppServer) Stop() {
	if s.cmd != nil && s.cmd.Process != nil {
		Log(LogTypeApp, "Stopping Server", "")
		s.cmd.Process.Kill()
		s.cmd.Wait() // release resources
	}
}
