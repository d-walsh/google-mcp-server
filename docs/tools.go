package docs

import (
	"context"
	"encoding/json"
	"fmt"

	"go.ngs.io/google-mcp-server/server"
)

// Handler implements the ServiceHandler interface for Docs
type Handler struct {
	client *Client
}

// NewHandler creates a new Docs handler
func NewHandler(client *Client) *Handler {
	return &Handler{client: client}
}

// GetTools returns the available Docs tools
func (h *Handler) GetTools() []server.Tool {
	return []server.Tool{
		{
			Name:        "docs_document_get",
			Description: "Get document content",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"document_id": {
						Type:        "string",
						Description: "Document ID",
					},
				},
				Required: []string{"document_id"},
			},
		},
		{
			Name:        "docs_document_create",
			Description: "Create a new plain text document (for Markdown use drive_markdown_upload instead)",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"title": {
						Type:        "string",
						Description: "Document title",
					},
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "docs_document_update",
			Description: "Update plain text document content - append or replace (for Markdown use drive_markdown_replace)",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"document_id": {
						Type:        "string",
						Description: "Document ID",
					},
					"content": {
						Type:        "string",
						Description: "Text content to add to the document",
					},
					"mode": {
						Type:        "string",
						Description: "Update mode: 'append' (default) or 'replace'",
					},
				},
				Required: []string{"document_id", "content"},
			},
		},
		{
			Name:        "docs_document_append",
			Description: "Append text content to the end of a document. Simpler than document_update for append-only use cases.",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"document_id": {
						Type:        "string",
						Description: "Document ID",
					},
					"content": {
						Type:        "string",
						Description: "Text or markdown content to append",
					},
				},
				Required: []string{"document_id", "content"},
			},
		},
		{
			Name:        "docs_document_find_replace",
			Description: "Find and replace text in a document. Replaces all occurrences.",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"document_id": {
						Type:        "string",
						Description: "Document ID",
					},
					"find": {
						Type:        "string",
						Description: "Text to find",
					},
					"replacement": {
						Type:        "string",
						Description: "Replacement text",
					},
					"match_case": {
						Type:        "boolean",
						Description: "Case-sensitive search (default false)",
					},
				},
				Required: []string{"document_id", "find", "replacement"},
			},
		},
		{
			Name:        "docs_document_insert_table",
			Description: "Insert a table at the end of a document, optionally populated with data",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"document_id": {
						Type:        "string",
						Description: "Document ID",
					},
					"rows": {
						Type:        "number",
						Description: "Number of rows",
					},
					"columns": {
						Type:        "number",
						Description: "Number of columns",
					},
					"data": {
						Type:        "array",
						Description: "Optional 2D array of strings to populate the table cells",
						Items: &server.Property{
							Type: "array",
						},
					},
				},
				Required: []string{"document_id", "rows", "columns"},
			},
		},
		{
			Name:        "docs_document_get_markdown",
			Description: "Get document content as clean markdown. Converts headings, bold, italic, links, lists, and tables to markdown format.",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"document_id": {
						Type:        "string",
						Description: "Document ID",
					},
				},
				Required: []string{"document_id"},
			},
		},
	}
}

// HandleToolCall handles a tool call for Docs service
func (h *Handler) HandleToolCall(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	switch name {
	case "docs_document_get":
		var args struct {
			DocumentID string `json:"document_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		doc, err := h.client.GetDocument(args.DocumentID)
		if err != nil {
			return nil, err
		}

		// Format document for response
		result := map[string]interface{}{
			"documentId": doc.DocumentId,
			"title":      doc.Title,
		}

		// Extract text content from body
		if doc.Body != nil && doc.Body.Content != nil {
			var textContent string
			for _, element := range doc.Body.Content {
				if element.Paragraph != nil {
					for _, elem := range element.Paragraph.Elements {
						if elem.TextRun != nil {
							textContent += elem.TextRun.Content
						}
					}
				}
			}
			result["content"] = textContent
		}

		return result, nil

	case "docs_document_create":
		var args struct {
			Title string `json:"title"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		doc, err := h.client.CreateDocument(args.Title)
		if err != nil {
			return nil, err
		}

		// Format created document response
		result := map[string]interface{}{
			"documentId": doc.DocumentId,
			"title":      doc.Title,
			"revisionId": doc.RevisionId,
		}
		return result, nil

	case "docs_document_update":
		var args struct {
			DocumentID string `json:"document_id"`
			Content    string `json:"content"`
			Mode       string `json:"mode"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		// Default to append mode
		if args.Mode == "" {
			args.Mode = "append"
		}

		// Update the document
		response, err := h.client.UpdateDocument(args.DocumentID, args.Content, args.Mode)
		if err != nil {
			return nil, err
		}

		// Format response
		result := map[string]interface{}{
			"documentId": response.DocumentId,
			"replies":    len(response.Replies),
			"success":    true,
		}
		return result, nil

	case "docs_document_append":
		var args struct {
			DocumentID string `json:"document_id"`
			Content    string `json:"content"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		response, err := h.client.AppendDocument(args.DocumentID, args.Content)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"documentId": response.DocumentId,
			"replies":    len(response.Replies),
			"success":    true,
		}, nil

	case "docs_document_find_replace":
		var args struct {
			DocumentID  string `json:"document_id"`
			Find        string `json:"find"`
			Replacement string `json:"replacement"`
			MatchCase   bool   `json:"match_case"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		response, err := h.client.FindReplaceDocument(args.DocumentID, args.Find, args.Replacement, args.MatchCase)
		if err != nil {
			return nil, err
		}
		// Extract the number of occurrences replaced from the response
		occurrences := 0
		if len(response.Replies) > 0 && response.Replies[0].ReplaceAllText != nil {
			occurrences = int(response.Replies[0].ReplaceAllText.OccurrencesChanged)
		}
		return map[string]interface{}{
			"documentId":         response.DocumentId,
			"occurrencesChanged": occurrences,
			"success":            true,
		}, nil

	case "docs_document_insert_table":
		var args struct {
			DocumentID string     `json:"document_id"`
			Rows       float64    `json:"rows"`
			Columns    float64    `json:"columns"`
			Data       [][]string `json:"data"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		response, err := h.client.InsertTable(args.DocumentID, int64(args.Rows), int64(args.Columns), args.Data)
		if err != nil {
			return nil, err
		}
		result := map[string]interface{}{
			"documentId": response.DocumentId,
			"rows":       int64(args.Rows),
			"columns":    int64(args.Columns),
			"success":    true,
		}
		return result, nil

	case "docs_document_get_markdown":
		var args struct {
			DocumentID string `json:"document_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		markdown, err := h.client.GetDocumentAsMarkdown(args.DocumentID)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"documentId": args.DocumentID,
			"markdown":   markdown,
		}, nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// GetResources returns the available Docs resources
func (h *Handler) GetResources() []server.Resource {
	return []server.Resource{}
}

// HandleResourceCall handles a resource call for Docs service
func (h *Handler) HandleResourceCall(ctx context.Context, uri string) (interface{}, error) {
	return nil, fmt.Errorf("no resources available for docs")
}
