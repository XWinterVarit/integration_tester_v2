package main

import (
	"fmt"
	"net/http"
	"time"

	dms "github.com/XWinterVarit/integrate_tester_v2/pkg/dynamic-mock-server"
)

func runAdvancedExamples(client *dms.Client) {
	// Example 1: Latency Simulation
	fmt.Println("1. Latency: Random wait 500ms-1000ms")
	err := client.RegisterRoute(MockPort, "GET", "/slow", []dms.ResponseFuncConfig{
		dms.SetRandomWait("", 500, 1000),
		dms.SetJsonBody("", `{"status": "done"}`),
		dms.SetStatusCode("", 200),
	})
	if err != nil {
		fmt.Printf("Error registering: %v\n", err)
		return
	}

	start := time.Now()
	printRequest("GET", fmt.Sprintf("http://localhost:%d/slow", MockPort), nil)
	fmt.Printf("Request took: %v\n", time.Since(start))

	// Example 2: Copying Headers (Trace ID)
	fmt.Println("2. Copy Headers: X-Trace-ID")
	err = client.RegisterRoute(MockPort, "GET", "/trace", []dms.ResponseFuncConfig{
		dms.CopyHeaderFromRequest("", "X-Trace-ID"),
		dms.SetJsonBody("", `{"status": "traced"}`),
		dms.SetStatusCode("", 200),
	})
	if err != nil {
		fmt.Printf("Error registering: %v\n", err)
		return
	}

	req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/trace", MockPort), nil)
	req.Header.Set("X-Trace-ID", "trace-12345")

	// Custom print to show headers
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("Response Header X-Trace-ID: %s\n", resp.Header.Get("X-Trace-ID"))

	// Example 3: Reset Port
	fmt.Println("3. Reset Port (Teardown)")
	err = client.ResetPort(MockPort)
	if err != nil {
		fmt.Printf("Error resetting port: %v\n", err)
		return
	}

	// Verify 404 or error
	resp, err = http.Get(fmt.Sprintf("http://localhost:%d/trace", MockPort))
	if err != nil {
		fmt.Printf("Request correctly failed (server down/reset): %v\n", err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == 404 {
			fmt.Println("Got 404 as expected (route cleared but server might still be up if only routes cleared?)")
			// Actually ResetPort stops the server instance in the current implementation.
			// So connection refused is likely unless auto-restart or something.
			// Let's check implementation: handleResetPort stops server.
			// So we expect connection error.
		} else {
			fmt.Printf("Unexpected response: %d\n", resp.StatusCode)
		}
	}
}
