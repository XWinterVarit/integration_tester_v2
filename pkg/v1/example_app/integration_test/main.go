package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	v1 "github.com/XWinterVarit/integrate_tester_v2/pkg/v1"

	_ "github.com/sijms/go-ora/v2"
)

// waitForServer polls url until the server responds (any HTTP status) or the
// timeout elapses. Used to wait for the example app to finish booting.
func waitForServer(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("server at %s not ready after %s", url, timeout)
}

func main() {
	mockUrl := flag.String("mock-url", "http://localhost:9001", "Mock Server Control URL")
	// Registers -mode (gui | cli | cli-command | server); see v1.Run.
	v1.RegisterModeFlag("")
	flag.Parse()

	// Resolve the example app source relative to this file so the build works
	// regardless of the process working directory (e.g. repo root in GoLand).
	_, thisFile, _, _ := runtime.Caller(0)
	exampleAppMain := filepath.Join(filepath.Dir(filepath.Dir(thisFile)), "main.go")

	t := v1.NewTester()
	// Mock Service Port (where the actual mocked service will listen)
	mockPort := 9002

	var app *v1.AppServer
	// DB client for the test runner to manipulate DB
	var db *v1.DBClient

	t.Stage("Setup", func() {
		// Build the example app binary to ensure the latest routes (e.g., /update-json) are available
		workDir, err := os.Getwd()
		v1.AssertNoError(err)
		appPath := filepath.Join(workDir, "example_app_bin")

		buildCmd := exec.Command("go", "build", "-o", appPath, exampleAppMain)
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		if err := buildCmd.Run(); err != nil {
			v1.Fail("Failed to build example app: %v", err)
		}

		// 1. Connect to Oracle
		dsn := "oracle://LEARN1:Welcome@localhost:1521/XE"
		db = v1.Connect("oracle", dsn)

		// Prepare Table (Oracle syntax)
		// Note: Oracle 11g/12c differences exist. We use simple NUMBER for ID.
		// Since ReplaceData provides ID, we don't need AUTO_INCREMENT/IDENTITY for this test.
		// So we just need PRIMARY KEY constraint.
		db.SetupTable("users", true, []v1.Field{
			{"id", "NUMBER PRIMARY KEY"},
			{"name", "VARCHAR2(100)"},
			{"status", "VARCHAR2(50)"},
		}, nil)

		// 2. Run App
		app = v1.RunAppServer(appPath, "-driver", "oracle", "-dsn", dsn, "-mock-service", fmt.Sprintf("http://localhost:%d", mockPort))

		// Wait for the app to finish booting (it connects to Oracle and binds :8080
		// asynchronously after RunAppServer returns).
		if !v1.IsDryRun() {
			if err := waitForServer("http://localhost:8080/read?id=1", 15*time.Second); err != nil {
				v1.Fail("App server did not become ready: %v", err)
			}
		}

		// 3. Insert Initial Data using the app /insert endpoint (new key/value struct style)
		resp := v1.SendRESTRequest("http://localhost:8080/insert?id=1&name=alice&status=active",
			v1.WithMethod(http.MethodPost),
		)
		v1.ExpectStatusCode(resp, http.StatusCreated)
		v1.ExpectJsonBody(resp, `{"result": "inserted"}`)
	})

	t.Stage("Success Case", func() {
		// "request test for success case"
		// 1. Update via App
		resp := v1.SendRequest("http://localhost:8080/update?id=1&status=updated")
		v1.ExpectStatusCode(resp, 200)

		// 2. Verify DB (Manipulate/Check record in DB)
		// Fetch uses QueryData which we updated to handle placeholders for Oracle
		result := db.Fetch("SELECT status FROM users WHERE id = ?", 1)
		result.ExpectCount(1)
		// Verify using simplified Expect method
		result.GetRow(0).Expect("status", "updated")

		// 3. Read via App (Should succeed now)
		resp = v1.SendRequest("http://localhost:8080/read?id=1")
		v1.ExpectJsonBody(resp, `{"id": "1", "status": "updated"}`)
	})

	t.Stage("Complex Request (POST JSON)", func() {
		requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())
		newStatus := "json-updated"

		resp := v1.SendRESTRequest("http://localhost:8080/update-json",
			v1.WithMethod(http.MethodPost),
			v1.WithHeader("X-Request-ID", requestID),
			v1.WithHeaders(map[string]string{"X-Trace": "integration-example"}),
			v1.WithJSONBody(map[string]interface{}{
				"id":     "1",
				"status": newStatus,
				"meta": map[string]interface{}{
					"requested_at": time.Now().Format(time.RFC3339),
				},
			}),
		)

		v1.ExpectStatusCode(resp, 200)
		v1.ExpectHeader(resp, "Content-Type", "application/json")
		v1.ExpectJsonBodyField(resp, "id", "1")
		v1.ExpectJsonBodyField(resp, "status", newStatus)
		v1.ExpectJsonBodyField(resp, "request_id", requestID)

		// Verify DB updated from POST JSON call
		result := db.Fetch("SELECT status FROM users WHERE id = ?", 1)
		result.ExpectCount(1)
		result.GetRow(0).Expect("status", newStatus)
	})

	t.Stage("Fail Case", func() {
		// "do another request for expected fail"
		// 1. Manipulate record in DB (Set to 'bad')
		// Update uses placeholders which we updated to handle Oracle
		db.Update("users", map[string]interface{}{"status": "bad"}, "id = ?", 1)

		// 2. Request Expected Fail
		resp := v1.SendRequest("http://localhost:8080/read?id=1")
		v1.ExpectStatusCode(resp, 500)
	})

	t.Stage("Redis Hash Operations", func() {
		redisServerAddr := "http://localhost:9100"
		if addr := os.Getenv("REDIS_SERVER_ADDR"); addr != "" {
			redisServerAddr = addr
		}
		redisAccessKey := "welcome"
		if key := os.Getenv("REDIS_ACCESS_KEY"); key != "" {
			redisAccessKey = key
		}
		redis := v1.ConnectRedis(redisServerAddr, redisAccessKey)

		// HSet: store user profile fields in a hash
		redis.HSet("profile:1", "name", "Alice")
		redis.HSet("profile:1", "email", "alice@example.com")

		// HGet: verify stored fields
		name := redis.HGet("profile:1", "name")
		if name != "Alice" {
			v1.Fail("Expected profile:1 name=Alice, got %s", name)
		}
		email := redis.HGet("profile:1", "email")
		if email != "alice@example.com" {
			v1.Fail("Expected profile:1 email=alice@example.com, got %s", email)
		}

		// HIncrement: track page view counter in a hash
		redis.HSet("stats:page", "views", "100")
		newViews := redis.HIncrement("stats:page", "views", 10)
		if newViews != 110 {
			v1.Fail("Expected stats:page views=110, got %d", newViews)
		}

		// Cleanup
		redis.Del("profile:1", "stats:page")
	})

	t.Stage("Complex Request (POST XML)", func() {
		resp := v1.SendRESTRequest("http://localhost:8080/update-xml",
			v1.WithMethod(http.MethodPost),
			v1.WithXMLBody(struct {
				XMLName struct{} `xml:"request"`
				ID      string   `xml:"id"`
				Status  string   `xml:"status"`
			}{
				ID:     "1",
				Status: "xml-updated",
			}),
		)

		v1.ExpectStatusCode(resp, 200)
		v1.ExpectHeader(resp, "Content-Type", "application/xml")
		v1.ExpectXmlBodyField(resp, "response.id", "1")
		v1.ExpectXmlBodyField(resp, "response.status", "xml-updated")

		// Pretty-print the XML response for readability
		fmt.Println("XML Response (pretty):")
		fmt.Println(v1.PrettyXml(resp.Body))

		// Verify DB updated from POST XML call
		result := db.Fetch("SELECT status FROM users WHERE id = ?", 1)
		result.ExpectCount(1)
		result.GetRow(0).Expect("status", "xml-updated")
	})

	t.Stage("Mock Server Example", func() {
		// 1. Connect to the Mock Server
		// The mock server must be running separately (e.g., via docker or separate process)
		client := v1.NewDynamicMockClient(*mockUrl)

		// 2. Register a Route with Complex Logic
		// We use mockPort defined in main
		err := client.RegisterRoute(mockPort, "GET", "/mock-test", []v1.ResponseFuncConfig{
			// Generator
			v1.GenerateRandomInt(10, 50, "DISCOUNT"),

			// Conditions
			v1.IfRequestQuerySetCase("user_type", v1.ConditionEqual, "vip", "VIP"),

			// Default Response
			v1.SetStatusCode("", 200),
			v1.SetJsonBody("", `{"message": "Hello User", "discount": 0}`),
			v1.SetHeader("", "Content-Type", "application/json"),

			// VIP Response
			v1.SetStatusCode("VIP", 200),
			v1.SetJsonBody("VIP", `{"message": "Hello VIP", "discount": {{.DISCOUNT}}}`),
		})
		v1.AssertNoError(err)

		// 3. Verify Default Case
		resp := v1.SendRequest("http://localhost:8080/call-mock")
		v1.ExpectStatusCode(resp, 200)
		v1.ExpectJsonBody(resp, `{"message": "Hello User", "discount": 0}`)

		// 4. Verify VIP Case
		respVip := v1.SendRequest("http://localhost:8080/call-mock?user_type=vip")
		v1.ExpectStatusCode(respVip, 200)
		// Verify dynamic content
		if !strings.Contains(respVip.Body, "Hello VIP") {
			v1.Fail("Expected VIP message, got: %s", respVip.Body)
		}
	})

	t.Stage("Mock Server XML Example", func() {
		// 1. Connect to the Mock Server
		client := v1.NewDynamicMockClient(*mockUrl)

		// 2. Register an XML Route with Case Routing
		err := client.RegisterRoute(mockPort, "POST", "/mock-xml", []v1.ResponseFuncConfig{
			// Extract a field from the XML request body
			v1.ExtractRequestXmlBody("request.user.name", "USER_NAME"),

			// Route based on XML body content
			v1.IfRequestXmlBodySetCase("request.user.role", v1.ConditionEqual, "admin", "AdminCase"),

			// Default Response (XML)
			v1.SetStatusCode("", 200),
			v1.SetHeader("", "Content-Type", "application/xml"),
			v1.SetXmlBody("", `<response><message>Hello {{.USER_NAME}}</message><access>basic</access></response>`),

			// Admin Response (XML)
			v1.SetStatusCode("AdminCase", 200),
			v1.SetHeader("AdminCase", "Content-Type", "application/xml"),
			v1.SetXmlBody("AdminCase", `<response><message>Welcome Admin {{.USER_NAME}}</message><access>full</access></response>`),
		})
		v1.AssertNoError(err)

		// 3. Verify Default Case (non-admin)
		resp := v1.SendRESTRequest(fmt.Sprintf("http://localhost:%d/mock-xml", mockPort),
			v1.WithMethod(http.MethodPost),
			v1.WithXMLBody(struct {
				XMLName struct{} `xml:"request"`
				User    struct {
					Name string `xml:"name"`
					Role string `xml:"role"`
				} `xml:"user"`
			}{
				User: struct {
					Name string `xml:"name"`
					Role string `xml:"role"`
				}{Name: "Alice", Role: "viewer"},
			}),
		)
		v1.ExpectStatusCode(resp, 200)
		v1.ExpectXmlBodyField(resp, "response.access", "basic")
		v1.ExpectXmlBodyField(resp, "response.message", "Hello Alice")

		// 4. Verify Admin Case
		respAdmin := v1.SendRESTRequest(fmt.Sprintf("http://localhost:%d/mock-xml", mockPort),
			v1.WithMethod(http.MethodPost),
			v1.WithXMLBody(struct {
				XMLName struct{} `xml:"request"`
				User    struct {
					Name string `xml:"name"`
					Role string `xml:"role"`
				} `xml:"user"`
			}{
				User: struct {
					Name string `xml:"name"`
					Role string `xml:"role"`
				}{Name: "Bob", Role: "admin"},
			}),
		)
		v1.ExpectStatusCode(respAdmin, 200)
		v1.ExpectXmlBodyField(respAdmin, "response.access", "full")
		v1.ExpectXmlBodyField(respAdmin, "response.message", "Welcome Admin Bob")

		// Pretty-print the admin XML response
		fmt.Println("Mock XML Admin Response (pretty):")
		fmt.Println(v1.PrettyXml(respAdmin.Body))
	})

	t.Stage("Mock Server Path Params Example", func() {
		// Route with named path parameters: /a/b/{ee}/c/{dd}
		client := v1.NewDynamicMockClient(*mockUrl)

		err := client.RegisterRoute(mockPort, "POST", "/a/b/{ee}/c/{dd}", []v1.ResponseFuncConfig{
			// Captured path params are auto-injected into templates ({{.ee}}, {{.dd}}).
			v1.ExtractRequestPathParam("ee", "EE_COPY"),

			// Condition on a captured path param (always set a value to avoid
			// "<no value>" in templates).
			v1.IfRequestPathParam("ee", v1.ConditionEqual, "hello", "EE_OK", "yes"),
			v1.IfRequestPathParam("ee", v1.ConditionNotEqual, "hello", "EE_OK", "no"),

			// Case routing based on a captured path param.
			v1.IfRequestPathParamSetCase("dd", v1.ConditionEqual, "admin", "AdminCase"),

			// Default response.
			v1.SetStatusCode("", 200),
			v1.SetHeader("", "Content-Type", "application/json"),
			v1.SetJsonBody("", `{"route":"default","ee":"{{.ee}}","dd":"{{.dd}}","ee_ok":"{{.EE_OK}}"}`),

			// Admin response.
			v1.SetStatusCode("AdminCase", 200),
			v1.SetJsonBody("AdminCase", `{"route":"admin","ee":"{{.ee}}","ee_copy":"{{.EE_COPY}}"}`),
		})
		v1.AssertNoError(err)

		// Default case: /a/b/hello/c/user
		resp := v1.SendRESTRequest(fmt.Sprintf("http://localhost:%d/a/b/hello/c/user", mockPort),
			v1.WithMethod(http.MethodPost),
		)
		v1.ExpectStatusCode(resp, 200)
		v1.ExpectJsonBodyField(resp, "route", "default")
		v1.ExpectJsonBodyField(resp, "ee", "hello")
		v1.ExpectJsonBodyField(resp, "dd", "user")
		v1.ExpectJsonBodyField(resp, "ee_ok", "yes")

		// Admin case: /a/b/hello/c/admin
		respAdmin := v1.SendRESTRequest(fmt.Sprintf("http://localhost:%d/a/b/hello/c/admin", mockPort),
			v1.WithMethod(http.MethodPost),
		)
		v1.ExpectStatusCode(respAdmin, 200)
		v1.ExpectJsonBodyField(respAdmin, "route", "admin")
		v1.ExpectJsonBodyField(respAdmin, "ee", "hello")
		v1.ExpectJsonBodyField(respAdmin, "ee_copy", "hello")

		// Non-matching literal segment should 404.
		respMiss := v1.SendRESTRequest(fmt.Sprintf("http://localhost:%d/a/b/hello/x/user", mockPort),
			v1.WithMethod(http.MethodPost),
		)
		v1.ExpectStatusCode(respMiss, 404)
	})

	// Cleanup must run last: it stops the example app that earlier stages use.
	t.Stage("Cleanup", func() {
		if app != nil {
			app.Stop()
		}
	})

	// Mode is selected at runtime: -mode gui|cli|cli-command|server,
	// or INTEGRATION_TESTER_MODE / IT_MODE. Defaults to gui.
	v1.Run(t)
}
