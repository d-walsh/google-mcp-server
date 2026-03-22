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

// FormatAsTable applies table formatting to a data range:
// bold header with dark blue background and white text, frozen header row,
// alternating row colors, thin borders, and auto-resized columns.
func (c *Client) FormatAsTable(spreadsheetID string, sheetID int64, rangeA1 string) error {
	// Get the spreadsheet metadata to determine the data range
	spreadsheet, err := c.service.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return fmt.Errorf("failed to get spreadsheet: %w", err)
	}

	// Find the sheet
	var sheet *sheets.Sheet
	for _, s := range spreadsheet.Sheets {
		if s.Properties.SheetId == sheetID {
			sheet = s
			break
		}
	}
	if sheet == nil {
		return fmt.Errorf("sheet with ID %d not found", sheetID)
	}

	// Determine the grid range
	var gridRange *sheets.GridRange
	if rangeA1 != "" {
		gridRange, err = parseA1Range(rangeA1, sheetID)
		if err != nil {
			return fmt.Errorf("failed to parse range %q: %w", rangeA1, err)
		}
	} else {
		// Use all data in the sheet
		rowCount := sheet.Properties.GridProperties.RowCount
		colCount := sheet.Properties.GridProperties.ColumnCount
		gridRange = &sheets.GridRange{
			SheetId:          sheetID,
			StartRowIndex:    0,
			EndRowIndex:      rowCount,
			StartColumnIndex: 0,
			EndColumnIndex:   colCount,
		}
	}

	// Header row range (just the first row of the range)
	headerRange := &sheets.GridRange{
		SheetId:          sheetID,
		StartRowIndex:    gridRange.StartRowIndex,
		EndRowIndex:      gridRange.StartRowIndex + 1,
		StartColumnIndex: gridRange.StartColumnIndex,
		EndColumnIndex:   gridRange.EndColumnIndex,
	}

	// Colors: dark blue #1a73e8, white, light gray #f8f9fa
	headerBg := &sheets.Color{Red: 0.102, Green: 0.451, Blue: 0.91}
	headerFg := &sheets.Color{Red: 1.0, Green: 1.0, Blue: 1.0}
	bandColor := &sheets.Color{Red: 0.973, Green: 0.976, Blue: 0.98}
	white := &sheets.Color{Red: 1.0, Green: 1.0, Blue: 1.0}

	// Thin border style
	thinBorder := &sheets.Border{
		Style: "SOLID",
		Color: &sheets.Color{Red: 0.8, Green: 0.8, Blue: 0.8},
	}

	requests := []*sheets.Request{
		// 1. Header styling: bold, white text, dark blue background
		{
			RepeatCell: &sheets.RepeatCellRequest{
				Range: headerRange,
				Cell: &sheets.CellData{
					UserEnteredFormat: &sheets.CellFormat{
						BackgroundColor: headerBg,
						TextFormat: &sheets.TextFormat{
							Bold:            true,
							ForegroundColor: headerFg,
						},
						HorizontalAlignment: "CENTER",
					},
				},
				Fields: "userEnteredFormat(backgroundColor,textFormat,horizontalAlignment)",
			},
		},
		// 2. Alternating row colors via banding
		{
			AddBanding: &sheets.AddBandingRequest{
				BandedRange: &sheets.BandedRange{
					Range: gridRange,
					RowProperties: &sheets.BandingProperties{
						HeaderColor:     headerBg,
						FirstBandColor:  white,
						SecondBandColor: bandColor,
					},
				},
			},
		},
		// 3. Borders around all cells
		{
			UpdateBorders: &sheets.UpdateBordersRequest{
				Range:           gridRange,
				Top:             thinBorder,
				Bottom:          thinBorder,
				Left:            thinBorder,
				Right:           thinBorder,
				InnerHorizontal: thinBorder,
				InnerVertical:   thinBorder,
			},
		},
		// 4. Freeze header row
		{
			UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
				Properties: &sheets.SheetProperties{
					SheetId: sheetID,
					GridProperties: &sheets.GridProperties{
						FrozenRowCount: gridRange.StartRowIndex + 1,
					},
				},
				Fields: "gridProperties.frozenRowCount",
			},
		},
		// 5. Auto-resize columns
		{
			AutoResizeDimensions: &sheets.AutoResizeDimensionsRequest{
				Dimensions: &sheets.DimensionRange{
					SheetId:    sheetID,
					Dimension:  "COLUMNS",
					StartIndex: gridRange.StartColumnIndex,
					EndIndex:   gridRange.EndColumnIndex,
				},
			},
		},
	}

	batchReq := &sheets.BatchUpdateSpreadsheetRequest{Requests: requests}
	_, err = c.service.Spreadsheets.BatchUpdate(spreadsheetID, batchReq).Do()
	if err != nil {
		return fmt.Errorf("failed to format as table: %w", err)
	}
	return nil
}

