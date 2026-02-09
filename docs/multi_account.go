package docs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.ngs.io/google-mcp-server/auth"
	"go.ngs.io/google-mcp-server/server"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/option"
)

var (
	// ErrNoAccount is returned when no account is available for the operation
	ErrNoAccount = errors.New("no account available")
)

// MultiAccountHandler handles docs operations across multiple Google accounts
type MultiAccountHandler struct {
	accountManager *auth.AccountManager
	defaultClient  *Client
}

// NewMultiAccountHandler creates a new multi-account docs handler
func NewMultiAccountHandler(accountManager *auth.AccountManager, defaultClient *Client) *MultiAccountHandler {
	return &MultiAccountHandler{
		accountManager: accountManager,
		defaultClient:  defaultClient,
	}
}

// getClientForAccount creates a docs client for a specific account
func (h *MultiAccountHandler) getClientForAccount(ctx context.Context, account *auth.Account) (*Client, error) {
	if account == nil || account.OAuthClient == nil {
		return nil, ErrNoAccount
	}

	service, err := docs.NewService(ctx, option.WithHTTPClient(account.OAuthClient.GetHTTPClient()))
	if err != nil {
		return nil, fmt.Errorf("failed to create docs service: %w", err)
	}

	return &Client{service: service}, nil
}

// HandleToolCall routes tool calls to the appropriate account
func (h *MultiAccountHandler) HandleToolCall(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	// Get account from context
	account, err := h.accountManager.GetAccountForContext(ctx, "")
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
	// Use default client or create a temporary one for tool definitions
	if h.defaultClient != nil {
		handler := NewHandler(h.defaultClient)
		return handler.GetTools()
	}
	// Return empty tools list if no default client
	// Tools will be registered when an account is added
	return []server.Tool{}
}

// GetResources returns the list of available resources
func (h *MultiAccountHandler) GetResources() []server.Resource {
	return []server.Resource{}
}

// HandleResourceCall handles resource calls
func (h *MultiAccountHandler) HandleResourceCall(ctx context.Context, uri string) (interface{}, error) {
	// Docs doesn't have resources yet, return not found
	return nil, fmt.Errorf("resource not found: %s", uri)
}
