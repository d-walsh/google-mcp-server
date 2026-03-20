package gmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"go.ngs.io/google-mcp-server/auth"
	"go.ngs.io/google-mcp-server/server"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// MultiAccountClient manages Gmail operations across multiple accounts
type MultiAccountClient struct {
	accountManager *auth.AccountManager
	clients        map[string]*Client
	mu             sync.RWMutex
}

// NewMultiAccountClient creates a new multi-account Gmail client
func NewMultiAccountClient(ctx context.Context, accountManager *auth.AccountManager) (*MultiAccountClient, error) {
	mac := &MultiAccountClient{
		accountManager: accountManager,
		clients:        make(map[string]*Client),
	}

	// Initialize clients for all accounts
	for email, oauthClient := range accountManager.GetAllOAuthClients() {
		service, err := gmail.NewService(ctx, option.WithHTTPClient(oauthClient.GetHTTPClient()))
		if err != nil {
			fmt.Printf("Warning: failed to create gmail service for %s: %v\n", email, err)
			continue
		}
		mac.clients[email] = &Client{service: service}
	}

	return mac, nil
}

// GetClientForContext returns the appropriate client based on context hints
func (mac *MultiAccountClient) GetClientForContext(ctx context.Context, hint string) (*Client, string, error) {
	// First try to get a specific account based on the hint
	account, err := mac.accountManager.GetAccountForContext(ctx, hint)
	if err == nil && account != nil {
		// Always create a fresh service using the current httpClient so that
		// token refreshes (accounts_refresh / accounts_add) take effect immediately.
		service, err := gmail.NewService(ctx, option.WithHTTPClient(account.OAuthClient.GetHTTPClient()))
		if err != nil {
			return nil, "", fmt.Errorf("failed to create gmail service: %w", err)
		}
		return &Client{service: service}, account.Email, nil
	}

	// If context is unclear but only one account exists, use it
	accounts := mac.accountManager.ListAccounts()
	if len(accounts) == 1 {
		account := accounts[0]
		service, err := gmail.NewService(ctx, option.WithHTTPClient(account.OAuthClient.GetHTTPClient()))
		if err != nil {
			return nil, "", fmt.Errorf("failed to create gmail service: %w", err)
		}
		return &Client{service: service}, account.Email, nil
	}

	// Return error with available accounts
	if len(accounts) == 0 {
		return nil, "", fmt.Errorf("no authenticated accounts available")
	}

	var accountList []string
	for _, acc := range accounts {
		accountList = append(accountList, acc.Email)
	}

	return nil, "", fmt.Errorf("please specify account: %s", strings.Join(accountList, ", "))
}

