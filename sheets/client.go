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

// GetSpreadsheetFull gets spreadsheet with grid data (formatting, values, notes) for specific ranges
func (c *Client) GetSpreadsheetFull(spreadsheetID string, ranges []string) (*sheets.Spreadsheet, error) {
	call := c.service.Spreadsheets.Get(spreadsheetID).IncludeGridData(true)
	if len(ranges) > 0 {
		call = call.Ranges(ranges...)
	}
	spreadsheet, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get spreadsheet with grid data: %w", err)
	}
	return spreadsheet, nil
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

// hexToColor converts a hex color string (e.g., "#ff0000") to a Google Sheets Color
func hexToColor(hex string) (*sheets.Color, error) {
	// Strip leading '#' if present
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return nil, fmt.Errorf("expected 6-character hex color, got %q", hex)
	}
	r, err := parseHexByte(hex[0:2])
	if err != nil {
		return nil, fmt.Errorf("invalid red component: %w", err)
	}
	g, err := parseHexByte(hex[2:4])
	if err != nil {
		return nil, fmt.Errorf("invalid green component: %w", err)
	}
	b, err := parseHexByte(hex[4:6])
	if err != nil {
		return nil, fmt.Errorf("invalid blue component: %w", err)
	}
	return &sheets.Color{
		Red:   float64(r) / 255.0,
		Green: float64(g) / 255.0,
		Blue:  float64(b) / 255.0,
	}, nil
}

func parseHexByte(s string) (byte, error) {
	var val byte
	for i := 0; i < 2; i++ {
		val <<= 4
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			val |= c - '0'
		case c >= 'a' && c <= 'f':
			val |= c - 'a' + 10
		case c >= 'A' && c <= 'F':
			val |= c - 'A' + 10
		default:
			return 0, fmt.Errorf("invalid hex character %c", c)
		}
	}
	return val, nil
}

// SetDataValidation sets data validation on a range
func (c *Client) SetDataValidation(spreadsheetID string, sheetID int64, rangeA1 string, validationType string, values []string, strict bool) error {
	gridRange, err := parseA1Range(rangeA1, sheetID)
	if err != nil {
		return fmt.Errorf("failed to parse range %q: %w", rangeA1, err)
	}

	var condition *sheets.BooleanCondition
	switch validationType {
	case "checkbox":
		condition = &sheets.BooleanCondition{
			Type: "BOOLEAN",
		}
	case "dropdown":
		condValues := make([]*sheets.ConditionValue, len(values))
		for i, v := range values {
			condValues[i] = &sheets.ConditionValue{UserEnteredValue: v}
		}
		condition = &sheets.BooleanCondition{
			Type:   "ONE_OF_LIST",
			Values: condValues,
		}
	case "number_between":
		if len(values) != 2 {
			return fmt.Errorf("number_between requires exactly 2 values [min, max], got %d", len(values))
		}
		condition = &sheets.BooleanCondition{
			Type: "NUMBER_BETWEEN",
			Values: []*sheets.ConditionValue{
				{UserEnteredValue: values[0]},
				{UserEnteredValue: values[1]},
			},
		}
	case "number_greater_than":
		if len(values) != 1 {
			return fmt.Errorf("number_greater_than requires exactly 1 value, got %d", len(values))
		}
		condition = &sheets.BooleanCondition{
			Type: "NUMBER_GREATER",
			Values: []*sheets.ConditionValue{
				{UserEnteredValue: values[0]},
			},
		}
	case "number_less_than":
		if len(values) != 1 {
			return fmt.Errorf("number_less_than requires exactly 1 value, got %d", len(values))
		}
		condition = &sheets.BooleanCondition{
			Type: "NUMBER_LESS",
			Values: []*sheets.ConditionValue{
				{UserEnteredValue: values[0]},
			},
		}
	case "date_after":
		if len(values) != 1 {
			return fmt.Errorf("date_after requires exactly 1 value, got %d", len(values))
		}
		condition = &sheets.BooleanCondition{
			Type: "DATE_AFTER",
			Values: []*sheets.ConditionValue{
				{UserEnteredValue: values[0]},
			},
		}
	case "date_before":
		if len(values) != 1 {
			return fmt.Errorf("date_before requires exactly 1 value, got %d", len(values))
		}
		condition = &sheets.BooleanCondition{
			Type: "DATE_BEFORE",
			Values: []*sheets.ConditionValue{
				{UserEnteredValue: values[0]},
			},
		}
	case "custom_formula":
		if len(values) != 1 {
			return fmt.Errorf("custom_formula requires exactly 1 value, got %d", len(values))
		}
		condition = &sheets.BooleanCondition{
			Type: "CUSTOM_FORMULA",
			Values: []*sheets.ConditionValue{
				{UserEnteredValue: values[0]},
			},
		}
	default:
		return fmt.Errorf("unsupported validation type: %s", validationType)
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				SetDataValidation: &sheets.SetDataValidationRequest{
					Range: gridRange,
					Rule: &sheets.DataValidationRule{
						Condition:    condition,
						Strict:       strict,
						ShowCustomUi: true,
					},
				},
			},
		},
	}
	_, err = c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to set data validation: %w", err)
	}
	return nil
}

