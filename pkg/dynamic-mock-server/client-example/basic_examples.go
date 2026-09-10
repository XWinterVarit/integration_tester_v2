package main

import (
	"fmt"
	"io"
	"net/http"

	dms "github.com/XWinterVarit/integrate_tester_v2/pkg/dynamic-mock-server"
)

func runBasicExamples(client *dms.Client) {
	// Example 1: Simple GET request
	fmt.Println("1. Registering simple GET /hello")
	err := client.RegisterRoute(MockPort, "GET", "/hello", []dms.ResponseFuncConfig{
		dms.SetStatusCode("", 200),
		dms.SetJsonBody("", `{"message": "Hello World"}`),
		dms.SetHeader("", "Content-Type", "application/json"),
	})
	if err != nil {
		fmt.Printf("Error registering route: %v\n", err)
		return
	}

	// Verify
	printRequest("GET", fmt.Sprintf("http://localhost:%d/hello", MockPort), nil)

	// Example 2: POST request with different status code
	fmt.Println("2. Registering POST /created")
	err = client.RegisterRoute(MockPort, "POST", "/created", []dms.ResponseFuncConfig{
		dms.SetStatusCode("", 201),
		dms.SetJsonBody("", `{"id": 123, "status": "created"}`),
	})
	if err != nil {
		fmt.Printf("Error registering route: %v\n", err)
		return
	}

	// Verify
	printRequest("POST", fmt.Sprintf("http://localhost:%d/created", MockPort), nil)
}

// Helper to call the mock and print result
func printRequest(method, url string, body io.Reader) {
	req, _ := http.NewRequest(method, url, body)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response [%d]: %s\n", resp.StatusCode, string(b))
}
