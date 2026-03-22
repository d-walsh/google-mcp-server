package sheets

import (
	"context"
	"encoding/json"
	"fmt"

	"go.ngs.io/google-mcp-server/server"
)

// Handler implements the ServiceHandler interface for Sheets
type Handler struct {
	client *Client
}

// NewHandler creates a new Sheets handler
func NewHandler(client *Client) *Handler {
	return &Handler{client: client}
}

// GetTools returns the available Sheets tools
func (h *Handler) GetTools() []server.Tool {
	return []server.Tool{
		{
			Name:        "sheets_spreadsheet_get",
			Description: "Get spreadsheet metadata",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
				},
				Required: []string{"spreadsheet_id"},
			},
		},
		{
			Name:        "sheets_values_get",
			Description: "Get cell values from a range",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"range": {
						Type:        "string",
						Description: "A1 notation range (e.g., 'Sheet1!A1:B10')",
					},
				},
				Required: []string{"spreadsheet_id", "range"},
			},
		},
		{
			Name:        "sheets_values_update",
			Description: "Update cell values in a range",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"range": {
						Type:        "string",
						Description: "A1 notation range",
					},
					"values": {
						Type:        "array",
						Description: "2D array of values",
						Items: &server.Property{
							Type: "array",
						},
					},
				},
				Required: []string{"spreadsheet_id", "range", "values"},
			},
		},
		{
			Name:        "sheets_spreadsheet_create",
			Description: "Create a new Google Spreadsheet",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"title": {
						Type:        "string",
						Description: "Title for the new spreadsheet",
					},
					"sheet_titles": {
						Type:        "array",
						Description: "Optional list of sheet/tab names to create (defaults to one 'Sheet1' tab)",
						Items: &server.Property{
							Type: "string",
						},
					},
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "sheets_values_append",
			Description: "Append rows to the end of a sheet",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"range": {
						Type:        "string",
						Description: "A1 notation range to append after (e.g., 'Sheet1!A1')",
					},
					"values": {
						Type:        "array",
						Description: "2D array of values to append",
						Items: &server.Property{
							Type: "array",
						},
					},
				},
				Required: []string{"spreadsheet_id", "range", "values"},
			},
		},
		{
			Name:        "sheets_values_clear",
			Description: "Clear cell values in a range (keeps formatting)",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"range": {
						Type:        "string",
						Description: "A1 notation range to clear",
					},
				},
				Required: []string{"spreadsheet_id", "range"},
			},
		},
		{
			Name:        "sheets_sheet_add",
			Description: "Add a new sheet (tab) to a spreadsheet",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"title": {
						Type:        "string",
						Description: "Title for the new sheet tab",
					},
				},
				Required: []string{"spreadsheet_id", "title"},
			},
		},
		{
			Name:        "sheets_sheet_delete",
			Description: "Delete a sheet (tab) from a spreadsheet by its sheetId (numeric ID from spreadsheet_get)",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"sheet_id": {
						Type:        "number",
						Description: "Numeric sheet ID (from sheets_spreadsheet_get, not the tab name)",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id"},
			},
		},
		{
			Name:        "sheets_sheet_rename",
			Description: "Rename a sheet (tab) in a spreadsheet",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"sheet_id": {
						Type:        "number",
						Description: "Numeric sheet ID (from sheets_spreadsheet_get)",
					},
					"new_title": {
						Type:        "string",
						Description: "New title for the sheet tab",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "new_title"},
			},
		},
		{
			Name:        "sheets_format_as_table",
			Description: "Apply table formatting to a data range: bold header row with dark blue background (#1a73e8) and white text, frozen header row, alternating row shading (light gray on even rows), thin borders around all cells, and auto-resized columns",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"sheet_id": {
						Type:        "number",
						Description: "Numeric sheet ID (from sheets_spreadsheet_get)",
					},
					"range": {
						Type:        "string",
						Description: "A1 notation range to format (e.g., 'A1:D10'). If omitted, formats all data in the sheet",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id"},
			},
		},
		{
			Name:        "sheets_auto_resize_columns",
			Description: "Auto-resize columns to fit their content",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"sheet_id": {
						Type:        "number",
						Description: "Numeric sheet ID (from sheets_spreadsheet_get)",
					},
					"start_column": {
						Type:        "number",
						Description: "Start column index (0-indexed, default 0)",
					},
					"end_column": {
						Type:        "number",
						Description: "End column index (0-indexed, exclusive). If omitted, resizes all columns",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id"},
			},
		},
		{
			Name:        "sheets_freeze_rows",
			Description: "Freeze rows at the top of the sheet so they remain visible when scrolling",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"sheet_id": {
						Type:        "number",
						Description: "Numeric sheet ID (from sheets_spreadsheet_get)",
					},
					"num_rows": {
						Type:        "number",
						Description: "Number of rows to freeze (default 1). Set to 0 to unfreeze",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id"},
			},
		},
	}
}

