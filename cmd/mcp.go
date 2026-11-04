package cmd

import (
	"fmt"
	"os"

	"open-parcels/pkg/engine"
	"open-parcels/pkg/mcp"
)

func runMCP(args []string) {
	eng := engine.New()
	srv := mcp.NewServer(eng)

	if err := srv.RunStdio(); err != nil {
		fmt.Fprintf(os.Stderr, "MCP stdio server terminated with error: %v\n", err)
		os.Exit(1)
	}
}