// AutoResizeColumns auto-resizes columns to fit their content.
func (c *Client) AutoResizeColumns(spreadsheetID string, sheetID int64, startCol, endCol int64) error {
	// If endCol is 0, get sheet properties to determine column count
	if endCol <= 0 {
		spreadsheet, err := c.service.Spreadsheets.Get(spreadsheetID).Do()
		if err != nil {
			return fmt.Errorf("failed to get spreadsheet: %w", err)
		}
		for _, s := range spreadsheet.Sheets {
			if s.Properties.SheetId == sheetID {
				endCol = s.Properties.GridProperties.ColumnCount
				break
			}
		}
		if endCol <= 0 {
			return fmt.Errorf("sheet with ID %d not found", sheetID)
		}
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AutoResizeDimensions: &sheets.AutoResizeDimensionsRequest{
					Dimensions: &sheets.DimensionRange{
						SheetId:    sheetID,
						Dimension:  "COLUMNS",
						StartIndex: startCol,
						EndIndex:   endCol,
					},
				},
			},
		},
	}
	_, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to auto-resize columns: %w", err)
	}
	return nil
}

// FreezeRows freezes the specified number of rows at the top of a sheet.
func (c *Client) FreezeRows(spreadsheetID string, sheetID int64, numRows int64) error {
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
					Properties: &sheets.SheetProperties{
						SheetId: sheetID,
						GridProperties: &sheets.GridProperties{
							FrozenRowCount: numRows,
						},
					},
					Fields: "gridProperties.frozenRowCount",
				},
			},
		},
	}
	_, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to freeze rows: %w", err)
	}
	return nil
}

// parseA1Range converts an A1 notation range (e.g. "A1:D10") to a GridRange.
// Supports formats like "A1:D10", "B2:F", "A:D", "Sheet1!A1:D10".
func parseA1Range(a1 string, sheetID int64) (*sheets.GridRange, error) {
	// Strip sheet name prefix if present (e.g. "Sheet1!A1:D10" -> "A1:D10")
	if idx := findExclamation(a1); idx >= 0 {
		a1 = a1[idx+1:]
	}

	// Split on ':'
	parts := splitColon(a1)
	if len(parts) != 2 {
		return nil, fmt.Errorf("expected range in A1:B2 format, got %q", a1)
	}

	startCol, startRow, err := parseA1Cell(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid start cell %q: %w", parts[0], err)
	}

	endCol, endRow, err := parseA1Cell(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid end cell %q: %w", parts[1], err)
	}

	gr := &sheets.GridRange{
		SheetId:          sheetID,
		StartColumnIndex: int64(startCol),
		EndColumnIndex:   int64(endCol + 1), // exclusive
	}
	if startRow >= 0 {
		gr.StartRowIndex = int64(startRow)
	}
	if endRow >= 0 {
		gr.EndRowIndex = int64(endRow + 1) // exclusive
	}

	return gr, nil
}

// parseA1Cell parses a cell reference like "A1", "B", "AA10" into
// (column 0-indexed, row 0-indexed). Row is -1 if not specified.
func parseA1Cell(cell string) (col int, row int, err error) {
	col = 0
	row = -1
	i := 0

	// Parse column letters
	for i < len(cell) && cell[i] >= 'A' && cell[i] <= 'Z' {
		col = col*26 + int(cell[i]-'A'+1)
		i++
	}
	if i == 0 {
		// Try lowercase
		for i < len(cell) && cell[i] >= 'a' && cell[i] <= 'z' {
			col = col*26 + int(cell[i]-'a'+1)
			i++
		}
	}
	if i == 0 {
		return 0, 0, fmt.Errorf("no column letter found in %q", cell)
	}
	col-- // convert to 0-indexed

	// Parse row number (optional)
	if i < len(cell) {
		rowNum := 0
		for i < len(cell) {
			if cell[i] < '0' || cell[i] > '9' {
				return 0, 0, fmt.Errorf("unexpected character %c in %q", cell[i], cell)
			}
			rowNum = rowNum*10 + int(cell[i]-'0')
			i++
		}
		row = rowNum - 1 // convert to 0-indexed
	}

	return col, row, nil
}

func findExclamation(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '!' {
			return i
		}
	}
	return -1
}

func splitColon(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