// AddConditionalFormatting adds a conditional formatting rule to a range
func (c *Client) AddConditionalFormatting(spreadsheetID string, sheetID int64, rangeA1 string, ruleType string, values []string, bgColor *sheets.Color, textColor *sheets.Color, bold bool) error {
	gridRange, err := parseA1Range(rangeA1, sheetID)
	if err != nil {
		return fmt.Errorf("failed to parse range %q: %w", rangeA1, err)
	}

	condValues := make([]*sheets.ConditionValue, len(values))
	for i, v := range values {
		condValues[i] = &sheets.ConditionValue{UserEnteredValue: v}
	}

	var condType string
	switch ruleType {
	case "number_less_than":
		condType = "NUMBER_LESS"
	case "number_greater_than":
		condType = "NUMBER_GREATER"
	case "number_between":
		condType = "NUMBER_BETWEEN"
	case "text_contains":
		condType = "TEXT_CONTAINS"
	case "text_eq":
		condType = "TEXT_EQ"
	case "is_empty":
		condType = "BLANK"
	case "is_not_empty":
		condType = "NOT_BLANK"
	case "custom_formula":
		condType = "CUSTOM_FORMULA"
	default:
		return fmt.Errorf("unsupported rule type: %s", ruleType)
	}

	format := &sheets.CellFormat{}
	if bgColor != nil {
		format.BackgroundColor = bgColor
	}
	if textColor != nil {
		format.TextFormat = &sheets.TextFormat{
			ForegroundColor: textColor,
		}
	}
	if bold {
		if format.TextFormat == nil {
			format.TextFormat = &sheets.TextFormat{}
		}
		format.TextFormat.Bold = true
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddConditionalFormatRule: &sheets.AddConditionalFormatRuleRequest{
					Rule: &sheets.ConditionalFormatRule{
						Ranges: []*sheets.GridRange{gridRange},
						BooleanRule: &sheets.BooleanRule{
							Condition: &sheets.BooleanCondition{
								Type:   condType,
								Values: condValues,
							},
							Format: format,
						},
					},
				},
			},
		},
	}
	_, err = c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to add conditional formatting: %w", err)
	}
	return nil
}

// SortRange sorts data in a range by a column
func (c *Client) SortRange(spreadsheetID string, sheetID int64, rangeA1 string, sortColumn int64, ascending bool) error {
	gridRange, err := parseA1Range(rangeA1, sheetID)
	if err != nil {
		return fmt.Errorf("failed to parse range %q: %w", rangeA1, err)
	}

	order := "ASCENDING"
	if !ascending {
		order = "DESCENDING"
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				SortRange: &sheets.SortRangeRequest{
					Range: gridRange,
					SortSpecs: []*sheets.SortSpec{
						{
							DimensionIndex: sortColumn,
							SortOrder:      order,
						},
					},
				},
			},
		},
	}
	_, err = c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to sort range: %w", err)
	}
	return nil
}

