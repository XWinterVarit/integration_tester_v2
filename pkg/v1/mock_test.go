package v1

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestMockServer(t *testing.T) {
	// Pick a random port or let it pick one?
	// The RunMockServer takes a string port. "0" usually means random.
	// But the implementation checks ":".
	// Let's try ":0" if supported by implementation, or a fixed high port.
	// Implementation: "Starting Server on :0" -> http.Server{Addr: ":0"} -> valid.

	// Problem: How to get the actual port if ":0" is used?
	// MockServer struct has *http.Server but doesn't expose the listener or address easily if not stored.
	// The code: ms.server.ListenAndServe().

	// If I use a fixed port, I risk collision.
	// Let's try 8999.
	port := "8999"

	handler := func(req Request) Response {
		return NewResponse(201, "Created")
	}

	handlers := map[string]MockHandlerFunc{
		"/test": handler,
	}

	ms := RunMockServer(port, handlers)
	defer ms.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Make request
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/test", port))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Created" {
		t.Errorf("Expected body 'Created', got '%s'", string(body))
	}

	// Test update
	newHandlers := map[string]MockHandlerFunc{
		"/test2": func(req Request) Response { return NewResponse(200, "OK2") },
	}
	UpdateMockServer(ms, newHandlers)

	// Old handler should still be there (Merge strategy)
	resp, _ = http.Get(fmt.Sprintf("http://localhost:%s/test", port))
	if resp.StatusCode != 201 {
		t.Errorf("Expected old handler to persist, got %d", resp.StatusCode)
	}

	// New handler should be there
	resp, _ = http.Get(fmt.Sprintf("http://localhost:%s/test2", port))
	if resp.StatusCode != 200 {
		t.Errorf("Expected new handler to work, got %d", resp.StatusCode)
	}
}
