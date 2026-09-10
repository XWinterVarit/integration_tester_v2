package v1

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

// localTokenHeader is an alternative to the Authorization header for clients
// (like EventSource) that cannot set custom request headers.
const localTokenHeader = "X-IT-Token"

// actionPayload is the JSON representation of a recorded Action.
type actionPayload struct {
	Index   int    `json:"index"`
	Summary string `json:"summary"`
}

// stagePayload is the JSON representation of a stage and its actions.
type stagePayload struct {
	Name    string          `json:"name"`
	Status  string          `json:"status"`
	Actions []actionPayload `json:"actions"`
}

// statePayload is the full snapshot the UI loads on start.
type statePayload struct {
	Stages []stagePayload `json:"stages"`
	Logs   []LogEntry     `json:"logs"`
}

// eventPayload is a single server-sent event.
type eventPayload struct {
	Type    string        `json:"type"`
	Log     *LogEntry     `json:"log,omitempty"`
	Stage   string        `json:"stage,omitempty"`
	Status  string        `json:"status,omitempty"`
	Message string        `json:"message,omitempty"`
	State   *statePayload `json:"state,omitempty"`
}

// UIServer exposes a Tester over HTTP so a web UI (Electron + React) can
// drive and inspect it.
type UIServer struct {
	tester   *Tester
	httpSrv  *http.Server
	listener net.Listener
	addr     string
	uiDir    string
	token    string

	mu      sync.Mutex
	runMu   sync.Mutex
	logs    []LogEntry
	status  map[string]string
	clients map[chan []byte]struct{}
}

var (
	serverRegistryMu sync.Mutex
	serverRegistry   = map[*UIServer]struct{}{}
	serverHandlers   sync.Once
)

// NewUIServer creates a server for the given tester. Call Start to listen.
func NewUIServer(t *Tester) *UIServer {
	status := make(map[string]string, len(t.Stages))
	for _, s := range t.Stages {
		status[s.Name] = "Not Run"
	}
	return &UIServer{
		tester:  t,
		status:  status,
		clients: make(map[chan []byte]struct{}),
		uiDir:   resolveUIDir(),
		token:   randomToken(),
	}
}

// Start binds the server to a free localhost port and begins serving.
func (s *UIServer) Start() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	s.listener = ln
	s.addr = ln.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/state", s.handleState)
	mux.HandleFunc("GET /api/events", s.handleEvents)
	mux.HandleFunc("POST /api/stage/run", s.handleRunStage)
	mux.HandleFunc("POST /api/action/run", s.handleRunAction)
	mux.HandleFunc("POST /api/run-all", s.handleRunAll)
	mux.HandleFunc("POST /api/discover", s.handleDiscover)
	mux.Handle("/", s.staticHandler())

	s.httpSrv = &http.Server{Handler: s.withCORS(s.requireAuth(mux))}
	registerServer(s)

	go func() {
		if err := s.httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("UI server stopped: %v", err)
		}
	}()
	return nil
}

// URL returns the base URL the server is listening on.
func (s *UIServer) URL() string {
	return "http://" + s.addr
}

// Token returns the per-process auth token required by the API. It is handed to
// the UI automatically when the server launches Electron or the browser.
func (s *UIServer) Token() string {
	return s.token
}

// URLWithToken returns the base URL with the auth token, for opening the UI
// manually (e.g. from another terminal).
func (s *UIServer) URLWithToken() string {
	return s.URL() + "/?token=" + url.QueryEscape(s.token)
}

// requireAuth rejects API requests that do not carry the server's token. Static
// assets and the health endpoint stay open so the UI can load and bootstrap.
func (s *UIServer) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}
		if subtle.ConstantTimeCompare([]byte(requestToken(r)), []byte(s.token)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestToken(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	if tok := r.Header.Get(localTokenHeader); tok != "" {
		return tok
	}
	return r.URL.Query().Get("token")
}

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("integration_tester: failed to generate auth token: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// Stop shuts the server down and releases its resources.
func (s *UIServer) Stop() {
	unregisterServer(s)
	s.mu.Lock()
	for ch := range s.clients {
		close(ch)
	}
	s.clients = make(map[chan []byte]struct{})
	s.mu.Unlock()

	if s.httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = s.httpSrv.Shutdown(ctx)
	}
}

func (s *UIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *UIServer) handleState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.snapshot())
}