// MergeCells merges or unmerges cells in a range
func (c *Client) MergeCells(spreadsheetID string, sheetID int64, rangeA1 string, mergeType string, unmerge bool) error {
	gridRange, err := parseA1Range(rangeA1, sheetID)
	if err != nil {
		return fmt.Errorf("failed to parse range %q: %w", rangeA1, err)
	}

	var request *sheets.Request
	if unmerge {
		request = &sheets.Request{
			UnmergeCells: &sheets.UnmergeCellsRequest{
				Range: gridRange,
			},
		}
	} else {
		request = &sheets.Request{
			MergeCells: &sheets.MergeCellsRequest{
				Range:     gridRange,
				MergeType: mergeType,
			},
		}
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{request},
	}
	_, err = c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		action := "merge"
		if unmerge {
			action = "unmerge"
		}
		return fmt.Errorf("failed to %s cells: %w", action, err)
	}
	return nil
}

// CopySheet copies a sheet tab to another spreadsheet
func (c *Client) CopySheet(spreadsheetID string, sheetID int64, destinationSpreadsheetID string) (*sheets.SheetProperties, error) {
	resp, err := c.service.Spreadsheets.Sheets.CopyTo(spreadsheetID, sheetID, &sheets.CopySheetToAnotherSpreadsheetRequest{
		DestinationSpreadsheetId: destinationSpreadsheetID,
	}).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to copy sheet: %w", err)
	}
	return resp, nil
}

// BatchUpdate executes a raw batch update request
func (c *Client) BatchUpdate(spreadsheetID string, requests []*sheets.Request) (*sheets.BatchUpdateSpreadsheetResponse, error) {
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}
	resp, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to batch update: %w", err)
	}
	return resp, nil
}

// FindReplace finds and replaces text in a spreadsheet
func (c *Client) FindReplace(spreadsheetID string, find string, replacement string, sheetID *int64, matchCase bool, matchEntireCell bool, searchByRegex bool) (*sheets.FindReplaceResponse, error) {
	findReplace := &sheets.FindReplaceRequest{
		Find:            find,
		Replacement:     replacement,
		MatchCase:       matchCase,
		MatchEntireCell: matchEntireCell,
		SearchByRegex:   searchByRegex,
		AllSheets:       sheetID == nil,
	}
	if sheetID != nil {
		findReplace.SheetId = *sheetID
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				FindReplace: findReplace,
			},
		},
	}
	resp, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to find and replace: %w", err)
	}
	if len(resp.Replies) > 0 && resp.Replies[0].FindReplace != nil {
		return resp.Replies[0].FindReplace, nil
	}
	return &sheets.FindReplaceResponse{}, nil
}

// InsertRows inserts empty rows at a position
func (c *Client) InsertRows(spreadsheetID string, sheetID int64, startIndex int64, numRows int64) error {
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				InsertDimension: &sheets.InsertDimensionRequest{
					Range: &sheets.DimensionRange{
						SheetId:    sheetID,
						Dimension:  "ROWS",
						StartIndex: startIndex,
						EndIndex:   startIndex + numRows,
					},
					InheritFromBefore: false,
				},
			},
		},
	}
	_, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to insert rows: %w", err)
	}
	return nil
}

// DeleteRows deletes rows from a sheet
func (c *Client) DeleteRows(spreadsheetID string, sheetID int64, startIndex int64, endIndex int64) error {
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				DeleteDimension: &sheets.DeleteDimensionRequest{
					Range: &sheets.DimensionRange{
						SheetId:    sheetID,
						Dimension:  "ROWS",
						StartIndex: startIndex,
						EndIndex:   endIndex,
					},
				},
			},
		},
	}
	_, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to delete rows: %w", err)
	}
	return nil
}

