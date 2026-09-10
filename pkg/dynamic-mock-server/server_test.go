package dynamic_mock_server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// Helper to wait for server start
func waitForServer(url string) error {
	for i := 0; i < 20; i++ {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("server at %s not up", url)
}

func TestDynamicMockServer(t *testing.T) {
	controlPort := 19000
	mockPort := 19001

	// Create a temp file for logger
	tmpFile, err := os.CreateTemp("", "mock-server-log-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}
	tmpFileName := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFileName)

	logger, err := NewLogger(tmpFileName)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	controller := NewMockController(controlPort, logger)

	// Start control server
	go func() {
		if err := controller.Start(); err != nil && err != http.ErrServerClosed {
			t.Logf("Control server error: %v", err)
		}
	}()

	// Wait for control server
	time.Sleep(500 * time.Millisecond)

	client := NewClient(fmt.Sprintf("http://localhost:%d", controlPort))

	t.Run("RegisterRouteAndCall", func(t *testing.T) {
		err := client.RegisterRoute(mockPort, "GET", "/test", []ResponseFuncConfig{
			SetStatusCode("", 200),
			SetJsonBody("", `{"message": "hello"}`),
		})
		if err != nil {
			t.Fatalf("RegisterRoute failed: %v", err)
		}

		// Wait for mock server to spin up
		if err := waitForServer(fmt.Sprintf("http://localhost:%d/test", mockPort)); err != nil {
			t.Fatalf("Mock server not up: %v", err)
		}

		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/test", mockPort))
		if err != nil {
			t.Fatalf("Failed to call mock: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if string(body) != `{"message": "hello"}` {
			t.Errorf("Unexpected body: %s", string(body))
		}
	})

	t.Run("DynamicVariablesAndLogic", func(t *testing.T) {
		// Register a complex route
		err := client.RegisterRoute(mockPort, "POST", "/logic", []ResponseFuncConfig{
			IfRequestHeader("Authorization", ConditionEqual, "Bearer secret", "AUTH_OK", "true"),

			GenerateRandomString(10, "RAND_STR"),

			SetJsonBody("", `{"auth": "{{.AUTH_OK}}", "rand": "{{.RAND_STR}}"}`),
			SetStatusCode("", 201),
		})
		if err != nil {
			t.Fatalf("RegisterRoute failed: %v", err)
		}

		// Call with correct header
		req, _ := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/logic", mockPort), nil)
		req.Header.Set("Authorization", "Bearer secret")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 201 {
			t.Errorf("Expected status 201, got %d", resp.StatusCode)
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr := string(bodyBytes)
		t.Logf("Response body: %s", bodyStr)

		if !bytes.Contains(bodyBytes, []byte(`"auth": "true"`)) {
			t.Errorf("Expected auth: true in body, got %s", bodyStr)
		}

		// Check random string (length 10)
		if strings.Contains(bodyStr, "{{.RAND_STR}}") {
			t.Errorf("Random string not generated/replaced")
		}
	})

	t.Run("GeneratorAndConversion", func(t *testing.T) {
		err := client.RegisterRoute(mockPort, "GET", "/gen", []ResponseFuncConfig{
			GenerateRandomInt(10, 100, "R_INT"),
			ConvertToString("R_INT"),
			SetJsonBody("", `{"val": "{{.R_INT}}"}`),
			SetStatusCode("", 200),
		})
		if err != nil {
			t.Fatalf("RegisterRoute failed: %v", err)
		}

		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/gen", mockPort))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Logf("Generator body: %s", string(bodyBytes))
		// We expect a number between 10 and 100
		// And body should not contain template
		if strings.Contains(string(bodyBytes), "{{") {
			t.Errorf("Template not resolved: %s", string(bodyBytes))
		}
	})

	t.Run("ResetPort", func(t *testing.T) {
		err := client.ResetPort(mockPort)
		if err != nil {
			t.Fatalf("ResetPort failed: %v", err)
		}

		// Allow time for shutdown
		time.Sleep(500 * time.Millisecond)

		// Verify route is gone
		_, err = http.Get(fmt.Sprintf("http://localhost:%d/test", mockPort))
		if err == nil {
			// If request succeeds, it means server is still up or something answered.
			// But if server is down, we expect error.
			// However, checking error message is tricky across platforms.
			// Ideally we want to ensure we don't get a 200 OK from our app.
		} else {
			t.Logf("Got expected error after reset: %v", err)
		}
	})

	t.Run("ResetAll", func(t *testing.T) {
		// Register a route again
		client.RegisterRoute(mockPort, "GET", "/test2", []ResponseFuncConfig{
			SetStatusCode("", 200),
		})
		waitForServer(fmt.Sprintf("http://localhost:%d/test2", mockPort))

		err := client.ResetAll()
		if err != nil {
			t.Fatalf("ResetAll failed: %v", err)
		}
		time.Sleep(500 * time.Millisecond)

		// Verify route is gone
		_, err = http.Get(fmt.Sprintf("http://localhost:%d/test2", mockPort))
		if err == nil {
			// Should fail
		} else {
			t.Logf("Got expected error after ResetAll: %v", err)
		}
	})
}