func (s *UIServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := make(chan []byte, 128)
	s.mu.Lock()
	s.clients[ch] = struct{}{}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		if _, ok := s.clients[ch]; ok {
			delete(s.clients, ch)
			close(ch)
		}
		s.mu.Unlock()
	}()

	// Send the current snapshot immediately so a fresh client is in sync.
	initial := s.snapshot()
	if data, err := json.Marshal(eventPayload{Type: "state", State: &initial}); err == nil {
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case data, ok := <-ch:
			if !ok {
				return
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (s *UIServer) handleRunStage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name             string `json:"name"`
		RunPrerequisites bool   `json:"runPrerequisites"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if body.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	go func() {
		s.runMu.Lock()
		defer s.runMu.Unlock()
		if body.RunPrerequisites {
			for _, st := range s.tester.Stages {
				if st.Name == body.Name {
					break
				}
				if s.getStatus(st.Name) == "PASSED" {
					continue
				}
				if err := s.runSingleStage(st.Name); err != nil {
					return
				}
			}
		}
		_ = s.runSingleStage(body.Name)
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (s *UIServer) handleRunAction(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Stage string `json:"stage"`
		Index int    `json:"index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	actions := GetStageActions(body.Stage)
	if body.Index < 0 || body.Index >= len(actions) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "action not found"})
		return
	}
	action := actions[body.Index]

	go func() {
		s.runMu.Lock()
		defer s.runMu.Unlock()
		defer func() {
			if rec := recover(); rec != nil {
				var msg string
				if te, ok := rec.(TestError); ok {
					msg = te.Message
				} else {
					msg = fmt.Sprintf("%v", rec)
				}
				Log(LogTypeInfo, "Manual Run FAILED: "+action.Summary, msg)
				s.broadcast(eventPayload{Type: "error", Stage: body.Stage, Message: msg})
			}
		}()
		Log(LogTypeInfo, "Manual Run: "+action.Summary, "")
		action.Func()
		Log(LogTypeInfo, "Manual Run PASSED: "+action.Summary, "")
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (s *UIServer) handleRunAll(w http.ResponseWriter, r *http.Request) {
	go func() {
		s.runMu.Lock()
		defer s.runMu.Unlock()
		for _, st := range s.tester.Stages {
			if err := s.runSingleStage(st.Name); err != nil {
				// Continue running the rest, matching CLI behaviour, but
				// report the failure through the event stream.
				s.broadcast(eventPayload{Type: "error", Stage: st.Name, Message: err.Error()})
			}
		}
	}()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (s *UIServer) handleDiscover(w http.ResponseWriter, r *http.Request) {
	go func() {
		s.runMu.Lock()
		defer s.runMu.Unlock()
		s.tester.DryRunAll()
		s.broadcast(eventPayload{Type: "refresh"})
	}()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (s *UIServer) runSingleStage(name string) error {
	s.setStatus(name, "Running...")
	err := s.tester.RunStageByName(name)
	if err != nil {
		s.setStatus(name, "FAILED")
	} else {
		s.setStatus(name, "PASSED")
	}
	return err
}

func (s *UIServer) getStatus(name string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status[name]
}

func (s *UIServer) setStatus(name, status string) {
	s.mu.Lock()
	s.status[name] = status
	s.mu.Unlock()
	s.broadcast(eventPayload{Type: "stage", Stage: name, Status: status})
}

func (s *UIServer) snapshot() statePayload {
	stages := make([]stagePayload, 0, len(s.tester.Stages))
	for _, st := range s.tester.Stages {
		acts := GetStageActions(st.Name)
		payload := make([]actionPayload, len(acts))
		for i, a := range acts {
			payload[i] = actionPayload{Index: i, Summary: a.Summary}
		}
		stages = append(stages, stagePayload{
			Name:    st.Name,
			Status:  s.getStatus(st.Name),
			Actions: payload,
		})
	}

	s.mu.Lock()
	logs := make([]LogEntry, len(s.logs))
	copy(logs, s.logs)
	s.mu.Unlock()

	return statePayload{Stages: stages, Logs: logs}
}

func (s *UIServer) broadcast(evt eventPayload) {
	data, err := json.Marshal(evt)
	if err != nil {
		return
	}

	// Hold the lock while sending so a concurrent Stop/deferred close cannot
	// close a channel between selection and send. Sends are non-blocking.
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.clients {
		select {
		case ch <- data:
		default:
			// Slow client: drop the event rather than block the tester.
		}
	}
}

func (s *UIServer) onLog(entry LogEntry) {
	s.mu.Lock()
	s.logs = append(s.logs, entry)
	s.mu.Unlock()
	s.broadcast(eventPayload{Type: "log", Log: &entry})
}

func (s *UIServer) onActions() {
	s.broadcast(eventPayload{Type: "refresh"})
}

func (s *UIServer) staticHandler() http.Handler {
	dist := filepath.Join(s.uiDir, "dist")
	if fileExists(filepath.Join(dist, "index.html")) {
		return spaFileServer(dist)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `<!doctype html><html><body style="font-family:sans-serif;padding:2rem">
<h1>Integration Tester UI</h1>
<p>The React UI has not been built yet.</p>
<pre>cd %s &amp;&amp; npm install &amp;&amp; npm run build</pre>
<p>Then reload this page, or run the Electron app with <code>npm start</code>.</p>
</body></html>`, s.uiDir)
	})
}

func spaFileServer(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			if _, err := os.Stat(filepath.Join(dir, filepath.Clean(r.URL.Path))); err != nil {
				r.URL.Path = "/"
			}
		}
		fs.ServeHTTP(w, r)
	})
}

