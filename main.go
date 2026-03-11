package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tickory/tickory-mcp/mcp"
)

var Version = "dev"

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "tickory-mcp: %v\n", err)
		os.Exit(1)
	}

	client, err := mcp.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tickory-mcp: %v\n", err)
		os.Exit(1)
	}

	server := mcp.NewServer(client, Version)
	if err := server.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "tickory-mcp: %v\n", err)
		os.Exit(1)
	}
}

func loadConfig() (mcp.Config, error) {
	baseURLFlag := flag.String("api-base-url", envOrDefault("TICKORY_API_BASE_URL", ""), "Tickory API base URL")
	apiKeyFlag := flag.String("api-key", envOrDefault("TICKORY_API_KEY", ""), "Tickory API key (prefer TICKORY_API_KEY env var; CLI args may be visible in process listings)")
	timeoutFlag := flag.Duration("timeout", envDurationOrDefault("TICKORY_TIMEOUT_SECONDS", 15*time.Second), "HTTP timeout for Tickory API requests")
	flag.Parse()

	return mcp.Config{
		BaseURL: strings.TrimSpace(*baseURLFlag),
		APIKey:  strings.TrimSpace(*apiKeyFlag),
		Timeout: *timeoutFlag,
	}, nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return fallback
	}

	return time.Duration(seconds) * time.Second
}
