package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alipio/awsconfigurator-go/configurator"
	"github.com/joho/godotenv"
)

func main() {
	os.Exit(run())
}

func run() int {
	_ = godotenv.Load()

	configPath := flag.String("config", "", "Path to the configuration file (required)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -config=<path_to_config>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "Error: config file path is required")
		flag.Usage()
		return 1
	}

	cfg, err := configurator.LoadConfig(*configPath)
	if err != nil {
		var errIC *configurator.InvalidConfigError
		if errors.As(err, &errIC) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", errIC)
			return 2
		}
		fmt.Fprintf(os.Stderr, "Failed to parse config: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	awsProvider, err := configurator.NewAwsProvider(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize AWS provider: %v\n", err)
		return 1
	}

	configurator := configurator.New(awsProvider, cfg)
	if err := configurator.Run(ctx); err != nil { //nolint:govet // shadowing is ok here.
		fmt.Fprintf(os.Stderr, "Configuration failed: %v\n", err)
		return 1
	}

	log.Println("Configuration completed successfully")

	return 0
}
