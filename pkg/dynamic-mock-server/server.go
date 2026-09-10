package dynamic_mock_server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type MockServerInstance struct {
	Server *http.Server
	Port   int
}

type MockController struct {
	ControlPort int
	Servers     map[int]*MockServerInstance
	// Routes: Port -> Method -> Path -> Steps
	Routes map[int]map[string]map[string][]ResponseFuncConfig
	mu     sync.RWMutex
	Logger *Logger
}

func NewMockController(controlPort int, logger *Logger) *MockController {
	return &MockController{
		ControlPort: controlPort,
		Servers:     make(map[int]*MockServerInstance),
		Routes:      make(map[int]map[string]map[string][]ResponseFuncConfig),
		Logger:      logger,
	}
}

func (mc *MockController) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/registerRoute", mc.handleRegisterRoute)
	mux.HandleFunc("/resetPort", mc.handleResetPort)
	mux.HandleFunc("/resetAll", mc.handleResetAll)
	mux.HandleFunc("/", mc.handleNotFound)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", mc.ControlPort),
		Handler: mux,
	}

	mc.Logger.Log("ControlServerStart", 0, fmt.Sprintf("Starting control server on port %d", mc.ControlPort))
	return server.ListenAndServe()
}

func (mc *MockController) handleRegisterRoute(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Ensure route structure exists
	if _, ok := mc.Routes[req.Port]; !ok {
		mc.Routes[req.Port] = make(map[string]map[string][]ResponseFuncConfig)
	}
	if _, ok := mc.Routes[req.Port][req.Method]; !ok {
		mc.Routes[req.Port][req.Method] = make(map[string][]ResponseFuncConfig)
	}

	// Register/Replace route
	mc.Routes[req.Port][req.Method][req.Path] = req.ResponseFunc

	// Check if server exists, if not start it
	if _, ok := mc.Servers[req.Port]; !ok {
		if err := mc.startMockServerLocked(req.Port); err != nil {
			mc.Logger.Log("RegisterRouteError", time.Since(start), fmt.Sprintf("Failed to start server on port %d: %v", req.Port, err))
			http.Error(w, fmt.Sprintf("Failed to start server: %v", err), http.StatusInternalServerError)
			return
		}
	}

	details := map[string]interface{}{
		"port":   req.Port,
		"method": req.Method,
		"path":   req.Path,
		"status": "Registered/Replaced",
	}
	mc.Logger.Log("RegisterRoute", time.Since(start), details)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Route registered"})
}

func (mc *MockController) startMockServerLocked(port int) error {
	// Assumes mc.mu is locked
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mc.handleMockRequest(port, w, r)
		}),
	}

	instance := &MockServerInstance{
		Server: server,
		Port:   port,
	}
	mc.Servers[port] = instance

	go func() {
		mc.Logger.Log("MockServerStart", 0, fmt.Sprintf("Starting mock server on port %d", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			mc.Logger.Log("MockServerError", 0, fmt.Sprintf("Mock server on port %d failed: %v", port, err))
		}
	}()

	return nil
}

func (mc *MockController) handleResetPort(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req map[string]int
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	port := req["port"]

	mc.mu.Lock()

	// Remove routes
	delete(mc.Routes, port)

	// Stop server
	if instance, ok := mc.Servers[port]; ok {
		delete(mc.Servers, port)
		mc.mu.Unlock() // Unlock during shutdown to avoid deadlock if shutdown takes time

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := instance.Server.Shutdown(ctx); err != nil {
			mc.Logger.Log("ResetPortError", time.Since(start), fmt.Sprintf("Failed to shutdown port %d: %v", port, err))
		}
	} else {
		mc.mu.Unlock()
	}

	mc.Logger.Log("ResetPort", time.Since(start), map[string]int{"port": port})
	w.WriteHeader(http.StatusOK)
}

func (mc *MockController) handleResetAll(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	mc.mu.Lock()

	ports := make([]int, 0, len(mc.Servers))
	instances := make([]*MockServerInstance, 0, len(mc.Servers))

	for p, i := range mc.Servers {
		ports = append(ports, p)
		instances = append(instances, i)
	}

	// Clear all state
	mc.Servers = make(map[int]*MockServerInstance)
	mc.Routes = make(map[int]map[string]map[string][]ResponseFuncConfig)
	mc.mu.Unlock()

	var wg sync.WaitGroup
	for _, inst := range instances {
		wg.Add(1)
		go func(s *http.Server) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s.Shutdown(ctx)
		}(inst.Server)
	}
	wg.Wait()

	mc.Logger.Log("ResetAll", time.Since(start), map[string]interface{}{"ports_reset": ports})
	w.WriteHeader(http.StatusOK)
}

func (mc *MockController) handleMockRequest(port int, w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Lookup route
	mc.mu.RLock()
	var steps []ResponseFuncConfig
	if portRoutes, ok := mc.Routes[port]; ok {
		if methodRoutes, ok := portRoutes[r.Method]; ok {
			if s, ok := methodRoutes[r.URL.Path]; ok {
				steps = s
			}
		}
	}
	mc.mu.RUnlock()

	if steps == nil {
		http.NotFound(w, r)
		mc.Logger.Log("MockRequest", time.Since(start), map[string]interface{}{
			"port": port, "method": r.Method, "path": r.URL.Path, "status": 404,
		})
		return
	}

	executor := NewHandlerExecutor(w, r)
	err := executor.Execute(steps)
	if err != nil {
		mc.Logger.Log("MockRequestError", time.Since(start), fmt.Sprintf("Error executing steps: %v", err))
		http.Error(w, fmt.Sprintf("Mock error: %v", err), http.StatusInternalServerError)
		return
	}

	executor.Finalize()

	mc.Logger.Log("MockRequest", time.Since(start), map[string]interface{}{
		"port": port, "method": r.Method, "path": r.URL.Path, "status": executor.StatusCode,
		"variables": executor.Variables,
	})
}

func (mc *MockController) handleNotFound(w http.ResponseWriter, r *http.Request) {
	mc.Logger.Log("ControlRequest", 0, map[string]interface{}{
		"path":   r.URL.Path,
		"method": r.Method,
		"status": 404,
	})
	http.NotFound(w, r)
}
