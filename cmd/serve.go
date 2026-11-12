package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"open-parcels/pkg/server"
)

func parseServeFlags(args []string) (server.Config, error) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	port := fs.Int("port", 8080, "Port to listen on")
	fs.IntVar(port, "p", 8080, "Port to listen on (shorthand)")
	enableMCP := fs.Bool("mcp", true, "Enable MCP streamable HTTP endpoint at /mcp")
	apiKey := fs.String("api-key", "", "API key (or comma-separated keys) for Bearer / X-API-Key authentication")
	fs.StringVar(apiKey, "k", "", "API key (shorthand)")
	rateLimitStr := fs.String("rate-limit", "", "Rate limit per client (e.g. '10', '10/s', '100/m')")
	rateBurst := fs.Int("rate-burst", 0, "Rate limit burst allowance (default: equal to rate limit)")

	if err := fs.Parse(args); err != nil {
		return server.Config{}, err
	}

	var rateLimit float64
	if *rateLimitStr != "" {
		parsed, err := server.ParseRateLimit(*rateLimitStr)
		if err != nil {
			return server.Config{}, fmt.Errorf("invalid --rate-limit value: %w", err)
		}
		rateLimit = parsed
	}

	return server.Config{
		Port:      *port,
		EnableMCP: *enableMCP,
		APIKey:    *apiKey,
		RateLimit: rateLimit,
		RateBurst: *rateBurst,
	}, nil
}

func runServe(args []string) {
	cfg, err := parseServeFlags(args)
	if err != nil {
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srv := server.New(cfg)

	fmt.Printf("Starting Open Parcels Daemon on port %d...\n", cfg.Port)
	if err := srv.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
