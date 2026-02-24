package sheets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.ngs.io/google-mcp-server/auth"
	"go.ngs.io/google-mcp-server/server"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var (
	// ErrNoAccount is returned when no account is available for the operation
	ErrNoAccount = errors.New("no account available")
)

// MultiAccountHandler handles sheets operations across multiple Google accounts
type MultiAccountHandler struct {
	accountManager *auth.AccountManager
	defaultClient  *Client
}

// NewMultiAccountHandler creates a new multi-account sheets handler
func NewMultiAccountHandler(accountManager *auth.AccountManager, defaultClient *Client) *MultiAccountHandler {
	return &MultiAccountHandler{
		accountManager: accountManager,
		defaultClient:  defaultClient,
	}
}

// getClientForAccount creates a sheets client for a specific account
func (h *MultiAccountHandler) getClientForAccount(ctx context.Context, account *auth.Account) (*Client, error) {
	if account == nil || account.OAuthClient == nil {
		return nil, ErrNoAccount
	}

	service, err := sheets.NewService(ctx, option.WithHTTPClient(account.OAuthClient.GetHTTPClient()))
	if err != nil {
		return nil, fmt.Errorf("failed to create sheets service: %w", err)
	}

	return &Client{service: service}, nil
}

// HandleToolCall routes tool calls to the appropriate account
func (h *MultiAccountHandler) HandleToolCall(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	// Extract account hint from arguments
	var accountHint string
	if arguments != nil {
		var args map[string]interface{}
		if err := json.Unmarshal(arguments, &args); err == nil {
			if account, ok := args["account"].(string); ok {
				accountHint = account
			}
		}
	}

	// Get account from context
	account, err := h.accountManager.GetAccountForContext(ctx, accountHint)
	if err != nil || account == nil {
		// Try to use the first available account
		accounts := h.accountManager.ListAccounts()
		if len(accounts) > 0 {
			account = accounts[0]
		}
	}

	var client *Client
	if account != nil {
		client, err = h.getClientForAccount(ctx, account)
		if err != nil {
			// Fall back to default client if available
			if h.defaultClient != nil {
				client = h.defaultClient
			} else {
				return nil, err
			}
		}
	} else if h.defaultClient != nil {
		client = h.defaultClient
	} else {
		return nil, ErrNoAccount
	}

	// Delegate to the regular handler
	handler := NewHandler(client)
	return handler.HandleToolCall(ctx, name, arguments)
}

// GetTools returns the list of available tools
func (h *MultiAccountHandler) GetTools() []server.Tool {
	var tools []server.Tool
	// Use default client or create a temporary one for tool definitions
	if h.defaultClient != nil {
		handler := NewHandler(h.defaultClient)
		tools = handler.GetTools()
	}

	// Add account parameter to existing tools
	for i := range tools {
		if tools[i].InputSchema.Properties == nil {
			tools[i].InputSchema.Properties = make(map[string]server.Property)
		}
		tools[i].InputSchema.Properties["account"] = server.Property{
			Type:        "string",
			Description: "Email address of the account to use (optional)",
		}
	}

	return tools
}

// GetResources returns the list of available resources
func (h *MultiAccountHandler) GetResources() []server.Resource {
	return []server.Resource{}
}

// HandleResourceCall handles resource calls
func (h *MultiAccountHandler) HandleResourceCall(ctx context.Context, uri string) (interface{}, error) {
	// Sheets doesn't have resources yet, return not found
	return nil, fmt.Errorf("resource not found: %s", uri)
}
