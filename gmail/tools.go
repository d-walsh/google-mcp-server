package gmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"go.ngs.io/google-mcp-server/server"
)

// Handler implements the ServiceHandler interface for Gmail
type Handler struct {
	client *Client
}

// NewHandler creates a new Gmail handler
func NewHandler(client *Client) *Handler {
	return &Handler{client: client}
}

// GetTools returns the available Gmail tools
func (h *Handler) GetTools() []server.Tool {
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
				},
				Required: []string{"message_id"},
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
				},
				Required: []string{"message_id", "attachment_id"},
			},
		},
		{
			Name:        "gmail_labels_list",
			Description: "List all Gmail labels",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{},
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
				},
				Required: []string{"name"},
			},
		},
	}
}

// HandleToolCall handles a tool call for Gmail service
func (h *Handler) HandleToolCall(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	switch name {
	case "gmail_messages_list":
		var args struct {
			Query      string  `json:"query"`
			MaxResults float64 `json:"max_results"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		messages, err := h.client.ListMessages(args.Query, int64(args.MaxResults))
		if err != nil {
			return nil, err
		}

		// Enrich each message with metadata
		messageList := make([]map[string]interface{}, 0, len(messages))
		for _, msg := range messages {
			entry := map[string]interface{}{
				"id":       msg.Id,
				"threadId": msg.ThreadId,
			}

			meta, metaErr := h.client.GetMessageMetadata(msg.Id)
			if metaErr == nil && meta != nil {
				entry["snippet"] = meta.Snippet
				entry["labelIds"] = meta.LabelIds
				if meta.Payload != nil {
					for _, hdr := range meta.Payload.Headers {
						switch strings.ToLower(hdr.Name) {
						case "subject":
							entry["subject"] = hdr.Value
						case "from":
							entry["from"] = hdr.Value
						case "date":
							entry["date"] = hdr.Value
						}
					}
				}
			}

			messageList = append(messageList, entry)
		}
		return map[string]interface{}{
			"messages": messageList,
		}, nil

	case "gmail_message_get":
		var args struct {
			MessageID string `json:"message_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		message, err := h.client.GetMessage(args.MessageID)
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

			// Include attachment info
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

	case "gmail_send":
		var args struct {
			To          string `json:"to"`
			Subject     string `json:"subject"`
			Body        string `json:"body"`
			Cc          string `json:"cc"`
			Bcc         string `json:"bcc"`
			ContentType string `json:"content_type"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		sent, err := h.client.SendMessage(args.To, args.Subject, args.Body, args.Cc, args.Bcc, args.ContentType)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"id":       sent.Id,
			"threadId": sent.ThreadId,
			"labelIds": sent.LabelIds,
		}, nil

	case "gmail_reply":
		var args struct {
			MessageID   string `json:"message_id"`
			Body        string `json:"body"`
			ReplyAll    *bool  `json:"reply_all"`
			ContentType string `json:"content_type"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		replyAll := true
		if args.ReplyAll != nil {
			replyAll = *args.ReplyAll
		}

		sent, err := h.client.ReplyToMessage(args.MessageID, args.Body, replyAll, args.ContentType)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"id":       sent.Id,
			"threadId": sent.ThreadId,
			"labelIds": sent.LabelIds,
		}, nil

	case "gmail_download_attachment":
		var args struct {
			MessageID    string `json:"message_id"`
			AttachmentID string `json:"attachment_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		attachment, err := h.client.GetAttachment(args.MessageID, args.AttachmentID)
		if err != nil {
			return nil, err
		}

		// Get filename from message
		message, msgErr := h.client.GetMessage(args.MessageID)
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

		decoded, decErr := base64.URLEncoding.DecodeString(attachment.Data)
		var data string
		if decErr == nil {
			data = base64.StdEncoding.EncodeToString(decoded)
		} else {
			data = attachment.Data
		}

		return map[string]interface{}{
			"data":     data,
			"size":     attachment.Size,
			"filename": filename,
			"mimeType": mimeType,
		}, nil

	case "gmail_labels_list":
		labels, err := h.client.ListLabels()
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
			"labels": labelList,
		}, nil

	case "gmail_label_add":
		var args struct {
			MessageID string   `json:"message_id"`
			LabelIDs  []string `json:"label_ids"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		message, err := h.client.ModifyMessageLabels(args.MessageID, args.LabelIDs, nil)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"id":       message.Id,
			"labelIds": message.LabelIds,
		}, nil

	case "gmail_label_remove":
		var args struct {
			MessageID string   `json:"message_id"`
			LabelIDs  []string `json:"label_ids"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		message, err := h.client.ModifyMessageLabels(args.MessageID, nil, args.LabelIDs)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"id":       message.Id,
			"labelIds": message.LabelIds,
		}, nil

	case "gmail_draft_create":
		var args struct {
			To          string `json:"to"`
			Subject     string `json:"subject"`
			Body        string `json:"body"`
			Cc          string `json:"cc"`
			ContentType string `json:"content_type"`
			ThreadID    string `json:"thread_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		draft, err := h.client.CreateDraft(args.To, args.Subject, args.Body, args.Cc, args.ContentType, args.ThreadID)
		if err != nil {
			return nil, err
		}

		result := map[string]interface{}{
			"id": draft.Id,
		}
		if draft.Message != nil {
			result["messageId"] = draft.Message.Id
			result["threadId"] = draft.Message.ThreadId
		}

		return result, nil

	case "gmail_trash":
		var args struct {
			MessageID string `json:"message_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		message, err := h.client.TrashMessage(args.MessageID)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"id":       message.Id,
			"labelIds": message.LabelIds,
		}, nil

	case "gmail_untrash":
		var args struct {
			MessageID string `json:"message_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		message, err := h.client.UntrashMessage(args.MessageID)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"id":       message.Id,
			"labelIds": message.LabelIds,
		}, nil

	case "gmail_mark_read":
		var args struct {
			MessageID string `json:"message_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		message, err := h.client.ModifyMessageLabels(args.MessageID, nil, []string{"UNREAD"})
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"id":       message.Id,
			"labelIds": message.LabelIds,
		}, nil

	case "gmail_label_create":
		var args struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		label, err := h.client.CreateLabel(args.Name)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"id":   label.Id,
			"name": label.Name,
		}, nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// GetResources returns the available Gmail resources
func (h *Handler) GetResources() []server.Resource {
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
func (h *Handler) HandleResourceCall(ctx context.Context, uri string) (interface{}, error) {
	if uri == "gmail://inbox" {
		messages, err := h.client.ListMessages("in:inbox", 20)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"messages": messages,
			"count":    len(messages),
		}, nil
	}
	return nil, fmt.Errorf("unknown resource: %s", uri)
}
