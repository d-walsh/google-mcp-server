package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"go.ngs.io/google-mcp-server/auth"
	"google.golang.org/api/gmail/v1"
)

// Client wraps the Google Gmail API client
type Client struct {
	service *gmail.Service
}

// NewClient creates a new Gmail client
func NewClient(ctx context.Context, oauth *auth.OAuthClient) (*Client, error) {
	service, err := gmail.NewService(ctx, oauth.GetClientOption())
	if err != nil {
		return nil, fmt.Errorf("failed to create gmail service: %w", err)
	}

	return &Client{
		service: service,
	}, nil
}

// ListMessages lists messages
func (c *Client) ListMessages(query string, maxResults int64) ([]*gmail.Message, error) {
	call := c.service.Users.Messages.List("me")
	if query != "" {
		call = call.Q(query)
	}
	if maxResults > 0 {
		call = call.MaxResults(maxResults)
	}

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	return response.Messages, nil
}

// GetMessage gets a message by ID
func (c *Client) GetMessage(messageID string) (*gmail.Message, error) {
	message, err := c.service.Users.Messages.Get("me", messageID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}
	return message, nil
}

// GetMessageMetadata gets a message with only metadata (headers), no body
func (c *Client) GetMessageMetadata(messageID string) (*gmail.Message, error) {
	message, err := c.service.Users.Messages.Get("me", messageID).Format("metadata").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message metadata: %w", err)
	}
	return message, nil
}

// buildRawMessage constructs an RFC 2822 message and returns it base64url-encoded
func buildRawMessage(to, cc, bcc, subject, body, contentType string, extraHeaders map[string]string) string {
	var msg strings.Builder

	// Write headers
	if to != "" {
		msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	}
	if cc != "" {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", cc))
	}
	if bcc != "" {
		msg.WriteString(fmt.Sprintf("Bcc: %s\r\n", bcc))
	}
	if subject != "" {
		msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	}
	for k, v := range extraHeaders {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	if contentType == "" {
		contentType = "text/plain"
	}
	msg.WriteString(fmt.Sprintf("Content-Type: %s; charset=\"UTF-8\"\r\n", contentType))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	return base64.URLEncoding.EncodeToString([]byte(msg.String()))
}

// SendMessage sends a new email
func (c *Client) SendMessage(to, subject, body, cc, bcc, contentType string) (*gmail.Message, error) {
	raw := buildRawMessage(to, cc, bcc, subject, body, contentType, nil)

	msg := &gmail.Message{
		Raw: raw,
	}

	sent, err := c.service.Users.Messages.Send("me", msg).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}
	return sent, nil
}

// getHeader returns the value of a header from a message
func getHeader(msg *gmail.Message, name string) string {
	if msg.Payload == nil {
		return ""
	}
	for _, h := range msg.Payload.Headers {
		if strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
}

// ReplyToMessage replies to an existing message
func (c *Client) ReplyToMessage(messageID, body string, replyAll bool, contentType string) (*gmail.Message, error) {
	// Fetch original message to get threading headers
	original, err := c.GetMessage(messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original message: %w", err)
	}

	threadID := original.ThreadId
	origMessageID := getHeader(original, "Message-Id")
	origReferences := getHeader(original, "References")
	origSubject := getHeader(original, "Subject")
	origFrom := getHeader(original, "From")
	origTo := getHeader(original, "To")
	origCc := getHeader(original, "Cc")

	// Build subject with Re: prefix
	subject := origSubject
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}

	// Build References header
	references := origReferences
	if origMessageID != "" {
		if references != "" {
			references = references + " " + origMessageID
		} else {
			references = origMessageID
		}
	}

	// Build recipient list
	to := origFrom // Always reply to sender
	cc := ""

	if replyAll {
		// Get sender's own email to exclude from recipients
		profile, profileErr := c.service.Users.GetProfile("me").Do()
		myEmail := ""
		if profileErr == nil {
			myEmail = strings.ToLower(profile.EmailAddress)
		}

		// Collect all To recipients minus ourselves
		allTo := parseAddresses(origTo)
		for _, addr := range allTo {
			if strings.ToLower(addr) != myEmail && !strings.Contains(strings.ToLower(to), strings.ToLower(addr)) {
				to = to + ", " + addr
			}
		}

		// CC stays as-is minus ourselves
		allCc := parseAddresses(origCc)
		var ccList []string
		for _, addr := range allCc {
			if strings.ToLower(addr) != myEmail {
				ccList = append(ccList, addr)
			}
		}
		if len(ccList) > 0 {
			cc = strings.Join(ccList, ", ")
		}
	}

	extraHeaders := map[string]string{}
	if origMessageID != "" {
		extraHeaders["In-Reply-To"] = origMessageID
	}
	if references != "" {
		extraHeaders["References"] = references
	}

	raw := buildRawMessage(to, cc, "", subject, body, contentType, extraHeaders)

	msg := &gmail.Message{
		Raw:      raw,
		ThreadId: threadID,
	}

	sent, err := c.service.Users.Messages.Send("me", msg).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to send reply: %w", err)
	}
	return sent, nil
}

// parseAddresses splits a comma-separated address list into individual addresses
func parseAddresses(addrList string) []string {
	if addrList == "" {
		return nil
	}
	parts := strings.Split(addrList, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// GetAttachment downloads an attachment by message ID and attachment ID
func (c *Client) GetAttachment(messageID, attachmentID string) (*gmail.MessagePartBody, error) {
	attachment, err := c.service.Users.Messages.Attachments.Get("me", messageID, attachmentID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}
	return attachment, nil
}

// ListLabels lists all labels for the user
func (c *Client) ListLabels() ([]*gmail.Label, error) {
	response, err := c.service.Users.Labels.List("me").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list labels: %w", err)
	}
	return response.Labels, nil
}

// ModifyMessageLabels adds or removes labels from a message
func (c *Client) ModifyMessageLabels(messageID string, addLabelIDs, removeLabelIDs []string) (*gmail.Message, error) {
	modReq := &gmail.ModifyMessageRequest{
		AddLabelIds:    addLabelIDs,
		RemoveLabelIds: removeLabelIDs,
	}
	message, err := c.service.Users.Messages.Modify("me", messageID, modReq).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to modify message labels: %w", err)
	}
	return message, nil
}

// TrashMessage moves a message to trash
func (c *Client) TrashMessage(messageID string) (*gmail.Message, error) {
	message, err := c.service.Users.Messages.Trash("me", messageID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to trash message: %w", err)
	}
	return message, nil
}

// UntrashMessage removes a message from trash
func (c *Client) UntrashMessage(messageID string) (*gmail.Message, error) {
	message, err := c.service.Users.Messages.Untrash("me", messageID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to untrash message: %w", err)
	}
	return message, nil
}

// CreateDraft creates a new draft
func (c *Client) CreateDraft(to, subject, body, cc, contentType, threadID string) (*gmail.Draft, error) {
	raw := buildRawMessage(to, cc, "", subject, body, contentType, nil)

	msg := &gmail.Message{
		Raw: raw,
	}
	if threadID != "" {
		msg.ThreadId = threadID
	}

	draft := &gmail.Draft{
		Message: msg,
	}

	created, err := c.service.Users.Drafts.Create("me", draft).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create draft: %w", err)
	}
	return created, nil
}

// CreateLabel creates a new label
func (c *Client) CreateLabel(name string) (*gmail.Label, error) {
	label := &gmail.Label{
		Name:                name,
		LabelListVisibility: "labelShow",
		MessageListVisibility: "show",
	}
	created, err := c.service.Users.Labels.Create("me", label).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create label: %w", err)
	}
	return created, nil
}