// HandleToolCall handles a tool call for Sheets service
func (h *Handler) HandleToolCall(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	switch name {
	case "sheets_spreadsheet_get":
		var args struct {
			SpreadsheetID string `json:"spreadsheet_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		spreadsheet, err := h.client.GetSpreadsheet(args.SpreadsheetID)
		if err != nil {
			return nil, err
		}

		// Format spreadsheet metadata for response
		result := map[string]interface{}{
			"spreadsheetId":  spreadsheet.SpreadsheetId,
			"spreadsheetUrl": spreadsheet.SpreadsheetUrl,
			"title":          spreadsheet.Properties.Title,
		}

		// Add sheets information
		if len(spreadsheet.Sheets) > 0 {
			sheets := make([]map[string]interface{}, len(spreadsheet.Sheets))
			for i, sheet := range spreadsheet.Sheets {
				sheets[i] = map[string]interface{}{
					"sheetId": sheet.Properties.SheetId,
					"title":   sheet.Properties.Title,
					"index":   sheet.Properties.Index,
				}
			}
			result["sheets"] = sheets
		}

		return result, nil

	case "sheets_values_get":
		var args struct {
			SpreadsheetID string `json:"spreadsheet_id"`
			Range         string `json:"range"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		values, err := h.client.GetValues(args.SpreadsheetID, args.Range)
		if err != nil {
			return nil, err
		}

		// Format values response
		result := map[string]interface{}{
			"range":          values.Range,
			"majorDimension": values.MajorDimension,
			"values":         values.Values,
		}
		return result, nil

	case "sheets_values_update":
		var args struct {
			SpreadsheetID string          `json:"spreadsheet_id"`
			Range         string          `json:"range"`
			Values        [][]interface{} `json:"values"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		response, err := h.client.UpdateValues(args.SpreadsheetID, args.Range, args.Values)
		if err != nil {
			return nil, err
		}

		// Format update response
		result := map[string]interface{}{
			"spreadsheetId":  response.SpreadsheetId,
			"updatedRange":   response.UpdatedRange,
			"updatedRows":    response.UpdatedRows,
			"updatedColumns": response.UpdatedColumns,
			"updatedCells":   response.UpdatedCells,
		}
		return result, nil

	case "sheets_spreadsheet_create":
		var args struct {
			Title       string   `json:"title"`
			SheetTitles []string `json:"sheet_titles"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		spreadsheet, err := h.client.CreateSpreadsheet(args.Title, args.SheetTitles)
		if err != nil {
			return nil, err
		}
		result := map[string]interface{}{
			"spreadsheetId":  spreadsheet.SpreadsheetId,
			"spreadsheetUrl": spreadsheet.SpreadsheetUrl,
			"title":          spreadsheet.Properties.Title,
		}
		if len(spreadsheet.Sheets) > 0 {
			sheets := make([]map[string]interface{}, len(spreadsheet.Sheets))
			for i, sheet := range spreadsheet.Sheets {
				sheets[i] = map[string]interface{}{
					"sheetId": sheet.Properties.SheetId,
					"title":   sheet.Properties.Title,
				}
			}
			result["sheets"] = sheets
		}
		return result, nil

	case "sheets_values_append":
		var args struct {
			SpreadsheetID string          `json:"spreadsheet_id"`
			Range         string          `json:"range"`
			Values        [][]interface{} `json:"values"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		response, err := h.client.AppendValues(args.SpreadsheetID, args.Range, args.Values)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"spreadsheetId": response.SpreadsheetId,
			"updatedRange":  response.Updates.UpdatedRange,
			"updatedRows":   response.Updates.UpdatedRows,
			"updatedCells":  response.Updates.UpdatedCells,
		}, nil

	case "sheets_values_clear":
		var args struct {
			SpreadsheetID string `json:"spreadsheet_id"`
			Range         string `json:"range"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		if err := h.client.ClearValues(args.SpreadsheetID, args.Range); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":       "cleared",
			"clearedRange": args.Range,
		}, nil

	case "sheets_sheet_add":
		var args struct {
			SpreadsheetID string `json:"spreadsheet_id"`
			Title         string `json:"title"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		props, err := h.client.AddSheet(args.SpreadsheetID, args.Title)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"sheetId": props.SheetId,
			"title":   props.Title,
			"index":   props.Index,
		}, nil

	case "sheets_sheet_delete":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			SheetID       float64 `json:"sheet_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		if err := h.client.DeleteSheet(args.SpreadsheetID, int64(args.SheetID)); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status": "deleted",
		}, nil

	case "sheets_sheet_rename":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			SheetID       float64 `json:"sheet_id"`
			NewTitle      string  `json:"new_title"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		if err := h.client.RenameSheet(args.SpreadsheetID, int64(args.SheetID), args.NewTitle); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":   "renamed",
			"newTitle": args.NewTitle,
		}, nil

	case "sheets_format_as_table":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			SheetID       float64 `json:"sheet_id"`
			Range         string  `json:"range"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		if err := h.client.FormatAsTable(args.SpreadsheetID, int64(args.SheetID), args.Range); err != nil {
			return nil, err
		}
		result := map[string]interface{}{
			"status": "formatted",
		}
		if args.Range != "" {
			result["range"] = args.Range
		}
		return result, nil

	case "sheets_auto_resize_columns":
		var args struct {
			SpreadsheetID string   `json:"spreadsheet_id"`
			SheetID       float64  `json:"sheet_id"`
			StartColumn   *float64 `json:"start_column"`
			EndColumn     *float64 `json:"end_column"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		var startCol, endCol int64
		if args.StartColumn != nil {
			startCol = int64(*args.StartColumn)
		}
		if args.EndColumn != nil {
			endCol = int64(*args.EndColumn)
		}
		if err := h.client.AutoResizeColumns(args.SpreadsheetID, int64(args.SheetID), startCol, endCol); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status": "resized",
		}, nil

	case "sheets_freeze_rows":
		var args struct {
			SpreadsheetID string   `json:"spreadsheet_id"`
			SheetID       float64  `json:"sheet_id"`
			NumRows       *float64 `json:"num_rows"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		numRows := int64(1) // default
		if args.NumRows != nil {
			numRows = int64(*args.NumRows)
		}
		if err := h.client.FreezeRows(args.SpreadsheetID, int64(args.SheetID), numRows); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":     "frozen",
			"frozenRows": numRows,
		}, nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// GetResources returns the available Sheets resources
func (h *Handler) GetResources() []server.Resource {
	return []server.Resource{}
}

// HandleResourceCall handles a resource call for Sheets service
func (h *Handler) HandleResourceCall(ctx context.Context, uri string) (interface{}, error) {
	return nil, fmt.Errorf("no resources available for sheets")
}
