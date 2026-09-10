package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	dms "github.com/XWinterVarit/integrate_tester_v2/pkg/dynamic-mock-server"
)

func runConditionalExamples(client *dms.Client) {
	// Example 1: Header Condition
	fmt.Println("1. Conditional Header: Authorization")

	err := client.RegisterRoute(MockPort, "POST", "/auth-check", []dms.ResponseFuncConfig{
		// Default variable state via IfRequestPath (matches current path)
		dms.IfRequestPath(dms.ConditionEqual, "/auth-check", "USER_NAME", "Guest"),

		dms.IfRequestHeader("Authorization", dms.ConditionEqual, "Bearer secret", "USER_NAME", "AdminUser"),
		dms.SetJsonBody("", `{"user": "{{.USER_NAME}}", "info": "If empty user, auth failed"}`),
		dms.SetStatusCode("", 200),
	})
	if err != nil {
		fmt.Printf("Error registering: %v\n", err)
		return
	}

	// 1.1 Request without header
	printRequest("POST", fmt.Sprintf("http://localhost:%d/auth-check", MockPort), nil)

	// 1.2 Request with header
	req, _ := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/auth-check", MockPort), nil)
	req.Header.Set("Authorization", "Bearer secret")
	printRequestWithReq(req)

	// Example 2: Body Condition
	fmt.Println("2. Conditional Body: type=premium")
	err = client.RegisterRoute(MockPort, "POST", "/upgrade", []dms.ResponseFuncConfig{
		// Default message
		dms.IfRequestPath(dms.ConditionEqual, "/upgrade", "MESSAGE", "Standard User"),

		dms.IfRequestJsonBody("type", dms.ConditionEqual, "premium", "MESSAGE", "Welcome Premium User!"),
		dms.SetJsonBody("", `{"msg": "{{.MESSAGE}}"}`),
		dms.SetStatusCode("", 200),
	})
	if err != nil {
		fmt.Printf("Error registering: %v\n", err)
		return
	}

	// 2.1 Request with premium
	printRequest("POST", fmt.Sprintf("http://localhost:%d/upgrade", MockPort),
		bytes.NewBufferString(`{"type": "premium"}`))

	// 2.2 Request with basic
	printRequest("POST", fmt.Sprintf("http://localhost:%d/upgrade", MockPort),
		bytes.NewBufferString(`{"type": "basic"}`))
}

func printRequestWithReq(req *http.Request) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response [%d]: %s\n", resp.StatusCode, strings.TrimSpace(string(b)))
}
