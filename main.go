package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.ngs.io/google-mcp-server/accounts"
	"go.ngs.io/google-mcp-server/auth"
	"go.ngs.io/google-mcp-server/calendar"
	"go.ngs.io/google-mcp-server/config"
	"go.ngs.io/google-mcp-server/docs"
	"go.ngs.io/google-mcp-server/drive"
	"go.ngs.io/google-mcp-server/gmail"
	"go.ngs.io/google-mcp-server/server"
	"go.ngs.io/google-mcp-server/sheets"
	"go.ngs.io/google-mcp-server/slides"
	"go.ngs.io/google-mcp-server/tasks"
)

func main() {
	// Set up logging immediately with no buffering
	log.SetOutput(os.Stderr)
	log.SetFlags(0) // Remove flags for cleaner MCP output

	// Check for version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("google-mcp-server v0.1.0")
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize account manager for multi-account support
	ctx := context.Background()
	accountManager, err := auth.NewAccountManager(ctx, cfg.OAuth)
	if err != nil {
		log.Fatalf("Failed to initialize account manager: %v", err)
	}
	log.Printf("[INFO] Account manager initialized with %d accounts\n", len(accountManager.ListAccounts()))

	// Initialize MCP server
	mcpServer := server.NewMCPServer(cfg)

	// Register services before starting the server
	log.Println("[INFO] Starting service registration...")
	if err := registerServices(ctx, mcpServer, accountManager, cfg); err != nil {
		log.Printf("[WARNING] Some services failed to register: %v", err)
	} else {
		log.Println("[INFO] All services registered successfully")
	}

	// Start the server (blocks until shutdown)
	if err := mcpServer.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func registerServices(ctx context.Context, srv *server.MCPServer, am *auth.AccountManager, cfg *config.Config) error {
	// Register account management service
	accountsHandler := accounts.NewHandler(am)
	srv.RegisterService("accounts", accountsHandler)
	log.Println("[DEBUG] Accounts service registered")

	if cfg.Services.Calendar.Enabled {
		log.Println("[DEBUG] Initializing Calendar service...")
		srv.RegisterService("calendar", calendar.NewMultiAccountHandler(am))
		log.Println("[DEBUG] Calendar service registered with multi-account support")
	}

	if cfg.Services.Drive.Enabled {
		log.Println("[DEBUG] Initializing Drive service...")
		srv.RegisterService("drive", drive.NewMultiAccountHandler(am))
		log.Println("[DEBUG] Drive service registered with multi-account support")
	}

	if cfg.Services.Gmail.Enabled {
		log.Println("[DEBUG] Initializing Gmail service...")
		srv.RegisterService("gmail", gmail.NewMultiAccountHandler(am))
		log.Println("[DEBUG] Gmail service registered with multi-account support")
	}

	if cfg.Services.Sheets.Enabled {
		log.Println("[DEBUG] Initializing Sheets service...")
		srv.RegisterService("sheets", sheets.NewMultiAccountHandler(am))
		log.Println("[DEBUG] Sheets service registered with multi-account support")
	}

	if cfg.Services.Docs.Enabled {
		log.Println("[DEBUG] Initializing Docs service...")
		srv.RegisterService("docs", docs.NewMultiAccountHandler(am))
		log.Println("[DEBUG] Docs service registered with multi-account support")
	}

	if cfg.Services.Slides.Enabled {
		log.Println("[DEBUG] Initializing Slides service...")
		srv.RegisterService("slides", slides.NewMultiAccountHandler(am))
		log.Println("[DEBUG] Slides service registered with multi-account support")
	}

	if cfg.Services.Tasks.Enabled {
		log.Println("[DEBUG] Initializing Tasks service...")
		srv.RegisterService("tasks", tasks.NewMultiAccountHandler(am))
		log.Println("[DEBUG] Tasks service registered with multi-account support")
	}

	return nil
}

func init() {
	// Logging is now set up in main()
}
