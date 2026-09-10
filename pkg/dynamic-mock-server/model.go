package dynamic_mock_server

import "github.com/XWinterVarit/integrate_tester_v2/pkg/condition"

// ResponseFuncConfig represents the JSON structure for a response function configuration
type ResponseFuncConfig struct {
	Group string        `json:"group"`
	Func  string        `json:"func"`
	Args  []interface{} `json:"args"`
}

// RegisterRouteRequest represents the body for /registerRoute
type RegisterRouteRequest struct {
	Port         int                  `json:"port"`
	Method       string               `json:"method"`
	Path         string               `json:"path"`
	ResponseFunc []ResponseFuncConfig `json:"responseFunc"`
}

// Constants for Response Func Groups
const (
	GroupPrepareData     = "PrepareData"
	GroupGenerator       = "Generator"
	GroupDynamicVariable = "DynamicVariable"
	GroupSetupResponse   = "SetupResponse"
)

// Constants for Response Func Names
const (
	// PrepareData
	FuncIfRequestHeader           = "IfRequestHeader"
	FuncIfRequestHeaderSetCase    = "IfRequestHeaderSetCase"
	FuncIfRequestJsonBody         = "IfRequestJsonBody"
	FuncIfRequestJsonBodySetCase  = "IfRequestJsonBodySetCase"
	FuncIfRequestXmlBody          = "IfRequestXmlBody"
	FuncIfRequestXmlBodySetCase   = "IfRequestXmlBodySetCase"
	FuncIfRequestPath             = "IfRequestPath"
	FuncIfRequestPathSetCase      = "IfRequestPathSetCase"
	FuncIfRequestPathParam        = "IfRequestPathParam"
	FuncIfRequestPathParamSetCase = "IfRequestPathParamSetCase"
	FuncIfRequestQuery            = "IfRequestQuery"
	FuncIfRequestQuerySetCase     = "IfRequestQuerySetCase"
	FuncIfDynamicVariable         = "IfDynamicVariable"
	FuncIfDynamicVariableSetCase  = "IfDynamicVariableSetCase"

	// JSON checks
	FuncIfRequestJsonArrayLength         = "IfRequestJsonArrayLength"
	FuncIfRequestJsonArrayLengthSetCase  = "IfRequestJsonArrayLengthSetCase"
	FuncIfRequestJsonObjectLength        = "IfRequestJsonObjectLength"
	FuncIfRequestJsonObjectLengthSetCase = "IfRequestJsonObjectLengthSetCase"
	FuncIfRequestJsonType                = "IfRequestJsonType"
	FuncIfRequestJsonTypeSetCase         = "IfRequestJsonTypeSetCase"

	FuncExtractRequestHeader    = "ExtractRequestHeader"
	FuncExtractRequestJsonBody  = "ExtractRequestJsonBody"
	FuncExtractRequestXmlBody   = "ExtractRequestXmlBody"
	FuncExtractRequestPath      = "ExtractRequestPath"
	FuncExtractRequestPathParam = "ExtractRequestPathParam"
	FuncExtractRequestQuery     = "ExtractRequestQuery"

	// Generator
	FuncGenerateRandomString       = "GenerateRandomString"
	FuncGenerateRandomInt          = "GenerateRandomInt"
	FuncGenerateRandomIntFixLength = "GenerateRandomIntFixLength"
	FuncGenerateRandomDecimal      = "GenerateRandomDecimal"
	FuncHashedString               = "HashedString"

	// DynamicVariable
	FuncConvertToString     = "ConvertToString"
	FuncConvertToInt        = "ConvertToInt"
	FuncDynamicVarSubstring = "DynamicVarSubstring"
	FuncDynamicVarJoin      = "DynamicVarJoin"
	FuncDelete              = "Delete"

	// SetupResponse
	FuncSetJsonBody           = "SetJsonBody"
	FuncSetXmlBody            = "SetXmlBody"
	FuncSetStatusCode         = "SetStatusCode"
	FuncSetWait               = "SetWait"
	FuncSetRandomWait         = "SetRandomWait"
	FuncSetMethod             = "SetMethod"
	FuncSetHeader             = "SetHeader"
	FuncCopyHeaderFromRequest = "CopyHeaderFromRequest"
)

// Conditions are re-exported from pkg/condition so existing callers can keep
// using dynamic_mock_server.ConditionEqual, etc. Matching uses the same strict
// engine as pkg/v1 assertions.
const (
	ConditionEqual              = condition.Equal
	ConditionNotEqual           = condition.NotEqual
	ConditionContains           = condition.Contains
	ConditionNotContains        = condition.NotContains
	ConditionStartsWith         = condition.StartsWith
	ConditionEndsWith           = condition.EndsWith
	ConditionGreaterThan        = condition.GreaterThan
	ConditionLessThan           = condition.LessThan
	ConditionGreaterThanOrEqual = condition.GreaterThanOrEqual
	ConditionLessThanOrEqual    = condition.LessThanOrEqual
	ConditionMatches            = condition.Matches
	ConditionIn                 = condition.In
	ConditionNotIn              = condition.NotIn
	ConditionEmpty              = condition.Empty
	ConditionNotEmpty           = condition.NotEmpty
)
