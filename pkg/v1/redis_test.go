package v1

import (
	"fmt"
	"net"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"

	rms "github.com/XWinterVarit/integrate_tester_v2/pkg/redis-mock-server"
)

const testAccessKey = "test-key"

// startTestServer starts a miniredis and a redis-mock-server backed by it.
// Returns the server base URL and a cleanup function.
func startTestServer(t *testing.T) (string, func()) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	// Find a free port for the mock server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		mr.Close()
		t.Fatalf("failed to find free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	logger := rms.NewConsoleLogger()
	server := rms.NewRedisServer(port, testAccessKey, mr.Addr(), "", 0, logger)

	started := make(chan struct{})
	go func() {
		close(started)
		server.Start()
	}()
	<-started

	// Wait for server to be ready
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	c := rms.NewClient(baseURL, testAccessKey)
	for range 50 {
		if err := c.Ping(); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	return baseURL, func() {
		mr.Close()
	}
}

func TestRedisHelpers(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	client := ConnectRedis(baseURL, testAccessKey)

	client.Set("foo", "bar", time.Minute)
	client.ExpectValue("foo", "bar")

	client.Set("num", 123, 0)
	if got := client.Get("num"); got != "123" {
		t.Fatalf("expected num=123, got %s", got)
	}

	client.Del("foo")

	client.FlushAll()
}

func TestRedisExpectFound(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	client := ConnectRedis(baseURL, testAccessKey)

	client.Set("existing", "value", time.Minute)
	client.ExpectFound("existing")
}

func TestRedisExpectNotFound(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	client := ConnectRedis(baseURL, testAccessKey)

	client.ExpectNotFound("missing")
}

func TestRedisSetJsonField(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	client := ConnectRedis(baseURL, testAccessKey)

	// Store initial JSON
	client.Set("eeeee", `{"a":{"b":[{"c":"old"}]}}`, 0)

	// Set nested field
	client.SetJsonField("eeeee", "a.b[0].c", "something")

	// Verify via ExpectJsonField
	client.ExpectJsonField("eeeee", "a.b[0].c", "something")

	// Set a top-level field
	client.Set("obj", `{"name":"alice","score":10}`, 0)
	client.SetJsonField("obj", "score", 99)
	client.ExpectJsonField("obj", "score", float64(99))

	// Set a field two levels deep
	client.Set("deep", `{"x":{"y":{"z":"before"}}}`, 0)
	client.SetJsonField("deep", "x.y.z", "after")
	client.ExpectJsonField("deep", "x.y.z", "after")
}

func TestRedisSetJsonFieldFailsOnMissingKey(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	client := ConnectRedis(baseURL, testAccessKey)

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		client.SetJsonField("nonexistent", "a.b", "val")
	}()

	if !panicked {
		t.Fatal("expected Fail (panic) for missing key")
	}
}

func TestRedisExpectJsonFieldFailsOnDecodeError(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	client := ConnectRedis(baseURL, testAccessKey)
	client.Set("bad", "not-json", 0)

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		client.ExpectJsonField("bad", "a", "x")
	}()

	if !panicked {
		t.Fatal("expected Fail (panic) for decode error")
	}
}

func TestRedisHSetJsonField(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	client := ConnectRedis(baseURL, testAccessKey)

	// Store initial JSON in a hash field
	client.HSet("user:1", "profile", `{"name":"alice","score":10,"address":{"city":"Bangkok"}}`)

	// Set a top-level field
	client.HSetJsonField("user:1", "profile", "score", 99)
	client.HExpectJsonField("user:1", "profile", "score", float64(99))

	// Set a nested field
	client.HSetJsonField("user:1", "profile", "address.city", "Chiang Mai")
	client.HExpectJsonField("user:1", "profile", "address.city", "Chiang Mai")

	// Set a field with array index
	client.HSet("user:2", "data", `{"items":[{"id":1},{"id":2}]}`)
	client.HSetJsonField("user:2", "data", "items[0].id", 42)
	client.HExpectJsonField("user:2", "data", "items[0].id", float64(42))

	// Set deep nested path
	client.HSet("user:3", "config", `{"a":{"b":[{"c":"old"}]}}`)
	client.HSetJsonField("user:3", "config", "a.b[0].c", "new")
	client.HExpectJsonField("user:3", "config", "a.b[0].c", "new")
}

func TestRedisHSetJsonFieldFailsOnMissingField(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	client := ConnectRedis(baseURL, testAccessKey)

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		client.HSetJsonField("nonexistent", "field", "a.b", "val")
	}()

	if !panicked {
		t.Fatal("expected Fail (panic) for missing hash key/field")
	}
}

func TestRedisHExpectJsonFieldFailsOnDecodeError(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	client := ConnectRedis(baseURL, testAccessKey)
	client.HSet("myhash", "badjson", "not-json")

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		client.HExpectJsonField("myhash", "badjson", "a", "x")
	}()

	if !panicked {
		t.Fatal("expected Fail (panic) for decode error")
	}
}

func TestRedisHashHelpers(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	client := ConnectRedis(baseURL, testAccessKey)

	// HSet and HGet
	client.HSet("user:1", "name", "Alice")
	client.HSet("user:1", "age", "30")

	if got := client.HGet("user:1", "name"); got != "Alice" {
		t.Fatalf("expected name=Alice, got %s", got)
	}
	if got := client.HGet("user:1", "age"); got != "30" {
		t.Fatalf("expected age=30, got %s", got)
	}

	// HIncrement
	client.HSet("counters", "visits", "10")
	result := client.HIncrement("counters", "visits", 5)
	if result != 15 {
		t.Fatalf("expected visits=15 after increment, got %d", result)
	}
	if got := client.HGet("counters", "visits"); got != "15" {
		t.Fatalf("expected visits=15 in redis, got %s", got)
	}

	// HIncrement on non-existing field starts from 0
	result = client.HIncrement("counters", "newfield", 3)
	if result != 3 {
		t.Fatalf("expected newfield=3, got %d", result)
	}
}
