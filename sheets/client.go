package sheets

import (
	"context"
	"fmt"

	"go.ngs.io/google-mcp-server/auth"
	"google.golang.org/api/sheets/v4"
)

// Client wraps the Google Sheets API client
type Client struct {
	service *sheets.Service
}

// NewClient creates a new Sheets client
func NewClient(ctx context.Context, oauth *auth.OAuthClient) (*Client, error) {
	service, err := sheets.NewService(ctx, oauth.GetClientOption())
	if err != nil {
		return nil, fmt.Errorf("failed to create sheets service: %w", err)
	}

	return &Client{
		service: service,
	}, nil
}

// CreateSpreadsheet creates a new spreadsheet
func (c *Client) CreateSpreadsheet(title string, sheetTitles []string) (*sheets.Spreadsheet, error) {
	spreadsheet := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title: title,
		},
	}
	if len(sheetTitles) > 0 {
		spreadsheet.Sheets = make([]*sheets.Sheet, len(sheetTitles))
		for i, t := range sheetTitles {
			spreadsheet.Sheets[i] = &sheets.Sheet{
				Properties: &sheets.SheetProperties{
					Title: t,
				},
			}
		}
	}
	created, err := c.service.Spreadsheets.Create(spreadsheet).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create spreadsheet: %w", err)
	}
	return created, nil
}

// AppendValues appends rows to a sheet
func (c *Client) AppendValues(spreadsheetID, range_ string, values [][]interface{}) (*sheets.AppendValuesResponse, error) {
	valueRange := &sheets.ValueRange{
		Values: values,
	}
	response, err := c.service.Spreadsheets.Values.Append(spreadsheetID, range_, valueRange).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to append values: %w", err)
	}
	return response, nil
}

// ClearValues clears cell values in a range
func (c *Client) ClearValues(spreadsheetID, range_ string) error {
	_, err := c.service.Spreadsheets.Values.Clear(spreadsheetID, range_, &sheets.ClearValuesRequest{}).Do()
	if err != nil {
		return fmt.Errorf("failed to clear values: %w", err)
	}
	return nil
}

// GetSpreadsheet gets spreadsheet metadata
func (c *Client) GetSpreadsheet(spreadsheetID string) (*sheets.Spreadsheet, error) {
	spreadsheet, err := c.service.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get spreadsheet: %w", err)
	}
	return spreadsheet, nil
}

// GetValues gets cell values from a range
func (c *Client) GetValues(spreadsheetID, range_ string) (*sheets.ValueRange, error) {
	values, err := c.service.Spreadsheets.Values.Get(spreadsheetID, range_).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get values: %w", err)
	}
	return values, nil
}

// UpdateValues updates cell values in a range
func (c *Client) UpdateValues(spreadsheetID, range_ string, values [][]interface{}) (*sheets.UpdateValuesResponse, error) {
	valueRange := &sheets.ValueRange{
		Values: values,
	}
	response, err := c.service.Spreadsheets.Values.Update(spreadsheetID, range_, valueRange).
		ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to update values: %w", err)
	}
	return response, nil
}

// AddSheet adds a new sheet (tab) to a spreadsheet
func (c *Client) AddSheet(spreadsheetID, title string) (*sheets.SheetProperties, error) {
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddSheet: &sheets.AddSheetRequest{
					Properties: &sheets.SheetProperties{
						Title: title,
					},
				},
			},
		},
	}
	resp, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to add sheet: %w", err)
	}
	if len(resp.Replies) > 0 && resp.Replies[0].AddSheet != nil {
		return resp.Replies[0].AddSheet.Properties, nil
	}
	return nil, fmt.Errorf("no reply from add sheet request")
}

// DeleteSheet deletes a sheet (tab) from a spreadsheet by sheetId
func (c *Client) DeleteSheet(spreadsheetID string, sheetID int64) error {
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				DeleteSheet: &sheets.DeleteSheetRequest{
					SheetId: sheetID,
				},
			},
		},
	}
	_, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to delete sheet: %w", err)
	}
	return nil
}

// RenameSheet renames a sheet (tab) in a spreadsheet
func (c *Client) RenameSheet(spreadsheetID string, sheetID int64, newTitle string) error {
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
					Properties: &sheets.SheetProperties{
						SheetId: sheetID,
						Title:   newTitle,
					},
					Fields: "title",
				},
			},
		},
	}
	_, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to rename sheet: %w", err)
	}
	return nil
}