// SearchAcrossAccounts searches for messages across all accounts
func (mac *MultiAccountClient) SearchAcrossAccounts(ctx context.Context, query string, maxResults int64) (map[string][]*gmail.Message, error) {
	results := make(map[string][]*gmail.Message)
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	mac.mu.RLock()
	clients := make(map[string]*Client)
	for email, client := range mac.clients {
		clients[email] = client
	}
	mac.mu.RUnlock()

	for email, client := range clients {
		wg.Add(1)
		go func(email string, client *Client) {
			defer wg.Done()

			messages, err := client.ListMessages(query, maxResults)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("%s: %w", email, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			results[email] = messages
			mu.Unlock()
		}(email, client)
	}

	wg.Wait()

	// If all searches failed, return the first error
	if len(errors) == len(clients) && len(errors) > 0 {
		return nil, errors[0]
	}

	return results, nil
}

// MultiAccountHandler handles Gmail operations with multi-account support
type MultiAccountHandler struct {
	multiClient *MultiAccountClient
	client      *Client // Default client for backward compatibility
}

// NewMultiAccountHandler creates a new handler with multi-account support
func NewMultiAccountHandler(accountManager *auth.AccountManager, defaultClient *Client) *MultiAccountHandler {
	ctx := context.Background()
	multiClient, err := NewMultiAccountClient(ctx, accountManager)
	if err != nil {
		// Log error but continue with limited functionality
		fmt.Printf("Warning: failed to initialize multi-account client: %v\n", err)
		multiClient = &MultiAccountClient{
			accountManager: accountManager,
			clients:        make(map[string]*Client),
		}
	}

	return &MultiAccountHandler{
		multiClient: multiClient,
		client:      defaultClient,
	}
}

// GetTools returns the available Gmail tools with multi-account support
func (h *MultiAccountHandler) GetTools() []server.Tool {
	accountProp := server.Property{
		Type:        "string",
		Description: "Email address of the account to use (optional)",
	}

	return []server.Tool{
		{
			Name:        "gmail_messages_list",
			Description: "List email messages with snippet, subject, from, and date for each message",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"query": {
						Type:        "string",
						Description: "Search query (e.g., 'from:user@example.com')",
					},
					"max_results": {
						Type:        "number",
						Description: "Maximum number of results",
					},
					"account": accountProp,
				},
			},
		},
		{
			Name:        "gmail_message_get",
			Description: "Get email message details",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"message_id": {
						Type:        "string",
						Description: "Message ID",
					},
					"account": accountProp,
				},
				Required: []string{"message_id"},
			},
		},
		{
			Name:        "gmail_messages_list_all_accounts",
			Description: "List messages from all authenticated accounts",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"query": {
						Type:        "string",
						Description: "Search query (e.g., 'is:unread')",
					},
					"max_results": {
						Type:        "number",
						Description: "Maximum number of results per account",
					},
				},
			},
		},
		{
			Name:        "gmail_send",
			Description: "Compose and send a new email",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"to": {
						Type:        "string",
						Description: "Recipient email address(es), comma-separated",
					},
					"subject": {
						Type:        "string",
						Description: "Email subject",
					},
					"body": {
						Type:        "string",
						Description: "Email body content",
					},
					"cc": {
						Type:        "string",
						Description: "CC recipients, comma-separated (optional)",
					},
					"bcc": {
						Type:        "string",
						Description: "BCC recipients, comma-separated (optional)",
					},
					"content_type": {
						Type:        "string",
						Description: "Content type: 'text/plain' (default) or 'text/html'",
					},
					"account": accountProp,
				},
				Required: []string{"to", "subject", "body"},
			},
		},
		{
			Name:        "gmail_reply",
			Description: "Reply to an existing email thread",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"message_id": {
						Type:        "string",
						Description: "Message ID to reply to",
					},
					"body": {
						Type:        "string",
						Description: "Reply body content",
					},
					"reply_all": {
						Type:        "boolean",
						Description: "Reply to all recipients (default: true)",
					},
					"content_type": {
						Type:        "string",
						Description: "Content type: 'text/plain' (default) or 'text/html'",
					},
					"account": accountProp,
				},
				Required: []string{"message_id", "body"},
			},
		},
		{
			Name:        "gmail_download_attachment",
			Description: "Download an email attachment by message ID and attachment ID",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"message_id": {
						Type:        "string",
						Description: "Message ID containing the attachment",
					},
					"attachment_id": {
						Type:        "string",
						Description: "Attachment ID to download",
					},
					"account": accountProp,
				},
				Required: []string{"message_id", "attachment_id"},
			},
		},
		{
			Name:        "gmail_labels_list",
			Description: "List all Gmail labels",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"account": accountProp,
				},
			},
		},
		{
			Name:        "gmail_label_add",
			Description: "Add label(s) to a message",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"message_id": {
						Type:        "string",
						Description: "Message ID",
					},
					"label_ids": {
						Type:        "array",
						Description: "Label IDs to add",
						Items: &server.Property{
							Type: "string",
						},
					},
					"account": accountProp,
				},
				Required: []string{"message_id", "label_ids"},
			},
		},
		{
			Name:        "gmail_label_remove",
			Description: "Remove label(s) from a message",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"message_id": {
						Type:        "string",
						Description: "Message ID",
					},
					"label_ids": {
						Type:        "array",
						Description: "Label IDs to remove",
						Items: &server.Property{
							Type: "string",
						},
					},
					"account": accountProp,
				},
				Required: []string{"message_id", "label_ids"},
			},
		},
		{
			Name:        "gmail_draft_create",
			Description: "Create a new email draft",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"to": {
						Type:        "string",
						Description: "Recipient email address(es), comma-separated",
					},
					"subject": {
						Type:        "string",
						Description: "Email subject",
					},
					"body": {
						Type:        "string",
						Description: "Email body content",
					},
					"cc": {
						Type:        "string",
						Description: "CC recipients, comma-separated (optional)",
					},
					"content_type": {
						Type:        "string",
						Description: "Content type: 'text/plain' (default) or 'text/html'",
					},
					"thread_id": {
						Type:        "string",
						Description: "Thread ID for reply drafts (optional)",
					},
					"account": accountProp,
				},
			},
		},
		{
			Name:        "gmail_trash",
			Description: "Move a message to trash",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"message_id": {
						Type:        "string",
						Description: "Message ID to trash",
					},
					"account": accountProp,
				},
				Required: []string{"message_id"},
			},
		},
		{
			Name:        "gmail_untrash",
			Description: "Remove a message from trash",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"message_id": {
						Type:        "string",
						Description: "Message ID to untrash",
					},
					"account": accountProp,
				},
				Required: []string{"message_id"},
			},
		},
		{
			Name:        "gmail_mark_read",
			Description: "Mark a message as read",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"message_id": {
						Type:        "string",
						Description: "Message ID to mark as read",
					},
					"account": accountProp,
				},
				Required: []string{"message_id"},
			},
		},
		{
			Name:        "gmail_label_create",
			Description: "Create a new Gmail label",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"name": {
						Type:        "string",
						Description: "Label name",
					},
					"account": accountProp,
				},
				Required: []string{"name"},
			},
		},
	}
}