// InsertColumns inserts empty columns at a position
func (c *Client) InsertColumns(spreadsheetID string, sheetID int64, startIndex int64, numColumns int64) error {
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				InsertDimension: &sheets.InsertDimensionRequest{
					Range: &sheets.DimensionRange{
						SheetId:    sheetID,
						Dimension:  "COLUMNS",
						StartIndex: startIndex,
						EndIndex:   startIndex + numColumns,
					},
					InheritFromBefore: false,
				},
			},
		},
	}
	_, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to insert columns: %w", err)
	}
	return nil
}

// DeleteColumns deletes columns from a sheet
func (c *Client) DeleteColumns(spreadsheetID string, sheetID int64, startIndex int64, endIndex int64) error {
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				DeleteDimension: &sheets.DeleteDimensionRequest{
					Range: &sheets.DimensionRange{
						SheetId:    sheetID,
						Dimension:  "COLUMNS",
						StartIndex: startIndex,
						EndIndex:   endIndex,
					},
				},
			},
		},
	}
	_, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to delete columns: %w", err)
	}
	return nil
}

// BatchGetValues reads values from multiple ranges in one call
func (c *Client) BatchGetValues(spreadsheetID string, ranges []string) (*sheets.BatchGetValuesResponse, error) {
	resp, err := c.service.Spreadsheets.Values.BatchGet(spreadsheetID).Ranges(ranges...).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to batch get values: %w", err)
	}
	return resp, nil
}

// CreateChart creates an embedded chart in a sheet
func (c *Client) CreateChart(spreadsheetID string, sheetID int64, chartType string, dataRange string, title string, positionRow int64, positionCol int64) (*sheets.AddChartResponse, error) {
	gridRange, err := parseA1Range(dataRange, sheetID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse data range %q: %w", dataRange, err)
	}

	basicChart := &sheets.BasicChartSpec{
		ChartType: chartType,
		Axis: []*sheets.BasicChartAxis{
			{Position: "BOTTOM_AXIS"},
			{Position: "LEFT_AXIS"},
		},
		Domains: []*sheets.BasicChartDomain{
			{
				Domain: &sheets.ChartData{
					SourceRange: &sheets.ChartSourceRange{
						Sources: []*sheets.GridRange{
							{
								SheetId:          gridRange.SheetId,
								StartRowIndex:    gridRange.StartRowIndex,
								EndRowIndex:      gridRange.EndRowIndex,
								StartColumnIndex: gridRange.StartColumnIndex,
								EndColumnIndex:   gridRange.StartColumnIndex + 1,
							},
						},
					},
				},
			},
		},
		Series: []*sheets.BasicChartSeries{},
	}

	// Add series for each column after the first (domain) column
	for col := gridRange.StartColumnIndex + 1; col < gridRange.EndColumnIndex; col++ {
		basicChart.Series = append(basicChart.Series, &sheets.BasicChartSeries{
			Series: &sheets.ChartData{
				SourceRange: &sheets.ChartSourceRange{
					Sources: []*sheets.GridRange{
						{
							SheetId:          gridRange.SheetId,
							StartRowIndex:    gridRange.StartRowIndex,
							EndRowIndex:      gridRange.EndRowIndex,
							StartColumnIndex: col,
							EndColumnIndex:   col + 1,
						},
					},
				},
			},
			TargetAxis: "LEFT_AXIS",
		})
	}

	chart := &sheets.EmbeddedChart{
		Spec: &sheets.ChartSpec{
			Title:      title,
			BasicChart: basicChart,
		},
		Position: &sheets.EmbeddedObjectPosition{
			OverlayPosition: &sheets.OverlayPosition{
				AnchorCell: &sheets.GridCoordinate{
					SheetId:     sheetID,
					RowIndex:    positionRow,
					ColumnIndex: positionCol,
				},
			},
		},
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddChart: &sheets.AddChartRequest{
					Chart: chart,
				},
			},
		},
	}
	resp, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create chart: %w", err)
	}
	if len(resp.Replies) > 0 && resp.Replies[0].AddChart != nil {
		return resp.Replies[0].AddChart, nil
	}
	return nil, fmt.Errorf("no reply from add chart request")
}

