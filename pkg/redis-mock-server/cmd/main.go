package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	rms "github.com/XWinterVarit/integrate_tester_v2/pkg/redis-mock-server"
)

func main() {
	port := flag.Int("port", 9100, "Port for the Redis Mock Server")
	accessKey := flag.String("key", "", "Access key for authentication (required)")
	redisAddr := flag.String("redis-addr", "localhost:6379", "Redis server address")
	redisPassword := flag.String("redis-password", "", "Redis password")
	redisDB := flag.Int("redis-db", 0, "Redis database number")
	logFile := flag.String("log", "", "Log file path (default: stdout)")
	flag.Parse()

	if *accessKey == "" {
		fmt.Fprintln(os.Stderr, "Error: -key flag is required")
		flag.Usage()
		os.Exit(1)
	}

	var logger *rms.Logger
	var err error

	if *logFile == "" {
		logger = rms.NewConsoleLogger()
	} else {
		logger, err = rms.NewLogger(*logFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
			os.Exit(1)
		}
	}
	defer logger.Close()

	server := rms.NewRedisServer(*port, *accessKey, *redisAddr, *redisPassword, *redisDB, logger)

	fmt.Printf("Starting Redis Mock Server on port %d...\n", *port)
	if *logFile == "" {
		fmt.Println("Logging to stdout")
	} else {
		fmt.Printf("Logging to %s\n", *logFile)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("Redis Mock Server failed: %v", err)
	}
}
