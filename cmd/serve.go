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

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 8080, "Port to listen on")
	fs.IntVar(port, "p", 8080, "Port to listen on (shorthand)")
	enableMCP := fs.Bool("mcp", true, "Enable MCP streamable HTTP endpoint at /mcp")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srv := server.New(server.Config{
		Port:      *port,
		EnableMCP: *enableMCP,
	})

	fmt.Printf("Starting Open Parcels Daemon on port %d...\n", *port)
	if err := srv.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