// FreezeColumns freezes columns on the left side of the sheet
func (c *Client) FreezeColumns(spreadsheetID string, sheetID int64, numColumns int64) error {
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
					Properties: &sheets.SheetProperties{
						SheetId: sheetID,
						GridProperties: &sheets.GridProperties{
							FrozenColumnCount: numColumns,
						},
					},
					Fields: "gridProperties.frozenColumnCount",
				},
			},
		},
	}
	_, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to freeze columns: %w", err)
	}
	return nil
}

// AddNamedRange creates a named range in a spreadsheet
func (c *Client) AddNamedRange(spreadsheetID string, name string, sheetID int64, rangeA1 string) (*sheets.NamedRange, error) {
	gridRange, err := parseA1Range(rangeA1, sheetID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse range %q: %w", rangeA1, err)
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddNamedRange: &sheets.AddNamedRangeRequest{
					NamedRange: &sheets.NamedRange{
						Name:  name,
						Range: gridRange,
					},
				},
			},
		},
	}
	resp, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to add named range: %w", err)
	}
	if len(resp.Replies) > 0 && resp.Replies[0].AddNamedRange != nil {
		return resp.Replies[0].AddNamedRange.NamedRange, nil
	}
	return nil, fmt.Errorf("no reply from add named range request")
}

// DuplicateSheet duplicates a sheet tab within the same spreadsheet
func (c *Client) DuplicateSheet(spreadsheetID string, sheetID int64, newName string) (*sheets.SheetProperties, error) {
	dupReq := &sheets.DuplicateSheetRequest{
		SourceSheetId: sheetID,
	}
	if newName != "" {
		dupReq.NewSheetName = newName
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				DuplicateSheet: dupReq,
			},
		},
	}
	resp, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to duplicate sheet: %w", err)
	}
	if len(resp.Replies) > 0 && resp.Replies[0].DuplicateSheet != nil {
		return resp.Replies[0].DuplicateSheet.Properties, nil
	}
	return nil, fmt.Errorf("no reply from duplicate sheet request")
}

// AddProtectedRange protects a range from editing
func (c *Client) AddProtectedRange(spreadsheetID string, sheetID int64, rangeA1 string, description string, warningOnly bool) (*sheets.ProtectedRange, error) {
	gridRange, err := parseA1Range(rangeA1, sheetID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse range %q: %w", rangeA1, err)
	}

	protectedRange := &sheets.ProtectedRange{
		Range:       gridRange,
		Description: description,
		WarningOnly: warningOnly,
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddProtectedRange: &sheets.AddProtectedRangeRequest{
					ProtectedRange: protectedRange,
				},
			},
		},
	}
	resp, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to add protected range: %w", err)
	}
	if len(resp.Replies) > 0 && resp.Replies[0].AddProtectedRange != nil {
		return resp.Replies[0].AddProtectedRange.ProtectedRange, nil
	}
	return nil, fmt.Errorf("no reply from add protected range request")
}

// SetColumnWidth sets the width of columns in a range
func (c *Client) SetColumnWidth(spreadsheetID string, sheetID int64, startColumn int64, endColumn int64, width int64) error {
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				UpdateDimensionProperties: &sheets.UpdateDimensionPropertiesRequest{
					Range: &sheets.DimensionRange{
						SheetId:    sheetID,
						Dimension:  "COLUMNS",
						StartIndex: startColumn,
						EndIndex:   endColumn,
					},
					Properties: &sheets.DimensionProperties{
						PixelSize: width,
					},
					Fields: "pixelSize",
				},
			},
		},
	}
	_, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to set column width: %w", err)
	}
	return nil
}

