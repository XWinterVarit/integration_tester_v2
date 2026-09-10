package dynamic_mock_server

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
	FuncIfRequestHeader          = "IfRequestHeader"
	FuncIfRequestHeaderSetCase   = "IfRequestHeaderSetCase"
	FuncIfRequestJsonBody        = "IfRequestJsonBody"
	FuncIfRequestJsonBodySetCase = "IfRequestJsonBodySetCase"
	FuncIfRequestXmlBody         = "IfRequestXmlBody"
	FuncIfRequestXmlBodySetCase  = "IfRequestXmlBodySetCase"
	FuncIfRequestPath            = "IfRequestPath"
	FuncIfRequestPathSetCase     = "IfRequestPathSetCase"
	FuncIfRequestQuery           = "IfRequestQuery"
	FuncIfRequestQuerySetCase    = "IfRequestQuerySetCase"
	FuncIfDynamicVariable        = "IfDynamicVariable"
	FuncIfDynamicVariableSetCase = "IfDynamicVariableSetCase"

	// JSON checks
	FuncIfRequestJsonArrayLength         = "IfRequestJsonArrayLength"
	FuncIfRequestJsonArrayLengthSetCase  = "IfRequestJsonArrayLengthSetCase"
	FuncIfRequestJsonObjectLength        = "IfRequestJsonObjectLength"
	FuncIfRequestJsonObjectLengthSetCase = "IfRequestJsonObjectLengthSetCase"
	FuncIfRequestJsonType                = "IfRequestJsonType"
	FuncIfRequestJsonTypeSetCase         = "IfRequestJsonTypeSetCase"

	FuncExtractRequestHeader   = "ExtractRequestHeader"
	FuncExtractRequestJsonBody = "ExtractRequestJsonBody"
	FuncExtractRequestXmlBody  = "ExtractRequestXmlBody"
	FuncExtractRequestPath     = "ExtractRequestPath"
	FuncExtractRequestQuery    = "ExtractRequestQuery"

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

// Conditions
const (
	ConditionEqual              = "Equal"
	ConditionNotEqual           = "NotEqual"
	ConditionContains           = "Contains"
	ConditionNotContains        = "NotContains"
	ConditionStartsWith         = "StartsWith"
	ConditionEndsWith           = "EndsWith"
	ConditionGreaterThan        = "GreaterThan"
	ConditionLessThan           = "LessThan"
	ConditionGreaterThanOrEqual = "GreaterThanOrEqual"
	ConditionLessThanOrEqual    = "LessThanOrEqual"
)
