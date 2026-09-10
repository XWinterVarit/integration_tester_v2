package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	dms "github.com/XWinterVarit/integrate_tester_v2/pkg/dynamic-mock-server"
)

const (
	ControlPort = 20000
	MockPort    = 20001
)

func main() {
	// 1. Start the Mock Server Controller (Server-side)
	// In a real scenario, this would be running as a separate service.
	// We start it here so this example is self-contained and runnable.
	startMockServerController()

	// 2. Initialize the Client
	// This is what the user would do in their test code.
	client := dms.NewClient(fmt.Sprintf("http://localhost:%d", ControlPort))
	fmt.Println("Client initialized.")

	// 3. Run Examples
	// We pause briefly to ensure server is up (though startMockServerController handles it partially)
	time.Sleep(1 * time.Second)

	fmt.Println("--- Running Basic Examples ---")
	runBasicExamples(client)

	fmt.Println("\n--- Running Conditional Examples ---")
	runConditionalExamples(client)

	fmt.Println("\n--- Running Generator Examples ---")
	runGeneratorExamples(client)

	fmt.Println("\n--- Running Advanced Examples ---")
	runAdvancedExamples(client)

	fmt.Println("\n--- Running Complex Examples ---")
	runComplexExamples(client)

	fmt.Println("\n--- Running Case Examples ---")
	runCaseExamples(client)

	fmt.Println("\n--- Running New Features Examples ---")
	runNewFeaturesExamples(client)

	fmt.Println("\n--- Running Extended Conditions Examples ---")
	runExtendedConditionsExamples(client)

	fmt.Println("\nAll examples executed successfully.")
}

func startMockServerController() {
	// Using a temp file for logging to avoid clutter
	logger, err := dms.NewLogger("mock_server_example.log")
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	controller := dms.NewMockController(ControlPort, logger)
	go func() {
		if err := controller.Start(); err != nil && err != http.ErrServerClosed {
			log.Printf("Control server stopped with error: %v", err)
		}
	}()
	fmt.Printf("Mock Controller started on port %d\n", ControlPort)
}
