package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/status-im/eth-rpc-proxy/api"
	"github.com/status-im/eth-rpc-proxy/config"
	"github.com/status-im/eth-rpc-proxy/core"
	requestsrunner "github.com/status-im/eth-rpc-proxy/requests_runner"
	"github.com/status-im/eth-rpc-proxy/scheduler"
)

func main() {
	// Parse command line flags
	checkerConfigPath := flag.String("checker-config", "checker_config.json", "path to checker config")
	defaultProvidersPath := flag.String("default-providers", "", "path to default providers JSON")
	referenceProvidersPath := flag.String("reference-providers", "", "path to reference providers JSON")
	flag.Parse()

	// Read configuration
	config, err := config.ReadCheckerConfig(*checkerConfigPath)
	if err != nil {
		log.Fatalf("failed to read checker configuration: %v", err)
	}

	// Set provider paths from flags if provided
	if *defaultProvidersPath != "" {
		config.DefaultProvidersPath = *defaultProvidersPath
	}
	if *referenceProvidersPath != "" {
		config.ReferenceProvidersPath = *referenceProvidersPath
	}
	if err != nil {
		log.Fatalf("failed to read configuration: %v", err)
	}

	// Create EVM method caller using RequestsRunner
	caller := requestsrunner.NewRequestsRunner()

	// Create validation function
	validationFunc := func() {
		// Create fresh runner for each execution
		log.Printf("config: %v", config)
		runner, err := core.NewRunnerFromConfig(*config, caller)
		if err != nil {
			log.Printf("failed to create runner: %v", err)
			return
		}
		runner.Run(context.Background())
	}

	// Create periodic task for running validation
	validationTask := scheduler.New(
		time.Duration(config.IntervalSeconds)*time.Second,
		validationFunc,
	)

	// Run initial validation
	validationFunc()

	// Start the periodic task
	validationTask.Start()
	defer validationTask.Stop()

	// Start HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := api.New(port, config.OutputProvidersPath, config.DefaultProvidersPath)
	if err := server.Start(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
