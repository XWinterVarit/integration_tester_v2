package v1

import (
	"io"
	"net/http"
)

// Request wraps http.Request to simplify usage.
type Request struct {
	Method string
	URL    string
	Header http.Header
	Body   string
}

// Response wraps http.Response (or mock response definition).
type Response struct {
	StatusCode int
	Body       string
	Header     map[string]string
}

// NewRequestWrapper creates a wrapper from http.Request.
func NewRequestWrapper(r *http.Request) Request {
	bodyBytes, _ := io.ReadAll(r.Body)
	// We don't close here because we might not own it, but actually we do read it all.
	// Standard practice: Server handlers don't need to close body.
	return Request{
		Method: r.Method,
		URL:    r.URL.String(),
		Header: r.Header,
		Body:   string(bodyBytes),
	}
}

// NewResponse Helper to create a response for mocks.
func NewResponse(code int, body string) Response {
	return Response{
		StatusCode: code,
		Body:       body,
		Header:     make(map[string]string),
	}
}
