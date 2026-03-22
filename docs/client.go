package docs

import (
	"context"
	"fmt"

	"go.ngs.io/google-mcp-server/auth"
	"google.golang.org/api/docs/v1"
)

// Client wraps the Google Docs API client
type Client struct {
	service *docs.Service
}

// NewClient creates a new Docs client
func NewClient(ctx context.Context, oauth *auth.OAuthClient) (*Client, error) {
	service, err := docs.NewService(ctx, oauth.GetClientOption())
	if err != nil {
		return nil, fmt.Errorf("failed to create docs service: %w", err)
	}

	return &Client{
		service: service,
	}, nil
}

// GetDocument gets a document by ID
func (c *Client) GetDocument(documentID string) (*docs.Document, error) {
	doc, err := c.service.Documents.Get(documentID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	return doc, nil
}

// CreateDocument creates a new document
func (c *Client) CreateDocument(title string) (*docs.Document, error) {
	doc := &docs.Document{
		Title: title,
	}
	created, err := c.service.Documents.Create(doc).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}
	return created, nil
}

// BatchUpdate performs batch updates on a document
func (c *Client) BatchUpdate(documentID string, requests []*docs.Request) (*docs.BatchUpdateDocumentResponse, error) {
	batchUpdate := &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}
	response, err := c.service.Documents.BatchUpdate(documentID, batchUpdate).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to batch update: %w", err)
	}
	return response, nil
}

// UpdateDocument updates a document's content
func (c *Client) UpdateDocument(documentID string, content string, mode string) (*docs.BatchUpdateDocumentResponse, error) {
	var requests []*docs.Request

	if mode == "replace" {
		// First, get the document to find the end index
		doc, err := c.GetDocument(documentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get document for replacement: %w", err)
		}

		// Find the end index of the document content
		endIndex := int64(1) // Default to 1 if document is empty
		if doc.Body != nil && doc.Body.Content != nil && len(doc.Body.Content) > 0 {
			lastElement := doc.Body.Content[len(doc.Body.Content)-1]
			if lastElement.EndIndex > 0 {
				endIndex = lastElement.EndIndex - 1 // Subtract 1 to avoid the final newline
			}
		}

		// Delete existing content (if any)
		if endIndex > 1 {
			requests = append(requests, &docs.Request{
				DeleteContentRange: &docs.DeleteContentRangeRequest{
					Range: &docs.Range{
						StartIndex: 1,
						EndIndex:   endIndex,
					},
				},
			})
		}

		// Insert new content at the beginning
		requests = append(requests, &docs.Request{
			InsertText: &docs.InsertTextRequest{
				Location: &docs.Location{
					Index: 1,
				},
				Text: content,
			},
		})
	} else {
		// Append mode: get the document to find where to append
		doc, err := c.GetDocument(documentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get document for appending: %w", err)
		}

		// Find the end index to append content
		appendIndex := int64(1) // Default to 1 if document is empty
		if doc.Body != nil && doc.Body.Content != nil && len(doc.Body.Content) > 0 {
			lastElement := doc.Body.Content[len(doc.Body.Content)-1]
			if lastElement.EndIndex > 0 {
				appendIndex = lastElement.EndIndex - 1 // Insert before the final newline
			}
		}

		// Insert text at the end
		requests = append(requests, &docs.Request{
			InsertText: &docs.InsertTextRequest{
				Location: &docs.Location{
					Index: appendIndex,
				},
				Text: content,
			},
		})
	}

	return c.BatchUpdate(documentID, requests)
}

// AppendDocument appends text content to the end of a document
func (c *Client) AppendDocument(documentID string, content string) (*docs.BatchUpdateDocumentResponse, error) {
	// Get the document to find the end index
	doc, err := c.GetDocument(documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document for appending: %w", err)
	}

	// Find the end index to append content
	appendIndex := int64(1)
	if doc.Body != nil && doc.Body.Content != nil && len(doc.Body.Content) > 0 {
		lastElement := doc.Body.Content[len(doc.Body.Content)-1]
		if lastElement.EndIndex > 0 {
			appendIndex = lastElement.EndIndex - 1
		}
	}

	requests := []*docs.Request{
		{
			InsertText: &docs.InsertTextRequest{
				Location: &docs.Location{
					Index: appendIndex,
				},
				Text: content,
			},
		},
	}

	return c.BatchUpdate(documentID, requests)
}

// FindReplaceDocument finds and replaces text in a document
func (c *Client) FindReplaceDocument(documentID string, find string, replacement string, matchCase bool) (*docs.BatchUpdateDocumentResponse, error) {
	requests := []*docs.Request{
		{
			ReplaceAllText: &docs.ReplaceAllTextRequest{
				ContainsText: &docs.SubstringMatchCriteria{
					Text:      find,
					MatchCase: matchCase,
				},
				ReplaceText: replacement,
			},
		},
	}

	return c.BatchUpdate(documentID, requests)
}

