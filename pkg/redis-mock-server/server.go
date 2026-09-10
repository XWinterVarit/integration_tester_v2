package redis_mock_server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisServer struct {
	ControlPort int
	AccessKey   string
	RedisClient *redis.Client
	Logger      *Logger
}

func NewRedisServer(controlPort int, accessKey string, redisAddr, redisPassword string, redisDB int, logger *Logger) *RedisServer {
	return &RedisServer{
		ControlPort: controlPort,
		AccessKey:   accessKey,
		RedisClient: redis.NewClient(&redis.Options{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       redisDB,
		}),
		Logger: logger,
	}
}

func (s *RedisServer) Start() error {
	// Verify Redis connection
	if err := s.RedisClient.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	s.Logger.Log("RedisConnected", 0, "Connected to Redis")

	mux := http.NewServeMux()
	mux.HandleFunc("/redis-execute", s.handleExecute)
	mux.HandleFunc("/", s.handleNotFound)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.ControlPort),
		Handler: mux,
	}

	s.Logger.Log("ServerStart", 0, fmt.Sprintf("Starting Redis Mock Server on port %d", s.ControlPort))
	return server.ListenAndServe()
}

func (s *RedisServer) authenticate(r *http.Request) bool {
	return r.Header.Get("X-Access-Key") == s.AccessKey
}

func (s *RedisServer) handleExecute(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.authenticate(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req RedisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	var resp RedisResponse

	switch req.Command {
	case CmdPing:
		err := s.RedisClient.Ping(ctx).Err()
		if err != nil {
			resp = RedisResponse{Success: false, Error: err.Error()}
		} else {
			resp = RedisResponse{Success: true, Data: "PONG"}
		}

	case CmdSet:
		valStr := fmt.Sprintf("%v", req.Value)
		err := s.RedisClient.Set(ctx, req.Key, valStr, req.Expiration).Err()
		if err != nil {
			resp = RedisResponse{Success: false, Error: err.Error()}
		} else {
			resp = RedisResponse{Success: true}
		}

	case CmdGet:
		val, err := s.RedisClient.Get(ctx, req.Key).Result()
		if err != nil {
			errMsg := err.Error()
			if err == redis.Nil {
				errMsg = "redis: nil"
			}
			resp = RedisResponse{Success: false, Error: errMsg}
		} else {
			resp = RedisResponse{Success: true, Data: val}
		}

	case CmdDel:
		keys := req.Keys
		if len(keys) == 0 && req.Key != "" {
			keys = []string{req.Key}
		}
		err := s.RedisClient.Del(ctx, keys...).Err()
		if err != nil {
			resp = RedisResponse{Success: false, Error: err.Error()}
		} else {
			resp = RedisResponse{Success: true}
		}

	case CmdExists:
		val, err := s.RedisClient.Exists(ctx, req.Key).Result()
		if err != nil {
			resp = RedisResponse{Success: false, Error: err.Error()}
		} else {
			resp = RedisResponse{Success: true, Data: val}
		}

	case CmdHSet:
		valStr := fmt.Sprintf("%v", req.Value)
		err := s.RedisClient.HSet(ctx, req.Key, req.Field, valStr).Err()
		if err != nil {
			resp = RedisResponse{Success: false, Error: err.Error()}
		} else {
			resp = RedisResponse{Success: true}
		}

	case CmdHGet:
		val, err := s.RedisClient.HGet(ctx, req.Key, req.Field).Result()
		if err != nil {
			errMsg := err.Error()
			if err == redis.Nil {
				errMsg = "redis: nil"
			}
			resp = RedisResponse{Success: false, Error: errMsg}
		} else {
			resp = RedisResponse{Success: true, Data: val}
		}

	case CmdHIncrBy:
		val, err := s.RedisClient.HIncrBy(ctx, req.Key, req.Field, req.Increment).Result()
		if err != nil {
			resp = RedisResponse{Success: false, Error: err.Error()}
		} else {
			resp = RedisResponse{Success: true, Data: val}
		}

	case CmdTTL:
		val, err := s.RedisClient.TTL(ctx, req.Key).Result()
		if err != nil {
			resp = RedisResponse{Success: false, Error: err.Error()}
		} else {
			resp = RedisResponse{Success: true, Data: val}
		}

	case CmdFlushDB:
		err := s.RedisClient.FlushDB(ctx).Err()
		if err != nil {
			resp = RedisResponse{Success: false, Error: err.Error()}
		} else {
			resp = RedisResponse{Success: true}
		}

	default:
		resp = RedisResponse{Success: false, Error: fmt.Sprintf("unknown command: %s", req.Command)}
	}

	s.Logger.Log("Execute", time.Since(start), map[string]interface{}{
		"command": req.Command,
		"key":     req.Key,
		"success": resp.Success,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *RedisServer) sendError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(RedisResponse{Success: false, Error: msg})
}

func (s *RedisServer) handleNotFound(w http.ResponseWriter, r *http.Request) {
	s.Logger.Log("NotFound", 0, map[string]interface{}{
		"path":   r.URL.Path,
		"method": r.Method,
	})
	http.NotFound(w, r)
}