// ListCharts lists all embedded charts in a spreadsheet, optionally filtered by sheetId
func (c *Client) ListCharts(spreadsheetID string, sheetID *int64) ([]map[string]interface{}, error) {
	spreadsheet, err := c.service.Spreadsheets.Get(spreadsheetID).Fields("sheets(properties,charts)").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get spreadsheet charts: %w", err)
	}

	var charts []map[string]interface{}
	for _, sheet := range spreadsheet.Sheets {
		if sheetID != nil && sheet.Properties.SheetId != *sheetID {
			continue
		}
		for _, chart := range sheet.Charts {
			chartInfo := map[string]interface{}{
				"chartId": chart.ChartId,
				"sheetId": sheet.Properties.SheetId,
			}

			// Extract title
			if chart.Spec != nil && chart.Spec.Title != "" {
				chartInfo["title"] = chart.Spec.Title
			}

			// Detect chart type
			if chart.Spec != nil {
				switch {
				case chart.Spec.BasicChart != nil:
					chartInfo["type"] = chart.Spec.BasicChart.ChartType
				case chart.Spec.PieChart != nil:
					chartInfo["type"] = "PIE"
				case chart.Spec.WaterfallChart != nil:
					chartInfo["type"] = "WATERFALL"
				case chart.Spec.HistogramChart != nil:
					chartInfo["type"] = "HISTOGRAM"
				case chart.Spec.BubbleChart != nil:
					chartInfo["type"] = "BUBBLE"
				case chart.Spec.CandlestickChart != nil:
					chartInfo["type"] = "CANDLESTICK"
				case chart.Spec.OrgChart != nil:
					chartInfo["type"] = "ORG"
				case chart.Spec.TreemapChart != nil:
					chartInfo["type"] = "TREEMAP"
				case chart.Spec.ScorecardChart != nil:
					chartInfo["type"] = "SCORECARD"
				default:
					chartInfo["type"] = "UNKNOWN"
				}
			}

			// Extract position
			if chart.Position != nil && chart.Position.OverlayPosition != nil {
				overlay := chart.Position.OverlayPosition
				if overlay.AnchorCell != nil {
					chartInfo["anchorCell"] = map[string]interface{}{
						"sheetId":     overlay.AnchorCell.SheetId,
						"rowIndex":    overlay.AnchorCell.RowIndex,
						"columnIndex": overlay.AnchorCell.ColumnIndex,
					}
				}
				if overlay.WidthPixels > 0 {
					chartInfo["width"] = overlay.WidthPixels
				}
				if overlay.HeightPixels > 0 {
					chartInfo["height"] = overlay.HeightPixels
				}
			}

			charts = append(charts, chartInfo)
		}
	}

	return charts, nil
}

// DeleteChart deletes an embedded chart by chartId
func (c *Client) DeleteChart(spreadsheetID string, chartID int64) error {
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				DeleteEmbeddedObject: &sheets.DeleteEmbeddedObjectRequest{
					ObjectId: chartID,
				},
			},
		},
	}
	_, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to delete chart: %w", err)
	}
	return nil
}

// UpdateChartPosition moves and/or resizes an embedded chart
func (c *Client) UpdateChartPosition(spreadsheetID string, chartID int64, sheetID *int64, anchorRow *int64, anchorCol *int64, width *int64, height *int64) error {
	// We need to build the new position and determine which fields changed
	newPosition := &sheets.EmbeddedObjectPosition{
		OverlayPosition: &sheets.OverlayPosition{
			AnchorCell: &sheets.GridCoordinate{},
		},
	}

	var fields []string

	if sheetID != nil {
		newPosition.OverlayPosition.AnchorCell.SheetId = *sheetID
		fields = append(fields, "anchorCell.sheetId")
	}
	if anchorRow != nil {
		newPosition.OverlayPosition.AnchorCell.RowIndex = *anchorRow
		fields = append(fields, "anchorCell.rowIndex")
	}
	if anchorCol != nil {
		newPosition.OverlayPosition.AnchorCell.ColumnIndex = *anchorCol
		fields = append(fields, "anchorCell.columnIndex")
	}
	if width != nil {
		newPosition.OverlayPosition.WidthPixels = *width
		fields = append(fields, "widthPixels")
	}
	if height != nil {
		newPosition.OverlayPosition.HeightPixels = *height
		fields = append(fields, "heightPixels")
	}

	if len(fields) == 0 {
		return fmt.Errorf("at least one position field must be specified")
	}

	// Build the fields mask string
	fieldsStr := ""
	for i, f := range fields {
		if i > 0 {
			fieldsStr += ","
		}
		fieldsStr += f
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				UpdateEmbeddedObjectPosition: &sheets.UpdateEmbeddedObjectPositionRequest{
					ObjectId:    chartID,
					NewPosition: newPosition,
					Fields:      fieldsStr,
				},
			},
		},
	}
	_, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to update chart position: %w", err)
	}
	return nil
}

