package redis_mock_server

import "time"

// RedisRequest is the generic request body for all Redis operations.
type RedisRequest struct {
	Command    string        `json:"command"`
	Key        string        `json:"key,omitempty"`
	Field      string        `json:"field,omitempty"`
	Value      interface{}   `json:"value,omitempty"`
	Expiration time.Duration `json:"expiration,omitempty"`
	Increment  int64         `json:"increment,omitempty"`
	Keys       []string      `json:"keys,omitempty"`
}

// RedisResponse is the generic response body for all Redis operations.
type RedisResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Redis command constants
const (
	CmdPing    = "PING"
	CmdSet     = "SET"
	CmdGet     = "GET"
	CmdDel     = "DEL"
	CmdExists  = "EXISTS"
	CmdHSet    = "HSET"
	CmdHGet    = "HGET"
	CmdHIncrBy = "HINCRBY"
	CmdTTL     = "TTL"
	CmdFlushDB = "FLUSHDB"
)
