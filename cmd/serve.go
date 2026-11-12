package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"open-parcels/pkg/server"
	"open-parcels/pkg/storage"
)

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 8080, "Port to listen on")
	fs.IntVar(port, "p", 8080, "Port to listen on (shorthand)")
	enableMCP := fs.Bool("mcp", true, "Enable MCP streamable HTTP endpoint at /mcp")
	dbPath := fs.String("db", "", "Path to SQLite database file (or OPEN_PARCELS_DB_PATH)")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var st storage.Store
	targetDB := *dbPath
	if targetDB == "" {
		targetDB = os.Getenv(storage.EnvDBPath)
	}
	if targetDB != "" {
		var err error
		st, err = storage.NewStore(targetDB)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to open store at %s: %v, using memory fallback\n", targetDB, err)
			st = storage.NewMemoryStore()
		} else {
			defer st.Close()
		}
	}

	srv := server.New(server.Config{
		Port:      *port,
		EnableMCP: *enableMCP,
		Store:     st,
	})

	fmt.Printf("Starting Open Parcels Daemon on port %d...\n", *port)
	if err := srv.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
