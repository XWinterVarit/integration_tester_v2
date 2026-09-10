package v1

import (
	"fmt"
	"net/http"
	"sync"
)

// MockHandlerFunc defines the handler function signature.
type MockHandlerFunc func(Request) Response

// MockServer represents a running mock server.
type MockServer struct {
	server   *http.Server
	handlers map[string]MockHandlerFunc
	mu       sync.RWMutex
}

// RunMockServer starts a mock server on the specified port with given handlers.
// port can be ":8080" or just "8080".
func RunMockServer(port string, handlers map[string]MockHandlerFunc) *MockServer {
	RecordAction(fmt.Sprintf("Mock Run: %s", port), func() { RunMockServer(port, handlers) })
	if IsDryRun() {
		return &MockServer{}
	}
	if len(port) > 0 && port[0] != ':' {
		port = ":" + port
	}

	ms := &MockServer{
		handlers: handlers,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ms.handle)

	ms.server = &http.Server{
		Addr:    port,
		Handler: mux,
	}

	go func() {
		Logf(LogTypeMock, "Starting Server on %s", port)
		if err := ms.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			Log(LogTypeMock, "Server failed", fmt.Sprintf("%v", err))
		}
	}()

	return ms
}

// UpdateMockServer updates the handlers of an existing mock server.
// It merges or replaces? The requirement says "UpdateMockServer".
// Usually replacing the map is safer/cleaner for a "stage" change.
func UpdateMockServer(ms *MockServer, handlers map[string]MockHandlerFunc) {
	RecordAction("Mock Update", func() { UpdateMockServer(ms, handlers) })
	if IsDryRun() {
		return
	}
	if ms.server == nil {
		Fail("MockServer is not running (possibly running a DryRun captured action without real execution context)")
	}
	Log(LogTypeMock, "Updating server handlers", "")
	ms.mu.Lock()
	defer ms.mu.Unlock()
	// We can merge or replace.
	// "Update" might imply adding/overwriting specific paths.
	// But resetting behavior for a new stage usually means "this is the new state".
	// Let's replace the whole map if that's what passed, or merge?
	// The example:
	// UpdateMockServer(mockA, paths["/a": func...])
	// If only "/a" is passed, does "/b" still exist?
	// Usually in tests, you want to override specific behaviors.
	// I'll implement Merge strategy (Update/Add).
	if ms.handlers == nil {
		ms.handlers = make(map[string]MockHandlerFunc)
	}
	for k, v := range handlers {
		ms.handlers[k] = v
	}
}

func (ms *MockServer) handle(w http.ResponseWriter, r *http.Request) {
	ms.mu.RLock()
	handler, ok := ms.handlers[r.URL.Path]
	ms.mu.RUnlock()

	if !ok {
		// Try generic catch-all if needed? Or 404.
		// For now 404.
		Logf(LogTypeMock, "Handled Request: %s %s -> 404 Not Found", r.Method, r.URL.Path)
		http.NotFound(w, r)
		return
	}

	reqWrapper := NewRequestWrapper(r)
	resp := handler(reqWrapper)

	Log(LogTypeMock, fmt.Sprintf("Handled Request: %s %s -> %d", r.Method, r.URL.Path, resp.StatusCode), fmt.Sprintf("Response Body: %s\nHeaders: %v", resp.Body, resp.Header))

	for k, v := range resp.Header {
		w.Header().Set(k, v)
	}
	w.WriteHeader(resp.StatusCode)
	w.Write([]byte(resp.Body))
}

// Stop stops the mock server.
func (ms *MockServer) Stop() {
	if ms.server != nil {
		ms.server.Close()
	}
}
