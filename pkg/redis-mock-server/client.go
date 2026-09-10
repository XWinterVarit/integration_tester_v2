package redis_mock_server

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL   string
	AccessKey string
	Client    *http.Client
}

func NewClient(baseURL, accessKey string) *Client {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	if strings.HasPrefix(strings.ToLower(baseURL), "https://") {
		httpClient.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	}

	return &Client{
		BaseURL:   baseURL,
		AccessKey: accessKey,
		Client:    httpClient,
	}
}

func (c *Client) execute(req RedisRequest) (*RedisResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.BaseURL+"/redis-execute", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Access-Key", c.AccessKey)

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: invalid access key")
	}

	var result RedisResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// Ping sends a PING command to verify connectivity.
func (c *Client) Ping() error {
	resp, err := c.execute(RedisRequest{Command: CmdPing})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("ping failed: %s", resp.Error)
	}
	return nil
}

// Set sets a key with expiration.
func (c *Client) Set(key string, value interface{}, expiration time.Duration) error {
	resp, err := c.execute(RedisRequest{
		Command:    CmdSet,
		Key:        key,
		Value:      value,
		Expiration: expiration,
	})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("SET failed: %s", resp.Error)
	}
	return nil
}

// Get retrieves a key value.
func (c *Client) Get(key string) (string, error) {
	resp, err := c.execute(RedisRequest{
		Command: CmdGet,
		Key:     key,
	})
	if err != nil {
		return "", err
	}
	if !resp.Success {
		return "", fmt.Errorf("%s", resp.Error)
	}
	return fmt.Sprintf("%v", resp.Data), nil
}

// Del deletes keys.
func (c *Client) Del(keys ...string) error {
	resp, err := c.execute(RedisRequest{
		Command: CmdDel,
		Keys:    keys,
	})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("DEL failed: %s", resp.Error)
	}
	return nil
}

// Exists checks if a key exists. Returns the count of existing keys.
func (c *Client) Exists(key string) (int64, error) {
	resp, err := c.execute(RedisRequest{
		Command: CmdExists,
		Key:     key,
	})
	if err != nil {
		return 0, err
	}
	if !resp.Success {
		return 0, fmt.Errorf("EXISTS failed: %s", resp.Error)
	}
	// JSON numbers decode as float64
	val, ok := resp.Data.(float64)
	if !ok {
		return 0, fmt.Errorf("unexpected EXISTS response type: %T", resp.Data)
	}
	return int64(val), nil
}

// HSet sets a field in a hash.
func (c *Client) HSet(key, field string, value interface{}) error {
	resp, err := c.execute(RedisRequest{
		Command: CmdHSet,
		Key:     key,
		Field:   field,
		Value:   value,
	})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("HSET failed: %s", resp.Error)
	}
	return nil
}

// HGet retrieves a field value from a hash.
func (c *Client) HGet(key, field string) (string, error) {
	resp, err := c.execute(RedisRequest{
		Command: CmdHGet,
		Key:     key,
		Field:   field,
	})
	if err != nil {
		return "", err
	}
	if !resp.Success {
		return "", fmt.Errorf("%s", resp.Error)
	}
	return fmt.Sprintf("%v", resp.Data), nil
}

// HIncrBy increments a hash field by the given integer amount.
func (c *Client) HIncrBy(key, field string, increment int64) (int64, error) {
	resp, err := c.execute(RedisRequest{
		Command:   CmdHIncrBy,
		Key:       key,
		Field:     field,
		Increment: increment,
	})
	if err != nil {
		return 0, err
	}
	if !resp.Success {
		return 0, fmt.Errorf("HINCRBY failed: %s", resp.Error)
	}
	val, ok := resp.Data.(float64)
	if !ok {
		return 0, fmt.Errorf("unexpected HINCRBY response type: %T", resp.Data)
	}
	return int64(val), nil
}

// TTL retrieves the TTL for a key.
func (c *Client) TTL(key string) (time.Duration, error) {
	resp, err := c.execute(RedisRequest{
		Command: CmdTTL,
		Key:     key,
	})
	if err != nil {
		return 0, err
	}
	if !resp.Success {
		return 0, fmt.Errorf("TTL failed: %s", resp.Error)
	}
	// TTL comes back as nanoseconds (time.Duration is int64 nanoseconds)
	val, ok := resp.Data.(float64)
	if !ok {
		return 0, fmt.Errorf("unexpected TTL response type: %T", resp.Data)
	}
	return time.Duration(int64(val)), nil
}

// FlushDB removes all keys from the current database.
func (c *Client) FlushDB() error {
	resp, err := c.execute(RedisRequest{Command: CmdFlushDB})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("FLUSHDB failed: %s", resp.Error)
	}
	return nil
}
