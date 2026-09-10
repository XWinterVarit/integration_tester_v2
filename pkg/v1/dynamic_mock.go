package v1

import (
	"fmt"

	dm "github.com/XWinterVarit/integrate_tester_v2/pkg/dynamic-mock-server"
)

// DynamicMockClient is a wrapper around the dynamic mock server client.
type DynamicMockClient struct {
	*dm.Client
}

// ResponseFuncConfig aliases the configuration struct from dynamic-mock-server.
type ResponseFuncConfig = dm.ResponseFuncConfig

// Constants for Conditions
const (
	ConditionEqual              = dm.ConditionEqual
	ConditionNotEqual           = dm.ConditionNotEqual
	ConditionContains           = dm.ConditionContains
	ConditionNotContains        = dm.ConditionNotContains
	ConditionStartsWith         = dm.ConditionStartsWith
	ConditionEndsWith           = dm.ConditionEndsWith
	ConditionGreaterThan        = dm.ConditionGreaterThan
	ConditionLessThan           = dm.ConditionLessThan
	ConditionGreaterThanOrEqual = dm.ConditionGreaterThanOrEqual
	ConditionLessThanOrEqual    = dm.ConditionLessThanOrEqual
)

// NewDynamicMockClient creates a new client for an existing dynamic mock server.
// controlURL is the base URL of the mock controller (e.g., "http://localhost:8888").
func NewDynamicMockClient(controlURL string) *DynamicMockClient {
	RecordAction("Mock NewClient", func() { NewDynamicMockClient(controlURL) })
	if IsDryRun() {
		return &DynamicMockClient{}
	}
	client := dm.NewClient(controlURL)
	return &DynamicMockClient{Client: client}
}

// RegisterRoute wraps the dynamic mock client, skipping external calls in dry-run mode.
func (c *DynamicMockClient) RegisterRoute(port int, method string, path string, responseFuncs []ResponseFuncConfig) error {
	RecordAction(fmt.Sprintf("Mock RegisterRoute: %s %s", method, path), func() { c.RegisterRoute(port, method, path, responseFuncs) })
	if IsDryRun() {
		return nil
	}
	if c == nil || c.Client == nil {
		return fmt.Errorf("mock client is not initialized")
	}
	return c.Client.RegisterRoute(port, method, path, responseFuncs)
}

// ResetPort resets routes for a port. No-op in dry-run.
func (c *DynamicMockClient) ResetPort(port int) error {
	RecordAction(fmt.Sprintf("Mock ResetPort: %d", port), func() { c.ResetPort(port) })
	if IsDryRun() {
		return nil
	}
	if c == nil || c.Client == nil {
		return fmt.Errorf("mock client is not initialized")
	}
	return c.Client.ResetPort(port)
}

// ResetAll resets all routes. No-op in dry-run.
func (c *DynamicMockClient) ResetAll() error {
	RecordAction("Mock ResetAll", func() { c.ResetAll() })
	if IsDryRun() {
		return nil
	}
	if c == nil || c.Client == nil {
		return fmt.Errorf("mock client is not initialized")
	}
	return c.Client.ResetAll()
}

// Generator and Condition Functions Aliases

var (
	IfRequestHeader          = dm.IfRequestHeader
	IfRequestJsonBody        = dm.IfRequestJsonBody
	IfRequestPath            = dm.IfRequestPath
	IfRequestQuery           = dm.IfRequestQuery
	IfRequestHeaderSetCase   = dm.IfRequestHeaderSetCase
	IfRequestJsonBodySetCase = dm.IfRequestJsonBodySetCase
	IfRequestXmlBody         = dm.IfRequestXmlBody
	IfRequestXmlBodySetCase  = dm.IfRequestXmlBodySetCase
	IfRequestPathSetCase     = dm.IfRequestPathSetCase
	IfRequestQuerySetCase    = dm.IfRequestQuerySetCase

	IfDynamicVariable        = dm.IfDynamicVariable
	IfDynamicVariableSetCase = dm.IfDynamicVariableSetCase

	IfRequestJsonArrayLength         = dm.IfRequestJsonArrayLength
	IfRequestJsonArrayLengthSetCase  = dm.IfRequestJsonArrayLengthSetCase
	IfRequestJsonObjectLength        = dm.IfRequestJsonObjectLength
	IfRequestJsonObjectLengthSetCase = dm.IfRequestJsonObjectLengthSetCase
	IfRequestJsonType                = dm.IfRequestJsonType
	IfRequestJsonTypeSetCase         = dm.IfRequestJsonTypeSetCase

	ExtractRequestHeader   = dm.ExtractRequestHeader
	ExtractRequestJsonBody = dm.ExtractRequestJsonBody
	ExtractRequestXmlBody  = dm.ExtractRequestXmlBody
	ExtractRequestPath     = dm.ExtractRequestPath
	ExtractRequestQuery    = dm.ExtractRequestQuery

	GenerateRandomString       = dm.GenerateRandomString
	GenerateRandomInt          = dm.GenerateRandomInt
	GenerateRandomIntFixLength = dm.GenerateRandomIntFixLength
	GenerateRandomDecimal      = dm.GenerateRandomDecimal
	HashedString               = dm.HashedString

	ConvertToString     = dm.ConvertToString
	ConvertToInt        = dm.ConvertToInt
	DynamicVarSubstring = dm.DynamicVarSubstring
	DynamicVarJoin      = dm.DynamicVarJoin
	Delete              = dm.Delete

	SetJsonBody           = dm.SetJsonBody
	SetXmlBody            = dm.SetXmlBody
	SetStatusCode         = dm.SetStatusCode
	SetWait               = dm.SetWait
	SetRandomWait         = dm.SetRandomWait
	SetMethod             = dm.SetMethod
	SetHeader             = dm.SetHeader
	CopyHeaderFromRequest = dm.CopyHeaderFromRequest
)
