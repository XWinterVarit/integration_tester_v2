package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	dms "github.com/XWinterVarit/integrate_tester_v2/pkg/dynamic-mock-server"
)

func main() {
	port := flag.Int("port", 9001, "Port for the mock controller")
	logFile := flag.String("log", "", "Log file path (default: stdout)")
	flag.Parse()

	var logger *dms.Logger
	var err error

	if *logFile == "" {
		logger = dms.NewConsoleLogger()
	} else {
		logger, err = dms.NewLogger(*logFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
			os.Exit(1)
		}
	}
	defer logger.Close()

	controller := dms.NewMockController(*port, logger)

	fmt.Printf("Starting Dynamic Mock Server Controller on port %d...\n", *port)
	if *logFile == "" {
		fmt.Println("Logging to stdout")
	} else {
		fmt.Printf("Logging to %s\n", *logFile)
	}

	if err := controller.Start(); err != nil {
		log.Fatalf("Mock Controller failed: %v", err)
	}
}
