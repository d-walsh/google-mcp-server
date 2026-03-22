package docs

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"go.ngs.io/google-mcp-server/auth"
	"go.ngs.io/google-mcp-server/server"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var (
	// ErrNoAccount is returned when no account is available for the operation
	ErrNoAccount = errors.New("no account available")
)

// MultiAccountHandler handles docs operations across multiple Google accounts
type MultiAccountHandler struct {
	accountManager *auth.AccountManager
}

// NewMultiAccountHandler creates a new multi-account docs handler
func NewMultiAccountHandler(accountManager *auth.AccountManager) *MultiAccountHandler {
	return &MultiAccountHandler{
		accountManager: accountManager,
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

// resolveAccount resolves an account — requires explicit account when multiple exist
func (h *MultiAccountHandler) resolveAccount(ctx context.Context, accountHint string) (*auth.Account, error) {
	return h.accountManager.ResolveAccount(ctx, accountHint)
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

	// Handle docs_document_export_pdf separately since it needs Drive API
	if name == "docs_document_export_pdf" {
		return h.handleExportPDF(ctx, arguments, accountHint)
	}

	// Get account from context
	account, err := h.resolveAccount(ctx, accountHint)
	if err != nil {
		return nil, err
	}

	client, err := h.getClientForAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	// Delegate to the regular handler
	handler := NewHandler(client)
	return handler.HandleToolCall(ctx, name, arguments)
}

// handleExportPDF exports a Google Doc as PDF using the Drive API
func (h *MultiAccountHandler) handleExportPDF(ctx context.Context, arguments json.RawMessage, accountHint string) (interface{}, error) {
	var args struct {
		DocumentID string `json:"document_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	account, err := h.resolveAccount(ctx, accountHint)
	if err != nil {
		return nil, err
	}

	// Create Drive service to export
	driveSvc, err := drive.NewService(ctx, option.WithHTTPClient(account.OAuthClient.GetHTTPClient()))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service for export: %w", err)
	}

	// Export as PDF
	resp, err := driveSvc.Files.Export(args.DocumentID, "application/pdf").Download()
	if err != nil {
		return nil, fmt.Errorf("failed to export document as PDF: %w", err)
	}
	defer resp.Body.Close()

	// Read the PDF content
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read PDF content: %w", err)
	}

	return map[string]interface{}{
		"document_id": args.DocumentID,
		"data":        base64.StdEncoding.EncodeToString(buf.Bytes()),
		"size":        buf.Len(),
		"mimeType":    "application/pdf",
	}, nil
}

// GetTools returns the list of available tools
func (h *MultiAccountHandler) GetTools() []server.Tool {
	// Tool definitions are static and don't require a live client
	handler := NewHandler(nil)
	tools := handler.GetTools()

	// Add account parameter to existing tools
	for i := range tools {
		if tools[i].InputSchema.Properties == nil {
			tools[i].InputSchema.Properties = make(map[string]server.Property)
		}
		tools[i].InputSchema.Properties["account"] = server.AccountProperty
	}

	// Add new tools
	tools = append(tools, server.Tool{
		Name:        "docs_document_export_pdf",
		Description: "Export a Google Doc as PDF. Returns the PDF content as base64-encoded data.",
		InputSchema: server.InputSchema{
			Type: "object",
			Properties: map[string]server.Property{
				"document_id": {
					Type:        "string",
					Description: "Document ID to export",
				},
				"account": server.AccountProperty,
			},
			Required: []string{"document_id"},
		},
	})

	return tools
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
