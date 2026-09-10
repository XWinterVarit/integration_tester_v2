package v1

import (
	"encoding/json"
	"fmt"
	"time"

	rms "github.com/XWinterVarit/integrate_tester_v2/pkg/redis-mock-server"
)

// RedisClient wraps a redis-mock-server client for test helpers.
type RedisClient struct {
	client *rms.Client
}

// ConnectRedis connects to Redis Mock Server using server address and access key.
func ConnectRedis(serverAddr, accessKey string) *RedisClient {
	RecordAction(fmt.Sprintf("Redis Connect: %s", serverAddr), func() { ConnectRedis(serverAddr, accessKey) })
	if IsDryRun() {
		return &RedisClient{}
	}
	Logf(LogTypeRedis, "Connecting to Redis Mock Server at %s", serverAddr)
	c := rms.NewClient(serverAddr, accessKey)
	if err := c.Ping(); err != nil {
		Fail("Failed to connect to Redis Mock Server: %v", err)
	}
	Log(LogTypeRedis, "Connected to Redis Mock Server", "")
	return &RedisClient{client: c}
}

// Set sets a key with expiration.
func (c *RedisClient) Set(key string, value interface{}, expiration time.Duration) {
	RecordAction(fmt.Sprintf("Redis Set: %s", key), func() { c.Set(key, value, expiration) })
	if IsDryRun() {
		return
	}
	if c.client == nil {
		Fail("RedisClient is not connected")
	}
	Log(LogTypeRedis, fmt.Sprintf("SET %s", key), fmt.Sprintf("value=%v, ttl=%s", value, expiration))
	if err := c.client.Set(key, value, expiration); err != nil {
		Fail("Failed to set redis key %s: %v", key, err)
	}
}

// Get retrieves a key value.
func (c *RedisClient) Get(key string) string {
	RecordAction(fmt.Sprintf("Redis Get: %s", key), func() { c.Get(key) })
	if IsDryRun() {
		return ""
	}
	if c.client == nil {
		Fail("RedisClient is not connected")
	}
	Logf(LogTypeRedis, "GET %s", key)
	val, err := c.client.Get(key)
	if err != nil {
		if err.Error() == "redis: nil" {
			Fail("Redis key %s not found", key)
		}
		Fail("Failed to get redis key %s: %v", key, err)
	}
	return val
}

// Del deletes keys.
func (c *RedisClient) Del(keys ...string) {
	RecordAction(fmt.Sprintf("Redis Del: %v", keys), func() { c.Del(keys...) })
	if IsDryRun() {
		return
	}
	if c.client == nil {
		Fail("RedisClient is not connected")
	}
	Log(LogTypeRedis, "DEL keys", fmt.Sprintf("%v", keys))
	if err := c.client.Del(keys...); err != nil {
		Fail("Failed to delete redis keys %v: %v", keys, err)
	}
}

// ExpectValue asserts that a key has the expected value.
func (c *RedisClient) ExpectValue(key string, expected string) {
	if IsDryRun() {
		return
	}
	val := c.Get(key)
	if val != expected {
		Fail("Redis value mismatch for key %s: expected %s, got %s", key, expected, val)
	}
	Logf(LogTypeExpect, "Redis key %s == %s - PASSED", key, expected)
}

// ExpectFound asserts that a key exists in Redis.
func (c *RedisClient) ExpectFound(key string) {
	if IsDryRun() {
		return
	}
	if c.client == nil {
		Fail("RedisClient is not connected")
	}
	exists, err := c.client.Exists(key)
	if err != nil {
		Fail("Failed to check existence of redis key %s: %v", key, err)
	}
	if exists == 0 {
		Fail("Expected redis key %s to exist, but it was not found", key)
	}
	Logf(LogTypeExpect, "Redis key %s exists - PASSED", key)
}

// ExpectNotFound asserts that a key does not exist in Redis.
func (c *RedisClient) ExpectNotFound(key string) {
	if IsDryRun() {
		return
	}
	if c.client == nil {
		Fail("RedisClient is not connected")
	}
	exists, err := c.client.Exists(key)
	if err != nil {
		Fail("Failed to check existence of redis key %s: %v", key, err)
	}
	if exists != 0 {
		Fail("Expected redis key %s to not exist, but it was found", key)
	}
	Logf(LogTypeExpect, "Redis key %s does not exist - PASSED", key)
}

