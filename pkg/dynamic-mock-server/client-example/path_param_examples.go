package main

import (
	"fmt"

	dms "github.com/XWinterVarit/integrate_tester_v2/pkg/dynamic-mock-server"
)

// runPathParamExamples shows how to register a route with named path
// parameters (e.g. "/a/b/{ee}/c/{dd}") and use the captured values in
// conditions, cases, and response templates.
func runPathParamExamples(client *dms.Client) {
	fmt.Println("9. Path Parameters: POST /a/b/{ee}/c/{dd}")

	err := client.RegisterRoute(MockPort, "POST", "/a/b/{ee}/c/{dd}", []dms.ResponseFuncConfig{
		// Captured path parameters are exposed automatically as template
		// variables ({{.ee}}, {{.dd}}). You can also copy one explicitly.
		dms.ExtractRequestPathParam("ee", "EE_COPY"),

		// Conditional variable based on a captured path parameter.
		dms.IfRequestPathParam("ee", dms.ConditionEqual, "hello", "EE_OK", "yes"),
		dms.IfRequestPathParam("ee", dms.ConditionNotEqual, "hello", "EE_OK", "no"),

		// Switch the active case based on a captured path parameter.
		dms.IfRequestPathParamSetCase("dd", dms.ConditionEqual, "admin", "AdminCase"),

		// Default response.
		dms.SetStatusCode("", 200),
		dms.SetHeader("", "Content-Type", "application/json"),
		dms.SetJsonBody("", `{"route":"default","ee":"{{.ee}}","dd":"{{.dd}}","ee_ok":"{{.EE_OK}}"}`),

		// Admin response.
		dms.SetStatusCode("AdminCase", 200),
		dms.SetJsonBody("AdminCase", `{"route":"admin","ee":"{{.ee}}","ee_copy":"{{.EE_COPY}}"}`),
	})
	if err != nil {
		fmt.Printf("Error registering: %v\n", err)
		return
	}

	fmt.Println("-> POST /a/b/hello/c/user (expect default, ee_ok=yes)")
	printRequest("POST", fmt.Sprintf("http://localhost:%d/a/b/hello/c/user", MockPort), nil)

	fmt.Println("-> POST /a/b/hello/c/admin (expect admin case)")
	printRequest("POST", fmt.Sprintf("http://localhost:%d/a/b/hello/c/admin", MockPort), nil)

	fmt.Println("-> POST /a/b/world/c/user (expect default, ee_ok=no)")
	printRequest("POST", fmt.Sprintf("http://localhost:%d/a/b/world/c/user", MockPort), nil)

	fmt.Println("-> POST /a/b/hello/x/user (expect 404, literal segment mismatch)")
	printRequest("POST", fmt.Sprintf("http://localhost:%d/a/b/hello/x/user", MockPort), nil)
}