// CreateWaterfallChart creates a waterfall chart with customizable colors and subtotals
func (c *Client) CreateWaterfallChart(spreadsheetID string, sheetID int64, domainRange string, dataRange string, title string, subtotalIndices []int64, positionRow int64, positionCol int64, width int64, height int64, positiveColor *sheets.Color, negativeColor *sheets.Color, subtotalColor *sheets.Color) (*sheets.AddChartResponse, error) {
	domainGrid, err := parseA1Range(domainRange, sheetID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse domain range %q: %w", domainRange, err)
	}

	dataGrid, err := parseA1Range(dataRange, sheetID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse data range %q: %w", dataRange, err)
	}

	// Build custom subtotals
	var customSubtotals []*sheets.WaterfallChartCustomSubtotal
	for _, idx := range subtotalIndices {
		customSubtotals = append(customSubtotals, &sheets.WaterfallChartCustomSubtotal{
			SubtotalIndex: idx,
			Label:         "",
		})
	}

	// Connector line style (thin dashed line)
	connectorLine := &sheets.LineStyle{
		Type:  "MEDIUM_DASHED",
		Width: 1,
	}

	waterfallSpec := &sheets.WaterfallChartSpec{
		Domain: &sheets.WaterfallChartDomain{
			Data: &sheets.ChartData{
				SourceRange: &sheets.ChartSourceRange{
					Sources: []*sheets.GridRange{domainGrid},
				},
			},
		},
		Series: []*sheets.WaterfallChartSeries{
			{
				Data: &sheets.ChartData{
					SourceRange: &sheets.ChartSourceRange{
						Sources: []*sheets.GridRange{dataGrid},
					},
				},
				PositiveColumnsStyle: &sheets.WaterfallChartColumnStyle{
					Color: positiveColor,
				},
				NegativeColumnsStyle: &sheets.WaterfallChartColumnStyle{
					Color: negativeColor,
				},
				SubtotalColumnsStyle: &sheets.WaterfallChartColumnStyle{
					Color: subtotalColor,
				},
				CustomSubtotals: customSubtotals,
			},
		},
		ConnectorLineStyle: connectorLine,
	}

	chart := &sheets.EmbeddedChart{
		Spec: &sheets.ChartSpec{
			Title:          title,
			WaterfallChart: waterfallSpec,
		},
		Position: &sheets.EmbeddedObjectPosition{
			OverlayPosition: &sheets.OverlayPosition{
				AnchorCell: &sheets.GridCoordinate{
					SheetId:     sheetID,
					RowIndex:    positionRow,
					ColumnIndex: positionCol,
				},
				WidthPixels:  width,
				HeightPixels: height,
			},
		},
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddChart: &sheets.AddChartRequest{
					Chart: chart,
				},
			},
		},
	}
	resp, err := c.service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create waterfall chart: %w", err)
	}
	if len(resp.Replies) > 0 && resp.Replies[0].AddChart != nil {
		return resp.Replies[0].AddChart, nil
	}
	return nil, fmt.Errorf("no reply from add waterfall chart request")
}
