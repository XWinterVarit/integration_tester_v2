package v1

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Defaults for the interactive command-mode session.
const (
	defaultCLIHeartbeat = 5 * time.Second
	defaultCLISlowAfter = time.Minute
)

// CLICommandOptions configures RunCLICommand / RunCLICommandIO. The zero value
// uses the defaults below. A negative duration disables that behaviour.
//
//   - Heartbeat: interval between "still running" progress lines (default 5s).
//   - SlowAfter: elapsed time after which a run is flagged SLOW (default 1m).
//   - Timeout:   hard limit; when exceeded the process exits to guarantee no
//     overlapping runs (default 0 = disabled).
type CLICommandOptions struct {
	Heartbeat time.Duration
	SlowAfter time.Duration
	Timeout   time.Duration
}

// RunCLI runs all stages sequentially without a GUI, printing results to stdout.
// If any stage fails, it prints the error and exits with code 1 after all stages complete.
// Stages are serialized by the tester, so they never overlap.
func RunCLI(t *Tester) {
	fmt.Println("=== Integration Test (CLI Mode) ===")
	failed := 0
	for _, s := range t.Stages {
		fmt.Printf("\n[STAGE] %s\n", s.Name)
		start := time.Now()
		err := t.RunStageByName(s.Name)
		duration := time.Since(start)
		if err != nil {
			fmt.Printf("  FAILED (%s): %v\n", fmtDuration(duration), err)
			failed++
		} else {
			fmt.Printf("  PASSED (%s)\n", fmtDuration(duration))
		}
	}
	fmt.Printf("\n=== Results: %d/%d stages passed ===\n", len(t.Stages)-failed, len(t.Stages))
	if failed > 0 {
		os.Exit(1)
	}
}

// RunCLICommand starts an interactive command-line session that reads commands
// from stdin. It is designed for an external driver (e.g. an AI agent) to run
// and debug individual stages.
//
// Only one stage runs at a time. When a stage is running, the session reports
// progress so the caller always knows what is happening:
//
//	RUNNING <stage>                                  stage started
//	HEARTBEAT <stage> elapsed=5s still-running       periodic progress
//	SLOW <stage> elapsed=1m0s still-running ...      exceeds the slow threshold
//	PASSED <stage> duration=12.3s                    finished successfully
//	FAILED <stage> duration=3.1s error=...           finished with an error
//	BUSY running=<stage> elapsed=... requested=...   run rejected, one is active
//
// Supported commands:
//
//	list             print all stage names
//	run <stage>      start a stage (rejected with BUSY if one is running)
//	status           print the current/last run and its elapsed/duration
//	exit / quit      leave the session (waits for any active run to finish)
//
// Responses are line-oriented and terminated by a blank line. A "run" streams
// RUNNING/HEARTBEAT/SLOW lines and ends with PASSED/FAILED plus a blank line.
func RunCLICommand(t *Tester) {
	RunCLICommandWithOptions(t, CLICommandOptions{})
}

// RunCLICommandWithOptions is RunCLICommand with explicit timing options.
func RunCLICommandWithOptions(t *Tester, opts CLICommandOptions) {
	RunCLICommandIO(t, os.Stdin, os.Stdout, opts)
}

// RunCLICommandIO is RunCLICommand with explicit input/output streams (useful
// for embedding and tests).
func RunCLICommandIO(t *Tester, in io.Reader, out io.Writer, opts CLICommandOptions) {
	opts = cliOptionsFromEnv(opts)
	if opts.Heartbeat == 0 {
		opts.Heartbeat = defaultCLIHeartbeat
	}
	if opts.SlowAfter == 0 {
		opts.SlowAfter = defaultCLISlowAfter
	}

	session := &cliSession{tester: t, out: &lockedWriter{w: out}, opts: opts}

	session.emit("=== Integration Test (Command Mode) ===")
	session.emit("Commands: list | run <stage> | status | exit")
	session.respond("Ready.")

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		switch {
		case line == "list":
			session.respond(session.listLines()...)

		case line == "status":
			session.respond(session.statusLines()...)

		case strings.HasPrefix(line, "run "):
			session.handleRun(strings.TrimSpace(strings.TrimPrefix(line, "run ")))

		case line == "exit", line == "quit":
			if cur := session.currentStage(); cur != "" {
				session.emit("EXIT waiting for " + cur + " to finish...")
			}
			session.waitIdle()
			session.emit("Bye.")
			return

		default:
			session.respond("ERROR unknown command: " + line)
		}
	}

	// stdin closed: never leave a stage running.
	session.waitIdle()
}

type cliSession struct {
	tester *Tester
	out    io.Writer
	opts   CLICommandOptions

	mu           sync.Mutex
	runningStage string
	startedAt    time.Time
	lastStage    string
	lastResult   string
	lastDuration time.Duration
	lastError    string
}

// emit writes a single line (used for streaming events).
func (s *cliSession) emit(line string) {
	fmt.Fprintln(s.out, line)
}

// respond writes a set of lines followed by a blank terminator, as one atomic
// write so concurrent progress lines cannot split the response.
func (s *cliSession) respond(lines ...string) {
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	fmt.Fprint(s.out, b.String())
}