// withCORS allows the Electron renderer (file://) and the Vite dev server to
// call the API from a different origin, while rejecting arbitrary web origins.
// Combined with the auth token this blocks drive-by requests from other sites
// and DNS-rebinding attacks.
func (s *UIServer) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && allowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, "+localTokenHeader)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// allowedOrigin permits same-origin/no-origin requests, Electron's file://
// origin ("null"), and loopback origins used by the Vite dev server.
func allowedOrigin(origin string) bool {
	if origin == "null" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// RunGUI starts the desktop UI. It serves the API, then launches the Electron
// app if it can be found, falling back to the default browser otherwise.
// The call blocks until the UI exits.
func RunGUI(t *Tester) {
	RunElectron(t)
}

// RunElectron starts the API server and launches the Electron + React UI.
// If Electron is not installed it falls back to opening the browser, where the
// same UI is served from the built assets in ui/dist.
func RunElectron(t *Tester) {
	srv := NewUIServer(t)
	if err := srv.Start(); err != nil {
		log.Fatalf("failed to start UI server: %v", err)
	}
	defer srv.Stop()
	log.Printf("Integration Tester UI server listening at %s", srv.URLWithToken())

	bin, baseArgs := resolveElectron(srv.uiDir)
	if bin != "" {
		if err := launchElectron(srv.URL(), srv.uiDir, srv.Token(), bin, baseArgs); err != nil {
			log.Printf("Electron exited: %v", err)
		}
		return
	}

	log.Printf("Electron not found; opening the browser instead")
	openBrowser(srv.URLWithToken())
	waitForSignal()
}

// RunServer starts only the HTTP API + web UI and opens the browser. Useful for
// headless machines or when running the UI separately.
func RunServer(t *Tester) {
	srv := NewUIServer(t)
	if err := srv.Start(); err != nil {
		log.Fatalf("failed to start UI server: %v", err)
	}
	defer srv.Stop()
	log.Printf("Integration Tester UI server listening at %s", srv.URLWithToken())
	openBrowser(srv.URLWithToken())
	waitForSignal()
}

func registerServer(s *UIServer) {
	serverRegistryMu.Lock()
	serverRegistry[s] = struct{}{}
	serverRegistryMu.Unlock()

	serverHandlers.Do(func() {
		RegisterLogHandler(func(entry LogEntry) {
			serverRegistryMu.Lock()
			defer serverRegistryMu.Unlock()
			for srv := range serverRegistry {
				srv.onLog(entry)
			}
		})
		RegisterActionUpdateHandler(func() {
			serverRegistryMu.Lock()
			defer serverRegistryMu.Unlock()
			for srv := range serverRegistry {
				srv.onActions()
			}
		})
	})
}

func unregisterServer(s *UIServer) {
	serverRegistryMu.Lock()
	delete(serverRegistry, s)
	serverRegistryMu.Unlock()
}

func launchElectron(serverURL, uiDir, token, bin string, baseArgs []string) error {
	args := append(append([]string{}, baseArgs...), uiDir)
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(),
		"IT_SERVER_URL="+serverURL,
		"IT_TOKEN="+token,
		"IT_UI_DIST="+filepath.Join(uiDir, "dist", "index.html"),
	)
	log.Printf("Launching Electron: %s %s", bin, strings.Join(args, " "))
	return cmd.Run()
}

func resolveElectron(uiDir string) (string, []string) {
	if bin := os.Getenv("ELECTRON_BIN"); bin != "" {
		return bin, nil
	}
	local := filepath.Join(uiDir, "node_modules", ".bin", "electron")
	if runtime.GOOS == "windows" {
		local += ".cmd"
	}
	if isExecutable(local) {
		return local, nil
	}
	if p, err := exec.LookPath("electron"); err == nil {
		return p, nil
	}
	if _, err := exec.LookPath("npx"); err == nil {
		return "npx", []string{"--no-install", "electron"}
	}
	return "", nil
}

func resolveUIDir() string {
	if dir := os.Getenv("INTEGRATION_TESTER_UI"); dir != "" {
		return dir
	}
	if wd, err := os.Getwd(); err == nil {
		dir := wd
		for i := 0; i < 8; i++ {
			candidate := filepath.Join(dir, "ui")
			if fileExists(filepath.Join(candidate, "electron", "main.cjs")) {
				return candidate
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
		if wd != "" {
			return filepath.Join(wd, "ui")
		}
	}
	return "ui"
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("Could not open browser automatically: %v", err)
	}
}

func waitForSignal() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}
