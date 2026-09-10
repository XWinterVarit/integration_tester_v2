package v1

import (
	"fmt"
	"sync"
)

// StageFunc represents the function to be executed in a stage.
type StageFunc func()

// StageDef represents a defined stage.
type StageDef struct {
	Name string
	Func StageFunc
}

// Action represents a runnable operation within a stage.
type Action struct {
	Summary string
	Func    func()
}

var (
	// stageActions maps StageName -> List of Actions
	stageActions = make(map[string][]Action)
	// currentStage tracks the currently running stage name
	currentStage string
	// isRecording determines if operations should be recorded
	isRecording bool
	// actionMu protects the global state
	actionMu sync.Mutex
	// actionHandlers are notified when actions list updates
	actionHandlers []func()
	// isDryRun indicates if the tester is in discovery mode
	isDryRun bool
	// dryRunMu ensures RunStageByName waits for any in-progress dry run to finish
	dryRunMu sync.RWMutex
)

// IsDryRun checks if the tester is in dry run mode.
func IsDryRun() bool {
	actionMu.Lock()
	defer actionMu.Unlock()
	return isDryRun
}

// RecordAction registers an operation for the current stage.
func RecordAction(summary string, fn func()) {
	actionMu.Lock()
	defer actionMu.Unlock()

	if !isRecording || currentStage == "" {
		return
	}

	stageActions[currentStage] = append(stageActions[currentStage], Action{
		Summary: summary,
		Func:    fn,
	})

	notifyActionHandlers()
}

// GetStageActions returns the recorded actions for a stage.
func GetStageActions(stageName string) []Action {
	actionMu.Lock()
	defer actionMu.Unlock()
	// Return copy to be safe
	src := stageActions[stageName]
	dst := make([]Action, len(src))
	copy(dst, src)
	return dst
}

// RegisterActionUpdateHandler adds a listener for action updates.
func RegisterActionUpdateHandler(fn func()) {
	actionMu.Lock()
	defer actionMu.Unlock()
	actionHandlers = append(actionHandlers, fn)
}

func notifyActionHandlers() {
	for _, h := range actionHandlers {
		h()
	}
}

// Tester is the main struct for the integration test library.
type Tester struct {
	Stages []StageDef
	mu     sync.Mutex
}

// NewTester creates a new Tester instance.
func NewTester() *Tester {
	return &Tester{
		Stages: make([]StageDef, 0),
	}
}

// Stage registers a new stage.
func (t *Tester) Stage(name string, fn StageFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Stages = append(t.Stages, StageDef{Name: name, Func: fn})
}

// RunStageByName runs a specific stage by name.
// It waits for any in-progress dry run to finish before executing.
func (t *Tester) RunStageByName(name string) (err error) {
	// Wait for any ongoing dry run to complete
	dryRunMu.RLock()
	defer dryRunMu.RUnlock()

	t.mu.Lock()
	var fn StageFunc
	for _, s := range t.Stages {
		if s.Name == name {
			fn = s.Func
			break
		}
	}
	t.mu.Unlock()

	if fn == nil {
		return fmt.Errorf("stage %s not found", name)
	}

	// Setup context for recording
	actionMu.Lock()
	currentStage = name
	isRecording = true
	stageActions[name] = []Action{} // Clear previous actions
	notifyActionHandlers()
	actionMu.Unlock()

	Log(LogTypeStage, fmt.Sprintf("Running Stage: %s", name), "")

	// Ensure recording stops after stage
	defer func() {
		actionMu.Lock()
		isRecording = false
		currentStage = ""
		actionMu.Unlock()
	}()

	// Error handling in stages should be handled by panic/recover or other means if we want to stop execution
	// For this lib, we assume stages might panic on failure.
	defer func() {
		if r := recover(); r != nil {
			if te, ok := r.(TestError); ok {
				Log(LogTypeStage, fmt.Sprintf("Stage %s FAILED", name), te.Message)
				err = fmt.Errorf("failed: %s", te.Message)
			} else {
				Log(LogTypeStage, fmt.Sprintf("Stage %s FAILED (Crash)", name), fmt.Sprintf("%v", r))
				err = fmt.Errorf("panic: %v", r)
			}
		} else {
			Log(LogTypeStage, fmt.Sprintf("Stage %s PASSED", name), "")
		}
	}()
	fn()
	return nil
}

// DryRunAll executes all stages in dry run mode to discover actions.
func (t *Tester) DryRunAll() {
	for _, s := range t.Stages {
		t.DryRunStage(s)
	}
}

// DryRunStage executes a single stage in dry run mode.
func (t *Tester) DryRunStage(s StageDef) {
	dryRunMu.Lock()
	defer dryRunMu.Unlock()

	actionMu.Lock()
	currentStage = s.Name
	isRecording = true
	isDryRun = true
	stageActions[s.Name] = []Action{}
	actionMu.Unlock()

	defer func() {
		actionMu.Lock()
		isRecording = false
		isDryRun = false
		currentStage = ""
		actionMu.Unlock()
		// Catch panics during dry run
		recover()
	}()

	s.Func()
}
