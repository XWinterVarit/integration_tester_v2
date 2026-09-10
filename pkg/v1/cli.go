package v1

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// RunCLI runs all stages sequentially without a GUI, printing results to stdout.
// If any stage fails, it prints the error and exits with code 1 after all stages complete.
// Usage: call RunCLI(t) instead of RunGUI(t) when running in headless/CI mode.
func RunCLI(t *Tester) {
	fmt.Println("=== Integration Test (CLI Mode) ===")
	failed := 0
	for _, s := range t.Stages {
		fmt.Printf("\n[STAGE] %s\n", s.Name)
		err := t.RunStageByName(s.Name)
		if err != nil {
			fmt.Printf("  FAILED: %v\n", err)
			failed++
		} else {
			fmt.Printf("  PASSED\n")
		}
	}
	fmt.Printf("\n=== Results: %d/%d stages passed ===\n", len(t.Stages)-failed, len(t.Stages))
	if failed > 0 {
		os.Exit(1)
	}
}

// RunCLICommand starts an interactive command-line session that reads commands from stdin.
// This allows an external process (e.g. an AI agent) to run specific stages by name,
// just like a user clicking on a stage in the GUI.
//
// Supported commands:
//
//	list                  — print all stage names
//	run <stage name>      — run a specific stage by name
//	exit / quit           — exit the session
//
// Each command produces a response terminated by a blank line, so the caller can detect
// when the output for a command is complete.
func RunCLICommand(t *Tester) {
	fmt.Println("=== Integration Test (Command Mode) ===")
	fmt.Println("Commands: list | run <stage name> | exit")
	fmt.Println("Ready.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		switch {
		case line == "list":
			for _, s := range t.Stages {
				fmt.Printf("  %s\n", s.Name)
			}
			fmt.Println()

		case strings.HasPrefix(line, "run "):
			stageName := strings.TrimPrefix(line, "run ")
			stageName = strings.TrimSpace(stageName)
			fmt.Printf("[STAGE] %s\n", stageName)
			err := t.RunStageByName(stageName)
			if err != nil {
				fmt.Printf("FAILED: %v\n", err)
			} else {
				fmt.Printf("PASSED\n")
			}
			fmt.Println()

		case line == "exit", line == "quit":
			fmt.Println("Bye.")
			return

		default:
			fmt.Printf("Unknown command: %s\n", line)
			fmt.Println()
		}
	}
}