// HSet sets a field in a hash.
func (c *RedisClient) HSet(key, field string, value interface{}) {
	RecordAction(fmt.Sprintf("Redis HSet: %s %s", key, field), func() { c.HSet(key, field, value) })
	if IsDryRun() {
		return
	}
	if c.client == nil {
		Fail("RedisClient is not connected")
	}
	Log(LogTypeRedis, fmt.Sprintf("HSET %s %s", key, field), fmt.Sprintf("value=%v", value))
	if err := c.client.HSet(key, field, value); err != nil {
		Fail("Failed to hset redis key %s field %s: %v", key, field, err)
	}
}

// HGet retrieves a field value from a hash.
func (c *RedisClient) HGet(key, field string) string {
	RecordAction(fmt.Sprintf("Redis HGet: %s %s", key, field), func() { c.HGet(key, field) })
	if IsDryRun() {
		return ""
	}
	if c.client == nil {
		Fail("RedisClient is not connected")
	}
	Logf(LogTypeRedis, "HGET %s %s", key, field)
	val, err := c.client.HGet(key, field)
	if err != nil {
		if err.Error() == "redis: nil" {
			Fail("Redis hash key %s field %s not found", key, field)
		}
		Fail("Failed to hget redis key %s field %s: %v", key, field, err)
	}
	return val
}

// HIncrement increments a hash field by the given integer amount.
func (c *RedisClient) HIncrement(key, field string, increment int64) int64 {
	RecordAction(fmt.Sprintf("Redis HIncrement: %s %s by %d", key, field, increment), func() {
		c.HIncrement(key, field, increment)
	})
	if IsDryRun() {
		return 0
	}
	if c.client == nil {
		Fail("RedisClient is not connected")
	}
	Logf(LogTypeRedis, "HINCRBY %s %s %d", key, field, increment)
	val, err := c.client.HIncrBy(key, field, increment)
	if err != nil {
		Fail("Failed to hincrby redis key %s field %s: %v", key, field, err)
	}
	return val
}

// SetJsonField retrieves the JSON value stored at key, sets the field at the given
// dot+bracket path (e.g. "a.b[0].c") to value, and saves the updated JSON back.
// Fails if the key is not found, the value is not valid JSON, or the path is invalid.
func (c *RedisClient) SetJsonField(key string, path string, value interface{}) {
	RecordAction(fmt.Sprintf("Redis SetJsonField: %s %s", key, path), func() { c.SetJsonField(key, path, value) })
	if IsDryRun() {
		return
	}
	if c.client == nil {
		Fail("RedisClient is not connected")
	}
	Logf(LogTypeRedis, "SetJsonField %s path=%s value=%v", key, path, value)

	raw, err := c.client.Get(key)
	if err != nil {
		if err.Error() == "redis: nil" {
			Fail("Redis key %s not found", key)
		}
		Fail("Failed to get redis key %s: %v", key, err)
	}

	var data interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		Fail("SetJsonField: value at key %s is not valid JSON: %v", key, err)
	}

	if err := setValueByPath(data, path, value); err != nil {
		Fail("SetJsonField: failed to set path '%s' on key %s: %v", path, key, err)
	}

	updated, err := json.Marshal(data)
	if err != nil {
		Fail("SetJsonField: failed to marshal updated JSON for key %s: %v", key, err)
	}

	ttl, err := c.client.TTL(key)
	if err != nil {
		Fail("SetJsonField: failed to get TTL for key %s: %v", key, err)
	}
	if err := c.client.Set(key, string(updated), ttl); err != nil {
		Fail("SetJsonField: failed to save updated JSON for key %s: %v", key, err)
	}
}

// HSetJsonField retrieves the JSON value stored in a hash field at key/field, sets the
// nested field at the given dot+bracket path (e.g. "a.b[0].c") to value, and saves
// the updated JSON back. Fails if the key/field is not found, the value is not valid
// JSON, or the path is invalid.
func (c *RedisClient) HSetJsonField(key, field, path string, value interface{}) {
	RecordAction(fmt.Sprintf("Redis HSetJsonField: %s %s %s", key, field, path), func() {
		c.HSetJsonField(key, field, path, value)
	})
	if IsDryRun() {
		return
	}
	if c.client == nil {
		Fail("RedisClient is not connected")
	}
	Logf(LogTypeRedis, "HSetJsonField %s field=%s path=%s value=%v", key, field, path, value)

	raw, err := c.client.HGet(key, field)
	if err != nil {
		if err.Error() == "redis: nil" {
			Fail("Redis hash key %s field %s not found", key, field)
		}
		Fail("Failed to hget redis key %s field %s: %v", key, field, err)
	}

	var data interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		Fail("HSetJsonField: value at key %s field %s is not valid JSON: %v", key, field, err)
	}

	if err := setValueByPath(data, path, value); err != nil {
		Fail("HSetJsonField: failed to set path '%s' on key %s field %s: %v", path, key, field, err)
	}

	updated, err := json.Marshal(data)
	if err != nil {
		Fail("HSetJsonField: failed to marshal updated JSON for key %s field %s: %v", key, field, err)
	}

	if err := c.client.HSet(key, field, string(updated)); err != nil {
		Fail("HSetJsonField: failed to save updated JSON for key %s field %s: %v", key, field, err)
	}
}