// InsertTable inserts a table into a document at the end, optionally with data
func (c *Client) InsertTable(documentID string, rows int64, columns int64, data [][]string) (*docs.BatchUpdateDocumentResponse, error) {
	// Get the document to find the end index
	doc, err := c.GetDocument(documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document for table insertion: %w", err)
	}

	// Find the end index
	insertIndex := int64(1)
	if doc.Body != nil && doc.Body.Content != nil && len(doc.Body.Content) > 0 {
		lastElement := doc.Body.Content[len(doc.Body.Content)-1]
		if lastElement.EndIndex > 0 {
			insertIndex = lastElement.EndIndex - 1
		}
	}

	// First, insert the table
	requests := []*docs.Request{
		{
			InsertTable: &docs.InsertTableRequest{
				Location: &docs.Location{
					Index: insertIndex,
				},
				Rows:    rows,
				Columns: columns,
			},
		},
	}

	resp, err := c.BatchUpdate(documentID, requests)
	if err != nil {
		return nil, fmt.Errorf("failed to insert table: %w", err)
	}

	// If data is provided, populate the cells
	if len(data) > 0 {
		// Re-fetch the document to get the table structure
		doc, err = c.GetDocument(documentID)
		if err != nil {
			return resp, fmt.Errorf("table created but failed to populate: %w", err)
		}

		// Find the table we just inserted (it should be the last table in the doc)
		var table *docs.Table
		if doc.Body != nil && doc.Body.Content != nil {
			for i := len(doc.Body.Content) - 1; i >= 0; i-- {
				if doc.Body.Content[i].Table != nil {
					table = doc.Body.Content[i].Table
					break
				}
			}
		}

		if table != nil {
			var cellRequests []*docs.Request
			for rowIdx, row := range data {
				if int64(rowIdx) >= rows {
					break
				}
				for colIdx, cellText := range row {
					if int64(colIdx) >= columns {
						break
					}
					if cellText == "" {
						continue
					}
					// Get the cell's content start index
					if rowIdx < len(table.TableRows) {
						tableRow := table.TableRows[rowIdx]
						if colIdx < len(tableRow.TableCells) {
							cell := tableRow.TableCells[colIdx]
							if len(cell.Content) > 0 {
								cellIndex := cell.Content[0].StartIndex
								cellRequests = append(cellRequests, &docs.Request{
									InsertText: &docs.InsertTextRequest{
										Location: &docs.Location{
											Index: cellIndex,
										},
										Text: cellText,
									},
								})
							}
						}
					}
				}
			}

			if len(cellRequests) > 0 {
				// Insert text in reverse order of index to avoid shifting issues
				for i, j := 0, len(cellRequests)-1; i < j; i, j = i+1, j-1 {
					cellRequests[i], cellRequests[j] = cellRequests[j], cellRequests[i]
				}
				resp, err = c.BatchUpdate(documentID, cellRequests)
				if err != nil {
					return resp, fmt.Errorf("table created but failed to populate cells: %w", err)
				}
			}
		}
	}

	return resp, nil
}

// GetDocumentAsMarkdown reads a document and converts its structure to markdown
func (c *Client) GetDocumentAsMarkdown(documentID string) (string, error) {
	doc, err := c.GetDocument(documentID)
	if err != nil {
		return "", err
	}

	var result string
	if doc.Body == nil || doc.Body.Content == nil {
		return "", nil
	}

	for _, element := range doc.Body.Content {
		if element.Paragraph != nil {
			para := element.Paragraph

			// Determine heading level
			prefix := ""
			suffix := ""
			if para.ParagraphStyle != nil {
				switch para.ParagraphStyle.NamedStyleType {
				case "HEADING_1":
					prefix = "# "
				case "HEADING_2":
					prefix = "## "
				case "HEADING_3":
					prefix = "### "
				case "HEADING_4":
					prefix = "#### "
				case "HEADING_5":
					prefix = "##### "
				case "HEADING_6":
					prefix = "###### "
				}
			}

			// Check if it's a list item
			if para.Bullet != nil {
				prefix = "- "
			}

			// Extract text with inline formatting
			var paraText string
			for _, elem := range para.Elements {
				if elem.TextRun != nil {
					text := elem.TextRun.Content
					if elem.TextRun.TextStyle != nil {
						if elem.TextRun.TextStyle.Bold && elem.TextRun.TextStyle.Italic {
							text = "***" + trimNewline(text) + "***"
						} else if elem.TextRun.TextStyle.Bold {
							text = "**" + trimNewline(text) + "**"
						} else if elem.TextRun.TextStyle.Italic {
							text = "*" + trimNewline(text) + "*"
						}
						if elem.TextRun.TextStyle.Link != nil && elem.TextRun.TextStyle.Link.Url != "" {
							text = "[" + trimNewline(text) + "](" + elem.TextRun.TextStyle.Link.Url + ")"
						}
					}
					paraText += text
				}
			}

			if paraText != "" {
				result += prefix + paraText + suffix
				// Ensure paragraphs end with newline
				if len(paraText) > 0 && paraText[len(paraText)-1] != '\n' {
					result += "\n"
				}
			}
		} else if element.Table != nil {
			// Convert table to markdown
			table := element.Table
			for rowIdx, row := range table.TableRows {
				result += "|"
				for _, cell := range row.TableCells {
					cellText := ""
					for _, content := range cell.Content {
						if content.Paragraph != nil {
							for _, elem := range content.Paragraph.Elements {
								if elem.TextRun != nil {
									cellText += trimNewline(elem.TextRun.Content)
								}
							}
						}
					}
					result += " " + cellText + " |"
				}
				result += "\n"
				// Add separator after header row
				if rowIdx == 0 {
					result += "|"
					for range row.TableCells {
						result += " --- |"
					}
					result += "\n"
				}
			}
			result += "\n"
		}
	}

	return result, nil
}

// trimNewline removes trailing newlines from text
func trimNewline(s string) string {
	for len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	return s
}