func (s *cliSession) listLines() []string {
	s.tester.mu.Lock()
	defer s.tester.mu.Unlock()
	lines := make([]string, 0, len(s.tester.Stages))
	for _, st := range s.tester.Stages {
		lines = append(lines, "  "+st.Name)
	}
	return lines
}

func (s *cliSession) hasStage(name string) bool {
	s.tester.mu.Lock()
	defer s.tester.mu.Unlock()
	for _, st := range s.tester.Stages {
		if st.Name == name {
			return true
		}
	}
	return false
}

func (s *cliSession) currentStage() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.runningStage
}

func (s *cliSession) statusLines() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runningStage != "" {
		return []string{fmt.Sprintf("STATUS RUNNING stage=%s elapsed=%s", s.runningStage, fmtDuration(time.Since(s.startedAt)))}
	}
	if s.lastStage == "" {
		return []string{"STATUS IDLE"}
	}
	if s.lastError != "" {
		return []string{fmt.Sprintf("STATUS IDLE last=%s result=%s duration=%s error=%s",
			s.lastStage, s.lastResult, fmtDuration(s.lastDuration), s.lastError)}
	}
	return []string{fmt.Sprintf("STATUS IDLE last=%s result=%s duration=%s",
		s.lastStage, s.lastResult, fmtDuration(s.lastDuration))}
}

func (s *cliSession) handleRun(stage string) {
	if stage == "" {
		s.respond("ERROR missing stage name")
		return
	}
	if !s.hasStage(stage) {
		s.respond("ERROR unknown stage: " + stage)
		return
	}

	s.mu.Lock()
	if s.runningStage != "" {
		running, started := s.runningStage, s.startedAt
		s.mu.Unlock()
		s.respond(fmt.Sprintf("BUSY running=%s elapsed=%s requested=%s",
			running, fmtDuration(time.Since(started)), stage))
		return
	}
	s.runningStage = stage
	s.startedAt = time.Now()
	s.mu.Unlock()

	// Acknowledge immediately; progress and the result stream asynchronously.
	s.emit("RUNNING " + stage)
	go s.execute(stage)
}

func (s *cliSession) execute(stage string) {
	start := time.Now()

	stop := make(chan struct{})
	var heartbeat sync.WaitGroup
	if s.opts.Heartbeat > 0 {
		heartbeat.Add(1)
		go func() {
			defer heartbeat.Done()
			ticker := time.NewTicker(s.opts.Heartbeat)
			defer ticker.Stop()
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					elapsed := time.Since(start)
					if s.opts.SlowAfter > 0 && elapsed >= s.opts.SlowAfter {
						s.emit(fmt.Sprintf("SLOW %s elapsed=%s still-running slow-threshold=%s",
							stage, fmtDuration(elapsed), fmtDuration(s.opts.SlowAfter)))
					} else {
						s.emit(fmt.Sprintf("HEARTBEAT %s elapsed=%s still-running", stage, fmtDuration(elapsed)))
					}
				}
			}
		}()
	}

	err := s.runWithTimeout(stage)
	close(stop)
	heartbeat.Wait()
	duration := time.Since(start)

	s.mu.Lock()
	s.runningStage = ""
	s.lastStage = stage
	s.lastDuration = duration
	if err != nil {
		s.lastResult = "FAILED"
		s.lastError = err.Error()
	} else {
		s.lastResult = "PASSED"
		s.lastError = ""
	}
	s.mu.Unlock()

	if err != nil {
		s.respond(fmt.Sprintf("FAILED %s duration=%s error=%s", stage, fmtDuration(duration), err))
		return
	}
	s.respond(fmt.Sprintf("PASSED %s duration=%s", stage, fmtDuration(duration)))
}

// runWithTimeout executes the stage. When Timeout is set and exceeded, the
// stage goroutine cannot be cancelled safely, so the process exits to guarantee
// that no second stage ever overlaps the stuck one.
func (s *cliSession) runWithTimeout(stage string) error {
	if s.opts.Timeout <= 0 {
		return s.tester.RunStageByName(stage)
	}
	done := make(chan error, 1)
	go func() { done <- s.tester.RunStageByName(stage) }()
	select {
	case err := <-done:
		return err
	case <-time.After(s.opts.Timeout):
		s.emit(fmt.Sprintf("TIMEOUT %s exceeded=%s aborting-to-avoid-overlapping-runs",
			stage, fmtDuration(s.opts.Timeout)))
		os.Exit(2)
		return nil
	}
}

func (s *cliSession) waitIdle() {
	for {
		s.mu.Lock()
		running := s.runningStage
		s.mu.Unlock()
		if running == "" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// lockedWriter serializes writes so progress lines and responses never corrupt
// each other when several goroutines emit at once.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

func fmtDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	return d.Round(time.Millisecond).String()
}

func cliOptionsFromEnv(opts CLICommandOptions) CLICommandOptions {
	override := func(key string, dst *time.Duration) {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			return
		}
		if d, err := time.ParseDuration(v); err == nil {
			*dst = d
		}
	}
	override("INTEGRATION_TESTER_CLI_HEARTBEAT", &opts.Heartbeat)
	override("INTEGRATION_TESTER_CLI_SLOW_AFTER", &opts.SlowAfter)
	override("INTEGRATION_TESTER_CLI_TIMEOUT", &opts.Timeout)
	return opts
}
