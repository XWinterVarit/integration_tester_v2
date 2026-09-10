package v1

import (
	"fmt"

	dp "github.com/XWinterVarit/integrate_tester_v2/pkg/dynamic-proxy-server"
)

// DynamicProxyClient is a wrapper around the dynamic proxy server client.
type DynamicProxyClient struct {
	*dp.Client
}

// ProxyStatus aliases the status struct from dynamic-proxy-server.
type ProxyStatus = dp.ProxyStatus

// NewDynamicProxyClient creates a new client for an existing dynamic proxy server.
// controlURL is the base URL of the proxy controller (e.g., "http://localhost:9002").
func NewDynamicProxyClient(controlURL string) *DynamicProxyClient {
	RecordAction("Proxy NewClient", func() { NewDynamicProxyClient(controlURL) })
	if IsDryRun() {
		return &DynamicProxyClient{}
	}
	return &DynamicProxyClient{Client: dp.NewClient(controlURL)}
}

// RegisterProxy registers a TCP proxy. No-op in dry-run.
// name: unique proxy identifier
// listenAddr: address the proxy listens on, e.g. "localhost:16379"
// upstreamAddr: address of the real service, e.g. "localhost:6379"
func (c *DynamicProxyClient) RegisterProxy(name, listenAddr, upstreamAddr string) error {
	RecordAction(fmt.Sprintf("Proxy RegisterProxy: %s (%s -> %s)", name, listenAddr, upstreamAddr), func() {
		c.RegisterProxy(name, listenAddr, upstreamAddr)
	})
	if IsDryRun() {
		return nil
	}
	if c == nil || c.Client == nil {
		return fmt.Errorf("proxy client is not initialized")
	}
	return c.Client.RegisterProxy(name, listenAddr, upstreamAddr)
}

// EnableProxy re-enables a disabled proxy. No-op in dry-run.
func (c *DynamicProxyClient) EnableProxy(name string) error {
	RecordAction(fmt.Sprintf("Proxy EnableProxy: %s", name), func() { c.EnableProxy(name) })
	if IsDryRun() {
		return nil
	}
	if c == nil || c.Client == nil {
		return fmt.Errorf("proxy client is not initialized")
	}
	return c.Client.EnableProxy(name)
}

// DisableProxy disables a proxy so new connections are dropped. No-op in dry-run.
func (c *DynamicProxyClient) DisableProxy(name string) error {
	RecordAction(fmt.Sprintf("Proxy DisableProxy: %s", name), func() { c.DisableProxy(name) })
	if IsDryRun() {
		return nil
	}
	if c == nil || c.Client == nil {
		return fmt.Errorf("proxy client is not initialized")
	}
	return c.Client.DisableProxy(name)
}

// SetTimeout disables a proxy for timeoutMs milliseconds then re-enables it. No-op in dry-run.
func (c *DynamicProxyClient) SetTimeout(name string, timeoutMs int) error {
	RecordAction(fmt.Sprintf("Proxy SetTimeout: %s for %dms", name, timeoutMs), func() { c.SetTimeout(name, timeoutMs) })
	if IsDryRun() {
		return nil
	}
	if c == nil || c.Client == nil {
		return fmt.Errorf("proxy client is not initialized")
	}
	return c.Client.SetTimeout(name, timeoutMs)
}

// RemoveProxy stops and removes a proxy. No-op in dry-run.
func (c *DynamicProxyClient) RemoveProxy(name string) error {
	RecordAction(fmt.Sprintf("Proxy RemoveProxy: %s", name), func() { c.RemoveProxy(name) })
	if IsDryRun() {
		return nil
	}
	if c == nil || c.Client == nil {
		return fmt.Errorf("proxy client is not initialized")
	}
	return c.Client.RemoveProxy(name)
}

// ResetAll stops and removes all proxies. No-op in dry-run.
func (c *DynamicProxyClient) ResetAll() error {
	RecordAction("Proxy ResetAll", func() { c.ResetAll() })
	if IsDryRun() {
		return nil
	}
	if c == nil || c.Client == nil {
		return fmt.Errorf("proxy client is not initialized")
	}
	return c.Client.ResetAll()
}

// Status returns the current status of all proxies. Returns empty slice in dry-run.
func (c *DynamicProxyClient) Status() ([]ProxyStatus, error) {
	RecordAction("Proxy Status", func() { c.Status() })
	if IsDryRun() {
		return []ProxyStatus{}, nil
	}
	if c == nil || c.Client == nil {
		return nil, fmt.Errorf("proxy client is not initialized")
	}
	return c.Client.Status()
}