// getClientOrDefault resolves the client for a given account hint, falling back to default
func (h *MultiAccountHandler) getClientOrDefault(ctx context.Context, account string) (*Client, string, error) {
	client, accountUsed, err := h.multiClient.GetClientForContext(ctx, account)
	if err != nil {
		if h.client != nil {
			return h.client, "default", nil
		}
		return nil, "", err
	}
	return client, accountUsed, nil
}

// HandleToolCall handles a tool call for Gmail service with multi-account support
func (h *MultiAccountHandler) HandleToolCall(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	switch name {
	case "gmail_messages_list":
		var args struct {
			Query      string  `json:"query"`
			MaxResults float64 `json:"max_results"`
			Account    string  `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		client, accountUsed, err := h.getClientOrDefault(ctx, args.Account)
		if err != nil {
			return nil, err
		}

		messages, err := client.ListMessages(args.Query, int64(args.MaxResults))
		if err != nil {
			return nil, err
		}

		// Enrich each message with metadata (snippet, subject, from, date, labels)
		messageList := make([]map[string]interface{}, 0, len(messages))
		for _, msg := range messages {
			entry := map[string]interface{}{
				"id":       msg.Id,
				"threadId": msg.ThreadId,
			}

			// Fetch metadata for richer listing
			meta, metaErr := client.GetMessageMetadata(msg.Id)
			if metaErr == nil && meta != nil {
				entry["snippet"] = meta.Snippet
				entry["labelIds"] = meta.LabelIds
				if meta.Payload != nil {
					for _, h := range meta.Payload.Headers {
						switch strings.ToLower(h.Name) {
						case "subject":
							entry["subject"] = h.Value
						case "from":
							entry["from"] = h.Value
						case "date":
							entry["date"] = h.Value
						}
					}
				}
			}

			messageList = append(messageList, entry)
		}
		return map[string]interface{}{
			"messages": messageList,
			"account":  accountUsed,
		}, nil

	case "gmail_message_get":
		var args struct {
			MessageID string `json:"message_id"`
			Account   string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		client, accountUsed, err := h.getClientOrDefault(ctx, args.Account)
		if err != nil {
			return nil, err
		}

		message, err := client.GetMessage(args.MessageID)
		if err != nil {
			return nil, err
		}

		result := map[string]interface{}{
			"id":           message.Id,
			"threadId":     message.ThreadId,
			"labelIds":     message.LabelIds,
			"snippet":      message.Snippet,
			"historyId":    message.HistoryId,
			"internalDate": message.InternalDate,
			"sizeEstimate": message.SizeEstimate,
			"account":      accountUsed,
		}

		if message.Payload != nil && message.Payload.Headers != nil {
			headers := make(map[string]string)
			for _, header := range message.Payload.Headers {
				headers[header.Name] = header.Value
			}
			result["headers"] = headers

			if message.Payload.Body != nil && message.Payload.Body.Data != "" {
				result["body"] = message.Payload.Body.Data
			}

			// Include attachment info from parts
			if message.Payload.Parts != nil {
				var attachments []map[string]interface{}
				for _, part := range message.Payload.Parts {
					if part.Filename != "" && part.Body != nil && part.Body.AttachmentId != "" {
						attachments = append(attachments, map[string]interface{}{
							"filename":     part.Filename,
							"mimeType":     part.MimeType,
							"size":         part.Body.Size,
							"attachmentId": part.Body.AttachmentId,
						})
					}
				}
				if len(attachments) > 0 {
					result["attachments"] = attachments
				}
			}
		}

		return result, nil

	case "gmail_messages_list_all_accounts":
		var args struct {
			Query      string  `json:"query"`
			MaxResults float64 `json:"max_results"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		query := args.Query
		if query == "" {
			query = "in:inbox"
		}

		results, err := h.multiClient.SearchAcrossAccounts(ctx, query, int64(args.MaxResults))
		if err != nil {
			return nil, err
		}

		formattedResults := make(map[string]interface{})
		totalMessages := 0
		for email, messages := range results {
			messageList := make([]map[string]interface{}, len(messages))
			for i, msg := range messages {
				messageList[i] = map[string]interface{}{
					"id":       msg.Id,
					"threadId": msg.ThreadId,
				}
			}
			formattedResults[email] = map[string]interface{}{
				"messages": messageList,
				"count":    len(messages),
			}
			totalMessages += len(messages)
		}

		return map[string]interface{}{
			"accounts":      formattedResults,
			"total_count":   totalMessages,
			"account_count": len(results),
		}, nil

	case "gmail_send":
		var args struct {
			To          string `json:"to"`
			Subject     string `json:"subject"`
			Body        string `json:"body"`
			Cc          string `json:"cc"`
			Bcc         string `json:"bcc"`
			ContentType string `json:"content_type"`
			Account     string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		client, accountUsed, err := h.getClientOrDefault(ctx, args.Account)
		if err != nil {
			return nil, err
		}

		sent, err := client.SendMessage(args.To, args.Subject, args.Body, args.Cc, args.Bcc, args.ContentType)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"id":       sent.Id,
			"threadId": sent.ThreadId,
			"labelIds": sent.LabelIds,
			"account":  accountUsed,
		}, nil

	case "gmail_reply":
		var args struct {
			MessageID   string `json:"message_id"`
			Body        string `json:"body"`
			ReplyAll    *bool  `json:"reply_all"`
			ContentType string `json:"content_type"`
			Account     string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		client, accountUsed, err := h.getClientOrDefault(ctx, args.Account)
		if err != nil {
			return nil, err
		}

		// Default reply_all to true
		replyAll := true
		if args.ReplyAll != nil {
			replyAll = *args.ReplyAll
		}

		sent, err := client.ReplyToMessage(args.MessageID, args.Body, replyAll, args.ContentType)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"id":       sent.Id,
			"threadId": sent.ThreadId,
			"labelIds": sent.LabelIds,
			"account":  accountUsed,
		}, nil

	case "gmail_download_attachment":
		var args struct {
			MessageID    string `json:"message_id"`
			AttachmentID string `json:"attachment_id"`
			Account      string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		client, accountUsed, err := h.getClientOrDefault(ctx, args.Account)
		if err != nil {
			return nil, err
		}

		attachment, err := client.GetAttachment(args.MessageID, args.AttachmentID)
		if err != nil {
			return nil, err
		}

		// Also get the message to find the filename
		message, msgErr := client.GetMessage(args.MessageID)
		filename := ""
		mimeType := ""
		if msgErr == nil && message.Payload != nil {
			for _, part := range message.Payload.Parts {
				if part.Body != nil && part.Body.AttachmentId == args.AttachmentID {
					filename = part.Filename
					mimeType = part.MimeType
					break
				}
			}
		}

		// The attachment data from Gmail API is base64url-encoded
		// Decode it and re-encode as standard base64 for the response
		decoded, decErr := base64.URLEncoding.DecodeString(attachment.Data)
		var data string
		if decErr == nil {
			data = base64.StdEncoding.EncodeToString(decoded)
		} else {
			// Fall back to raw data if decode fails
			data = attachment.Data
		}

		return map[string]interface{}{
			"data":     data,
			"size":     attachment.Size,
			"filename": filename,
			"mimeType": mimeType,
			"account":  accountUsed,
		}, nil

	case "gmail_labels_list":
		var args struct {
			Account string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		client, accountUsed, err := h.getClientOrDefault(ctx, args.Account)
		if err != nil {
			return nil, err
		}

		labels, err := client.ListLabels()
		if err != nil {
			return nil, err
		}

		labelList := make([]map[string]interface{}, len(labels))
		for i, label := range labels {
			labelList[i] = map[string]interface{}{
				"id":   label.Id,
				"name": label.Name,
				"type": label.Type,
			}
		}

		return map[string]interface{}{
			"labels":  labelList,
			"account": accountUsed,
		}, nil

	case "gmail_label_add":
		var args struct {
			MessageID string   `json:"message_id"`
			LabelIDs  []string `json:"label_ids"`
			Account   string   `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		client, accountUsed, err := h.getClientOrDefault(ctx, args.Account)
		if err != nil {
			return nil, err
		}

		message, err := client.ModifyMessageLabels(args.MessageID, args.LabelIDs, nil)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"id":       message.Id,
			"labelIds": message.LabelIds,
			"account":  accountUsed,
		}, nil

	case "gmail_label_remove":
		var args struct {
			MessageID string   `json:"message_id"`
			LabelIDs  []string `json:"label_ids"`
			Account   string   `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		client, accountUsed, err := h.getClientOrDefault(ctx, args.Account)
		if err != nil {
			return nil, err
		}

		message, err := client.ModifyMessageLabels(args.MessageID, nil, args.LabelIDs)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"id":       message.Id,
			"labelIds": message.LabelIds,
			"account":  accountUsed,
		}, nil

	case "gmail_draft_create":
		var args struct {
			To          string `json:"to"`
			Subject     string `json:"subject"`
			Body        string `json:"body"`
			Cc          string `json:"cc"`
			ContentType string `json:"content_type"`
			ThreadID    string `json:"thread_id"`
			Account     string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		client, accountUsed, err := h.getClientOrDefault(ctx, args.Account)
		if err != nil {
			return nil, err
		}

		draft, err := client.CreateDraft(args.To, args.Subject, args.Body, args.Cc, args.ContentType, args.ThreadID)
		if err != nil {
			return nil, err
		}

		result := map[string]interface{}{
			"id":      draft.Id,
			"account": accountUsed,
		}
		if draft.Message != nil {
			result["messageId"] = draft.Message.Id
			result["threadId"] = draft.Message.ThreadId
		}

		return result, nil

	case "gmail_trash":
		var args struct {
			MessageID string `json:"message_id"`
			Account   string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		client, accountUsed, err := h.getClientOrDefault(ctx, args.Account)
		if err != nil {
			return nil, err
		}

		message, err := client.TrashMessage(args.MessageID)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"id":       message.Id,
			"labelIds": message.LabelIds,
			"account":  accountUsed,
		}, nil

	case "gmail_untrash":
		var args struct {
			MessageID string `json:"message_id"`
			Account   string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		client, accountUsed, err := h.getClientOrDefault(ctx, args.Account)
		if err != nil {
			return nil, err
		}

		message, err := client.UntrashMessage(args.MessageID)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"id":       message.Id,
			"labelIds": message.LabelIds,
			"account":  accountUsed,
		}, nil

	case "gmail_mark_read":
		var args struct {
			MessageID string `json:"message_id"`
			Account   string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		client, accountUsed, err := h.getClientOrDefault(ctx, args.Account)
		if err != nil {
			return nil, err
		}

		message, err := client.ModifyMessageLabels(args.MessageID, nil, []string{"UNREAD"})
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"id":       message.Id,
			"labelIds": message.LabelIds,
			"account":  accountUsed,
		}, nil

	case "gmail_label_create":
		var args struct {
			Name    string `json:"name"`
			Account string `json:"account"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		client, accountUsed, err := h.getClientOrDefault(ctx, args.Account)
		if err != nil {
			return nil, err
		}

		label, err := client.CreateLabel(args.Name)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"id":      label.Id,
			"name":    label.Name,
			"account": accountUsed,
		}, nil

	default:
		if h.client != nil {
			handler := &Handler{client: h.client}
			return handler.HandleToolCall(ctx, name, arguments)
		}
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// GetResources returns the available Gmail resources
func (h *MultiAccountHandler) GetResources() []server.Resource {
	return []server.Resource{
		{
			URI:         "gmail://inbox",
			Name:        "Inbox",
			Description: "Gmail inbox messages",
			MimeType:    "application/json",
		},
	}
}

// HandleResourceCall handles a resource call for Gmail service
func (h *MultiAccountHandler) HandleResourceCall(ctx context.Context, uri string) (interface{}, error) {
	if uri == "gmail://inbox" {
		// List inbox messages from all accounts
		results, err := h.multiClient.SearchAcrossAccounts(ctx, "in:inbox", 20)
		if err != nil {
			// Fall back to default client if available
			if h.client != nil {
				messages, err := h.client.ListMessages("in:inbox", 20)
				if err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"messages": messages,
					"count":    len(messages),
				}, nil
			}
			return nil, err
		}

		// Format results
		formattedResults := make(map[string]interface{})
		totalMessages := 0
		for email, messages := range results {
			formattedResults[email] = map[string]interface{}{
				"messages": messages,
				"count":    len(messages),
			}
			totalMessages += len(messages)
		}

		return map[string]interface{}{
			"accounts":    formattedResults,
			"total_count": totalMessages,
		}, nil
	}
	return nil, fmt.Errorf("unknown resource: %s", uri)
}