// HExpectJsonField retrieves the JSON value stored in a hash field at key/field, reads
// the nested field at the given dot+bracket path (e.g. "a.b[0].c"), and asserts it
// equals expectedValue. Fails if the key/field is not found, the value is not valid
// JSON, or the path is invalid.
func (c *RedisClient) HExpectJsonField(key, field, path string, expectedValue interface{}) {
	if IsDryRun() {
		return
	}
	if c.client == nil {
		Fail("RedisClient is not connected")
	}

	raw, err := c.client.HGet(key, field)
	if err != nil {
		if err.Error() == "redis: nil" {
			Fail("Redis hash key %s field %s not found", key, field)
		}
		Fail("Failed to hget redis key %s field %s: %v", key, field, err)
	}

	var data interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		Fail("HExpectJsonField: value at key %s field %s is not valid JSON: %v", key, field, err)
	}

	gotValue, err := getValueByPath(data, path)
	if err != nil {
		Fail("HExpectJsonField: failed to get path '%s' on key %s field %s: %v", path, key, field, err)
	}

	match := false
	if isNumber(gotValue) && isNumber(expectedValue) {
		match = toFloat64(gotValue) == toFloat64(expectedValue)
	} else {
		match = fmt.Sprintf("%v", gotValue) == fmt.Sprintf("%v", expectedValue)
	}

	if !match {
		Fail("HExpectJsonField failed for key %s field %s path '%s':\nExpected: %v (%T)\nGot:      %v (%T)", key, field, path, expectedValue, expectedValue, gotValue, gotValue)
	}
	Logf(LogTypeExpect, "Redis key %s field %s path '%s' == %v - PASSED", key, field, path, expectedValue)
}

// ExpectJsonField retrieves the JSON value stored at key, reads the field at the
// given dot+bracket path (e.g. "a.b[0].c"), and asserts it equals expectedValue.
// Fails if the key is not found, the value is not valid JSON, or the path is invalid.
func (c *RedisClient) ExpectJsonField(key string, path string, expectedValue interface{}) {
	if IsDryRun() {
		return
	}
	if c.client == nil {
		Fail("RedisClient is not connected")
	}

	raw, err := c.client.Get(key)
	if err != nil {
		if err.Error() == "redis: nil" {
			Fail("Redis key %s not found", key)
		}
		Fail("Failed to get redis key %s: %v", key, err)
	}

	var data interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		Fail("ExpectJsonField: value at key %s is not valid JSON: %v", key, err)
	}

	gotValue, err := getValueByPath(data, path)
	if err != nil {
		Fail("ExpectJsonField: failed to get path '%s' on key %s: %v", path, key, err)
	}

	match := false
	if isNumber(gotValue) && isNumber(expectedValue) {
		match = toFloat64(gotValue) == toFloat64(expectedValue)
	} else {
		match = fmt.Sprintf("%v", gotValue) == fmt.Sprintf("%v", expectedValue)
	}

	if !match {
		Fail("ExpectJsonField failed for key %s path '%s':\nExpected: %v (%T)\nGot:      %v (%T)", key, path, expectedValue, expectedValue, gotValue, gotValue)
	}
	Logf(LogTypeExpect, "Redis key %s path '%s' == %v - PASSED", key, path, expectedValue)
}

// FlushAll removes all keys from the current database.
func (c *RedisClient) FlushAll() {
	RecordAction("Redis FlushAll", func() { c.FlushAll() })
	if IsDryRun() {
		return
	}
	if c.client == nil {
		Fail("RedisClient is not connected")
	}
	Log(LogTypeRedis, "FLUSHALL", "")
	if err := c.client.FlushDB(); err != nil {
		Fail("Failed to flush redis db: %v", err)
	}
}
