package sheets

import (
	"context"
	"encoding/json"
	"fmt"

	"go.ngs.io/google-mcp-server/server"
	"google.golang.org/api/sheets/v4"
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
			Name:        "sheets_spreadsheet_get_full",
			Description: "Get spreadsheet with full cell data including formatting, notes, and values for specific ranges. Returns cell background colors, text formatting (bold, italic, font, size), borders, number formats, notes, hyperlinks, and data validation. Use this to inspect how a sheet looks.",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"ranges": {
						Type:        "array",
						Description: "A1 notation ranges to include (e.g., ['Sheet1!A1:D5']). If omitted, returns all data (can be large).",
						Items: &server.Property{
							Type: "string",
						},
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
		{
			Name:        "sheets_add_data_validation",
			Description: "Add data validation to a range (checkboxes, dropdown lists, number ranges, date constraints, or custom formulas)",
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
						Description: "A1 notation range (e.g., 'E2:E100')",
					},
					"type": {
						Type:        "string",
						Description: "Validation type",
						Enum:        []string{"checkbox", "dropdown", "number_between", "number_greater_than", "number_less_than", "date_after", "date_before", "custom_formula"},
					},
					"values": {
						Type:        "array",
						Description: "For 'dropdown': list of options. For 'number_between': [min, max]. For 'custom_formula': [formula]. Not needed for 'checkbox'.",
						Items: &server.Property{
							Type: "string",
						},
					},
					"strict": {
						Type:        "boolean",
						Description: "Reject invalid input (default true)",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "range", "type"},
			},
		},
		{
			Name:        "sheets_add_conditional_formatting",
			Description: "Add conditional formatting rules to highlight cells based on their values",
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
						Description: "A1 notation range (e.g., 'A1:D10')",
					},
					"rule_type": {
						Type:        "string",
						Description: "Condition type for the rule",
						Enum:        []string{"number_less_than", "number_greater_than", "number_between", "text_contains", "text_eq", "is_empty", "is_not_empty", "custom_formula"},
					},
					"values": {
						Type:        "array",
						Description: "Threshold values or formula (e.g., ['100'] for number_greater_than, ['50', '100'] for number_between, ['=A1>B1'] for custom_formula)",
						Items: &server.Property{
							Type: "string",
						},
					},
					"background_color": {
						Type:        "string",
						Description: "Hex color for matching cells background (e.g., '#ff0000' for red)",
					},
					"text_color": {
						Type:        "string",
						Description: "Hex color for text in matching cells (e.g., '#ffffff' for white)",
					},
					"bold": {
						Type:        "boolean",
						Description: "Make text bold in matching cells",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "range", "rule_type", "values"},
			},
		},
		{
			Name:        "sheets_sort_range",
			Description: "Sort data in a range by one or more columns",
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
						Description: "A1 notation range to sort (e.g., 'A2:D100')",
					},
					"sort_column": {
						Type:        "number",
						Description: "0-indexed column to sort by (e.g., 0 for column A, 1 for column B)",
					},
					"ascending": {
						Type:        "boolean",
						Description: "Sort ascending (default true). Set false for descending",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "range", "sort_column"},
			},
		},
		{
			Name:        "sheets_merge_cells",
			Description: "Merge or unmerge cells in a range",
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
						Description: "A1 notation range to merge/unmerge (e.g., 'A1:C1')",
					},
					"merge_type": {
						Type:        "string",
						Description: "Merge type (default 'MERGE_ALL')",
						Enum:        []string{"MERGE_ALL", "MERGE_ROWS", "MERGE_COLUMNS"},
					},
					"unmerge": {
						Type:        "boolean",
						Description: "If true, unmerge cells instead of merging (default false)",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "range"},
			},
		},
		{
			Name:        "sheets_copy_sheet",
			Description: "Copy a sheet tab to another spreadsheet (or the same spreadsheet)",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Source spreadsheet ID",
					},
					"sheet_id": {
						Type:        "number",
						Description: "Source sheet tab ID to copy",
					},
					"destination_spreadsheet_id": {
						Type:        "string",
						Description: "Target spreadsheet ID (can be the same as source)",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "destination_spreadsheet_id"},
			},
		},
		{
			Name:        "sheets_batch_update",
			Description: "Generic batch update for power users — accepts raw BatchUpdate request JSON matching the Google Sheets API schema. Use this for any operation not covered by specific tools.",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"requests": {
						Type:        "array",
						Description: "Array of request objects matching the Google Sheets API BatchUpdate schema",
					},
				},
				Required: []string{"spreadsheet_id", "requests"},
			},
		},
		{
			Name:        "sheets_find_replace",
			Description: "Find and replace text across a sheet or entire spreadsheet",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"find": {
						Type:        "string",
						Description: "Text to find",
					},
					"replacement": {
						Type:        "string",
						Description: "Replacement text",
					},
					"sheet_id": {
						Type:        "number",
						Description: "Limit search to a specific sheet (omit to search all sheets)",
					},
					"match_case": {
						Type:        "boolean",
						Description: "Case-sensitive search (default false)",
					},
					"match_entire_cell": {
						Type:        "boolean",
						Description: "Only match if the entire cell matches (default false)",
					},
					"search_by_regex": {
						Type:        "boolean",
						Description: "Treat 'find' as a regular expression (default false)",
					},
				},
				Required: []string{"spreadsheet_id", "find", "replacement"},
			},
		},
		{
			Name:        "sheets_set_column_width",
			Description: "Set specific column widths in pixels",
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
						Description: "Start column index (0-indexed, e.g., 0 for column A)",
					},
					"end_column": {
						Type:        "number",
						Description: "End column index (0-indexed, exclusive, e.g., 3 for columns A-C)",
					},
					"width": {
						Type:        "number",
						Description: "Column width in pixels",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "start_column", "end_column", "width"},
			},
		},
		{
			Name:        "sheets_insert_rows",
			Description: "Insert empty rows at a position in a sheet",
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
					"start_index": {
						Type:        "number",
						Description: "0-indexed row to insert before",
					},
					"num_rows": {
						Type:        "number",
						Description: "Number of rows to insert",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "start_index", "num_rows"},
			},
		},
		{
			Name:        "sheets_delete_rows",
			Description: "Delete rows from a sheet",
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
					"start_index": {
						Type:        "number",
						Description: "0-indexed first row to delete",
					},
					"end_index": {
						Type:        "number",
						Description: "0-indexed exclusive end row",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "start_index", "end_index"},
			},
		},
		{
			Name:        "sheets_insert_columns",
			Description: "Insert empty columns at a position in a sheet",
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
					"start_index": {
						Type:        "number",
						Description: "0-indexed column to insert before",
					},
					"num_columns": {
						Type:        "number",
						Description: "Number of columns to insert",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "start_index", "num_columns"},
			},
		},
		{
			Name:        "sheets_delete_columns",
			Description: "Delete columns from a sheet",
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
					"start_index": {
						Type:        "number",
						Description: "0-indexed first column to delete",
					},
					"end_index": {
						Type:        "number",
						Description: "0-indexed exclusive end column",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "start_index", "end_index"},
			},
		},
		{
			Name:        "sheets_batch_get_values",
			Description: "Read values from multiple ranges in one call",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"ranges": {
						Type:        "array",
						Description: "List of A1 notation ranges (e.g., ['Sheet1!A1:B5', 'Sheet2!C1:D10'])",
						Items: &server.Property{
							Type: "string",
						},
					},
				},
				Required: []string{"spreadsheet_id", "ranges"},
			},
		},
		{
			Name:        "sheets_create_chart",
			Description: "Create an embedded chart in a sheet. The first column of the data range is used as the domain (X axis) and remaining columns become series (Y axis). For PIE charts, column 0 is domain and column 1 is the series. For WATERFALL charts, column 0 is domain and column 1 is values with default positive/negative/subtotal colors.",
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
					"chart_type": {
						Type:        "string",
						Description: "Chart type. PIE uses PieChartSpec, WATERFALL uses WaterfallChartSpec, all others use BasicChartSpec.",
						Enum:        []string{"LINE", "BAR", "COLUMN", "PIE", "AREA", "SCATTER", "WATERFALL"},
					},
					"data_range": {
						Type:        "string",
						Description: "A1 notation for the chart data (e.g., 'A1:C10')",
					},
					"title": {
						Type:        "string",
						Description: "Chart title (optional)",
					},
					"position_row": {
						Type:        "number",
						Description: "Anchor row for chart placement (0-indexed, default 0)",
					},
					"position_col": {
						Type:        "number",
						Description: "Anchor column for chart placement (0-indexed, default 0)",
					},
					"legend_position": {
						Type:        "string",
						Description: "Legend position (default 'BOTTOM_LEGEND')",
						Enum:        []string{"BOTTOM_LEGEND", "RIGHT_LEGEND", "TOP_LEGEND", "NO_LEGEND"},
					},
					"stacked": {
						Type:        "boolean",
						Description: "If true, stack series (only applies to BAR, COLUMN, AREA). Default false.",
					},
					"width": {
						Type:        "number",
						Description: "Chart width in pixels (default 700)",
					},
					"height": {
						Type:        "number",
						Description: "Chart height in pixels (default 400)",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "chart_type", "data_range"},
			},
		},
		{
			Name:        "sheets_freeze_columns",
			Description: "Freeze columns on the left side of the sheet so they remain visible when scrolling horizontally",
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
					"num_columns": {
						Type:        "number",
						Description: "Number of columns to freeze (default 1). Set to 0 to unfreeze",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id"},
			},
		},
		{
			Name:        "sheets_add_named_range",
			Description: "Create a named range that can be referenced by name in formulas",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"name": {
						Type:        "string",
						Description: "Range name (e.g., 'NET_WORTH_TOTAL')",
					},
					"sheet_id": {
						Type:        "number",
						Description: "Numeric sheet ID (from sheets_spreadsheet_get)",
					},
					"range": {
						Type:        "string",
						Description: "A1 notation range (e.g., 'A1:B10')",
					},
				},
				Required: []string{"spreadsheet_id", "name", "sheet_id", "range"},
			},
		},
		{
			Name:        "sheets_duplicate_sheet",
			Description: "Duplicate a sheet tab within the same spreadsheet",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"sheet_id": {
						Type:        "number",
						Description: "Numeric sheet ID of the source sheet to duplicate",
					},
					"new_name": {
						Type:        "string",
						Description: "Name for the duplicated sheet (optional, defaults to 'Copy of <original>')",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id"},
			},
		},
		{
			Name:        "sheets_add_protected_range",
			Description: "Protect a range from editing. Can block edits entirely or show a warning when users try to edit.",
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
						Description: "A1 notation range to protect (e.g., 'A1:D10')",
					},
					"description": {
						Type:        "string",
						Description: "Description of why the range is protected (optional)",
					},
					"warning_only": {
						Type:        "boolean",
						Description: "Show a warning instead of blocking edits (default false)",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "range"},
			},
		},
		{
			Name:        "sheets_list_charts",
			Description: "List all embedded charts in a spreadsheet with their details (chartId, title, type, position, dimensions)",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"sheet_id": {
						Type:        "number",
						Description: "Filter to a specific sheet (optional, omit for all sheets)",
					},
				},
				Required: []string{"spreadsheet_id"},
			},
		},
		{
			Name:        "sheets_delete_chart",
			Description: "Delete an embedded chart by its chartId",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"chart_id": {
						Type:        "number",
						Description: "Chart ID to delete (from sheets_list_charts)",
					},
				},
				Required: []string{"spreadsheet_id", "chart_id"},
			},
		},
		{
			Name:        "sheets_update_chart_position",
			Description: "Move and/or resize an embedded chart. Only specified fields are updated.",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"chart_id": {
						Type:        "number",
						Description: "Chart ID to move/resize (from sheets_list_charts)",
					},
					"sheet_id": {
						Type:        "number",
						Description: "Move chart to a different sheet (optional)",
					},
					"anchor_row": {
						Type:        "number",
						Description: "New anchor row (0-indexed, optional)",
					},
					"anchor_col": {
						Type:        "number",
						Description: "New anchor column (0-indexed, optional)",
					},
					"width": {
						Type:        "number",
						Description: "New width in pixels (optional)",
					},
					"height": {
						Type:        "number",
						Description: "New height in pixels (optional)",
					},
				},
				Required: []string{"spreadsheet_id", "chart_id"},
			},
		},
		{
			Name:        "sheets_update_chart",
			Description: "Update an existing chart's title, type, or data range without deleting and recreating it. Fetches the current chart spec, modifies only the specified fields, and sends the updated spec back.",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"chart_id": {
						Type:        "number",
						Description: "Chart ID to update (from sheets_list_charts)",
					},
					"title": {
						Type:        "string",
						Description: "New chart title (optional)",
					},
					"chart_type": {
						Type:        "string",
						Description: "New chart type (optional). When changing to PIE or WATERFALL, data_range must also be provided.",
						Enum:        []string{"LINE", "BAR", "COLUMN", "PIE", "AREA", "SCATTER", "WATERFALL"},
					},
					"data_range": {
						Type:        "string",
						Description: "New data range in A1 notation (optional). Requires sheet_id.",
					},
					"sheet_id": {
						Type:        "number",
						Description: "Numeric sheet ID — required when changing data_range (needed for GridRange conversion)",
					},
				},
				Required: []string{"spreadsheet_id", "chart_id"},
			},
		},
		{
			Name:        "sheets_set_number_format",
			Description: "Set the number format for a range of cells (currency, percent, date, number, text, scientific)",
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
						Description: "A1 notation range (e.g., 'B2:B100')",
					},
					"format_type": {
						Type:        "string",
						Description: "Number format type",
						Enum:        []string{"CURRENCY", "PERCENT", "DATE", "NUMBER", "TEXT", "SCIENTIFIC"},
					},
					"pattern": {
						Type:        "string",
						Description: "Optional format pattern (e.g., '$#,##0.00', '0.00%', 'yyyy-mm-dd', '#,##0.00'). If omitted, uses the default for the format type.",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "range", "format_type"},
			},
		},
		{
			Name:        "sheets_add_note",
			Description: "Add a note (yellow sticky comment) to a single cell",
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
						Description: "A1 notation for a single cell (e.g., 'A1:A1' or 'C5:C5')",
					},
					"note": {
						Type:        "string",
						Description: "Note text to add to the cell. Set to empty string to remove an existing note.",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "range", "note"},
			},
		},
		{
			Name:        "sheets_set_sheet_visibility",
			Description: "Hide or show a sheet tab in a spreadsheet",
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
					"hidden": {
						Type:        "boolean",
						Description: "True to hide the sheet, false to show it",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "hidden"},
			},
		},
		{
			Name:        "sheets_set_row_height",
			Description: "Set the height of rows in a range",
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
					"start_index": {
						Type:        "number",
						Description: "0-indexed first row",
					},
					"end_index": {
						Type:        "number",
						Description: "0-indexed exclusive end row",
					},
					"height": {
						Type:        "number",
						Description: "Row height in pixels",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "start_index", "end_index", "height"},
			},
		},
		{
			Name:        "sheets_batch_update_values",
			Description: "Write values to multiple ranges in one call. More efficient than multiple sheets_values_update calls.",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"spreadsheet_id": {
						Type:        "string",
						Description: "Spreadsheet ID",
					},
					"data": {
						Type:        "array",
						Description: "Array of {range, values} objects. Each range is A1 notation (e.g., 'Sheet1!A1:B2'), values is a 2D array.",
					},
				},
				Required: []string{"spreadsheet_id", "data"},
			},
		},
		{
			Name:        "sheets_create_waterfall_chart",
			Description: "Create a waterfall chart with positive/negative/subtotal bar coloring. Useful for budget breakdowns, P&L waterfalls, and variance analysis.",
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
					"domain_range": {
						Type:        "string",
						Description: "A1 notation for labels/categories (e.g., 'P2:P22')",
					},
					"data_range": {
						Type:        "string",
						Description: "A1 notation for values (e.g., 'Q2:Q22')",
					},
					"title": {
						Type:        "string",
						Description: "Chart title (optional)",
					},
					"subtotal_indices": {
						Type:        "array",
						Description: "0-indexed row positions within the data to mark as subtotal bars (gray). E.g., [5, 10, 20] marks the 6th, 11th, and 21st data points as subtotals.",
						Items: &server.Property{
							Type: "number",
						},
					},
					"position_row": {
						Type:        "number",
						Description: "Anchor row for chart placement (0-indexed, default 0)",
					},
					"position_col": {
						Type:        "number",
						Description: "Anchor column for chart placement (0-indexed, default 0)",
					},
					"width": {
						Type:        "number",
						Description: "Chart width in pixels (default 800)",
					},
					"height": {
						Type:        "number",
						Description: "Chart height in pixels (default 500)",
					},
					"positive_color": {
						Type:        "string",
						Description: "Hex color for positive (increase) bars (default '#4285f4' blue)",
					},
					"negative_color": {
						Type:        "string",
						Description: "Hex color for negative (decrease) bars (default '#ea4335' red)",
					},
					"subtotal_color": {
						Type:        "string",
						Description: "Hex color for subtotal bars (default '#bfbfbf' gray)",
					},
				},
				Required: []string{"spreadsheet_id", "sheet_id", "domain_range", "data_range"},
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

	case "sheets_spreadsheet_get_full":
		var args struct {
			SpreadsheetID string   `json:"spreadsheet_id"`
			Ranges        []string `json:"ranges"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		spreadsheet, err := h.client.GetSpreadsheetFull(args.SpreadsheetID, args.Ranges)
		if err != nil {
			return nil, err
		}

		result := map[string]interface{}{
			"spreadsheetId":  spreadsheet.SpreadsheetId,
			"title":          spreadsheet.Properties.Title,
			"spreadsheetUrl": spreadsheet.SpreadsheetUrl,
		}

		// Extract sheet data with formatting info
		sheetsData := make([]map[string]interface{}, 0)
		for _, sheet := range spreadsheet.Sheets {
			sheetInfo := map[string]interface{}{
				"sheetId": sheet.Properties.SheetId,
				"title":   sheet.Properties.Title,
			}
			if sheet.Properties.GridProperties != nil {
				sheetInfo["frozenRowCount"] = sheet.Properties.GridProperties.FrozenRowCount
				sheetInfo["frozenColumnCount"] = sheet.Properties.GridProperties.FrozenColumnCount
				sheetInfo["rowCount"] = sheet.Properties.GridProperties.RowCount
				sheetInfo["columnCount"] = sheet.Properties.GridProperties.ColumnCount
			}

			// Extract grid data (cell values + formatting)
			if len(sheet.Data) > 0 {
				grids := make([]map[string]interface{}, 0)
				for _, gridData := range sheet.Data {
					rows := make([]map[string]interface{}, 0)
					for rowIdx, row := range gridData.RowData {
						cells := make([]map[string]interface{}, 0)
						for colIdx, cell := range row.Values {
							cellInfo := map[string]interface{}{
								"row": rowIdx,
								"col": colIdx,
							}
							// Value
							if cell.FormattedValue != "" {
								cellInfo["value"] = cell.FormattedValue
							}
							// Note
							if cell.Note != "" {
								cellInfo["note"] = cell.Note
							}
							// Hyperlink
							if cell.Hyperlink != "" {
								cellInfo["hyperlink"] = cell.Hyperlink
							}
							// Formatting
							if cell.EffectiveFormat != nil {
								fmt := cell.EffectiveFormat
								format := map[string]interface{}{}
								if fmt.BackgroundColor != nil {
									format["backgroundColor"] = colorToHex(fmt.BackgroundColor)
								}
								if fmt.TextFormat != nil {
									tf := map[string]interface{}{}
									if fmt.TextFormat.Bold {
										tf["bold"] = true
									}
									if fmt.TextFormat.Italic {
										tf["italic"] = true
									}
									if fmt.TextFormat.FontFamily != "" {
										tf["fontFamily"] = fmt.TextFormat.FontFamily
									}
									if fmt.TextFormat.FontSize > 0 {
										tf["fontSize"] = fmt.TextFormat.FontSize
									}
									if fmt.TextFormat.ForegroundColor != nil {
										tf["color"] = colorToHex(fmt.TextFormat.ForegroundColor)
									}
									if len(tf) > 0 {
										format["textFormat"] = tf
									}
								}
								if fmt.NumberFormat != nil {
									format["numberFormat"] = map[string]interface{}{
										"type":    fmt.NumberFormat.Type,
										"pattern": fmt.NumberFormat.Pattern,
									}
								}
								if fmt.HorizontalAlignment != "" {
									format["horizontalAlignment"] = fmt.HorizontalAlignment
								}
								if len(format) > 0 {
									cellInfo["format"] = format
								}
							}
							// Only include cells with some data
							if len(cellInfo) > 2 { // more than just row/col
								cells = append(cells, cellInfo)
							}
						}
						if len(cells) > 0 {
							rows = append(rows, map[string]interface{}{
								"rowIndex": rowIdx,
								"cells":    cells,
							})
						}
					}
					if len(rows) > 0 {
						grids = append(grids, map[string]interface{}{
							"startRow":    gridData.StartRow,
							"startColumn": gridData.StartColumn,
							"rows":        rows,
						})
					}
				}
				if len(grids) > 0 {
					sheetInfo["data"] = grids
				}
			}

			// Banding info
			if len(sheet.BandedRanges) > 0 {
				sheetInfo["hasBanding"] = true
			}

			sheetsData = append(sheetsData, sheetInfo)
		}
		result["sheets"] = sheetsData
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

	case "sheets_add_data_validation":
		var args struct {
			SpreadsheetID string   `json:"spreadsheet_id"`
			SheetID       float64  `json:"sheet_id"`
			Range         string   `json:"range"`
			Type          string   `json:"type"`
			Values        []string `json:"values"`
			Strict        *bool    `json:"strict"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		strict := true
		if args.Strict != nil {
			strict = *args.Strict
		}
		if err := h.client.SetDataValidation(args.SpreadsheetID, int64(args.SheetID), args.Range, args.Type, args.Values, strict); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status": "validation_set",
			"range":  args.Range,
			"type":   args.Type,
		}, nil

	case "sheets_add_conditional_formatting":
		var args struct {
			SpreadsheetID   string   `json:"spreadsheet_id"`
			SheetID         float64  `json:"sheet_id"`
			Range           string   `json:"range"`
			RuleType        string   `json:"rule_type"`
			Values          []string `json:"values"`
			BackgroundColor *string  `json:"background_color"`
			TextColor       *string  `json:"text_color"`
			Bold            bool     `json:"bold"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		var bgColor *sheets.Color
		if args.BackgroundColor != nil {
			var err error
			bgColor, err = hexToColor(*args.BackgroundColor)
			if err != nil {
				return nil, fmt.Errorf("invalid background_color: %w", err)
			}
		}
		var textColor *sheets.Color
		if args.TextColor != nil {
			var err error
			textColor, err = hexToColor(*args.TextColor)
			if err != nil {
				return nil, fmt.Errorf("invalid text_color: %w", err)
			}
		}
		if err := h.client.AddConditionalFormatting(args.SpreadsheetID, int64(args.SheetID), args.Range, args.RuleType, args.Values, bgColor, textColor, args.Bold); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":   "formatting_added",
			"range":    args.Range,
			"ruleType": args.RuleType,
		}, nil

	case "sheets_sort_range":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			SheetID       float64 `json:"sheet_id"`
			Range         string  `json:"range"`
			SortColumn    float64 `json:"sort_column"`
			Ascending     *bool   `json:"ascending"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		ascending := true
		if args.Ascending != nil {
			ascending = *args.Ascending
		}
		if err := h.client.SortRange(args.SpreadsheetID, int64(args.SheetID), args.Range, int64(args.SortColumn), ascending); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":    "sorted",
			"range":     args.Range,
			"column":    args.SortColumn,
			"ascending": ascending,
		}, nil

	case "sheets_merge_cells":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			SheetID       float64 `json:"sheet_id"`
			Range         string  `json:"range"`
			MergeType     string  `json:"merge_type"`
			Unmerge       bool    `json:"unmerge"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		mergeType := args.MergeType
		if mergeType == "" {
			mergeType = "MERGE_ALL"
		}
		if err := h.client.MergeCells(args.SpreadsheetID, int64(args.SheetID), args.Range, mergeType, args.Unmerge); err != nil {
			return nil, err
		}
		action := "merged"
		if args.Unmerge {
			action = "unmerged"
		}
		return map[string]interface{}{
			"status": action,
			"range":  args.Range,
		}, nil

	case "sheets_copy_sheet":
		var args struct {
			SpreadsheetID            string  `json:"spreadsheet_id"`
			SheetID                  float64 `json:"sheet_id"`
			DestinationSpreadsheetID string  `json:"destination_spreadsheet_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		props, err := h.client.CopySheet(args.SpreadsheetID, int64(args.SheetID), args.DestinationSpreadsheetID)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":  "copied",
			"sheetId": props.SheetId,
			"title":   props.Title,
			"index":   props.Index,
		}, nil

	case "sheets_batch_update":
		var args struct {
			SpreadsheetID string          `json:"spreadsheet_id"`
			Requests      json.RawMessage `json:"requests"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		// Unmarshal the raw requests into sheets.Request objects
		var requests []*sheets.Request
		if err := json.Unmarshal(args.Requests, &requests); err != nil {
			return nil, fmt.Errorf("invalid requests format: %w", err)
		}
		resp, err := h.client.BatchUpdate(args.SpreadsheetID, requests)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":        "updated",
			"spreadsheetId": resp.SpreadsheetId,
			"totalReplies":  len(resp.Replies),
		}, nil

	case "sheets_find_replace":
		var args struct {
			SpreadsheetID   string   `json:"spreadsheet_id"`
			Find            string   `json:"find"`
			Replacement     string   `json:"replacement"`
			SheetID         *float64 `json:"sheet_id"`
			MatchCase       bool     `json:"match_case"`
			MatchEntireCell bool     `json:"match_entire_cell"`
			SearchByRegex   bool     `json:"search_by_regex"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		var sheetID *int64
		if args.SheetID != nil {
			id := int64(*args.SheetID)
			sheetID = &id
		}
		result, err := h.client.FindReplace(args.SpreadsheetID, args.Find, args.Replacement, sheetID, args.MatchCase, args.MatchEntireCell, args.SearchByRegex)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":             "replaced",
			"occurrencesChanged": result.OccurrencesChanged,
			"rowsChanged":        result.RowsChanged,
			"sheetsChanged":      result.SheetsChanged,
			"valuesChanged":      result.ValuesChanged,
			"formulasChanged":    result.FormulasChanged,
		}, nil

	case "sheets_set_column_width":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			SheetID       float64 `json:"sheet_id"`
			StartColumn   float64 `json:"start_column"`
			EndColumn     float64 `json:"end_column"`
			Width         float64 `json:"width"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		if err := h.client.SetColumnWidth(args.SpreadsheetID, int64(args.SheetID), int64(args.StartColumn), int64(args.EndColumn), int64(args.Width)); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":      "width_set",
			"startColumn": args.StartColumn,
			"endColumn":   args.EndColumn,
			"width":       args.Width,
		}, nil

	case "sheets_insert_rows":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			SheetID       float64 `json:"sheet_id"`
			StartIndex    float64 `json:"start_index"`
			NumRows       float64 `json:"num_rows"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		if err := h.client.InsertRows(args.SpreadsheetID, int64(args.SheetID), int64(args.StartIndex), int64(args.NumRows)); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":     "rows_inserted",
			"startIndex": args.StartIndex,
			"numRows":    args.NumRows,
		}, nil

	case "sheets_delete_rows":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			SheetID       float64 `json:"sheet_id"`
			StartIndex    float64 `json:"start_index"`
			EndIndex      float64 `json:"end_index"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		if err := h.client.DeleteRows(args.SpreadsheetID, int64(args.SheetID), int64(args.StartIndex), int64(args.EndIndex)); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":     "rows_deleted",
			"startIndex": args.StartIndex,
			"endIndex":   args.EndIndex,
		}, nil

	case "sheets_insert_columns":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			SheetID       float64 `json:"sheet_id"`
			StartIndex    float64 `json:"start_index"`
			NumColumns    float64 `json:"num_columns"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		if err := h.client.InsertColumns(args.SpreadsheetID, int64(args.SheetID), int64(args.StartIndex), int64(args.NumColumns)); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":     "columns_inserted",
			"startIndex": args.StartIndex,
			"numColumns": args.NumColumns,
		}, nil

	case "sheets_delete_columns":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			SheetID       float64 `json:"sheet_id"`
			StartIndex    float64 `json:"start_index"`
			EndIndex      float64 `json:"end_index"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		if err := h.client.DeleteColumns(args.SpreadsheetID, int64(args.SheetID), int64(args.StartIndex), int64(args.EndIndex)); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":     "columns_deleted",
			"startIndex": args.StartIndex,
			"endIndex":   args.EndIndex,
		}, nil

	case "sheets_batch_get_values":
		var args struct {
			SpreadsheetID string   `json:"spreadsheet_id"`
			Ranges        []string `json:"ranges"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		resp, err := h.client.BatchGetValues(args.SpreadsheetID, args.Ranges)
		if err != nil {
			return nil, err
		}
		valueRanges := make([]map[string]interface{}, len(resp.ValueRanges))
		for i, vr := range resp.ValueRanges {
			valueRanges[i] = map[string]interface{}{
				"range":          vr.Range,
				"majorDimension": vr.MajorDimension,
				"values":         vr.Values,
			}
		}
		return map[string]interface{}{
			"spreadsheetId": resp.SpreadsheetId,
			"valueRanges":   valueRanges,
		}, nil

	case "sheets_create_chart":
		var args struct {
			SpreadsheetID  string   `json:"spreadsheet_id"`
			SheetID        float64  `json:"sheet_id"`
			ChartType      string   `json:"chart_type"`
			DataRange      string   `json:"data_range"`
			Title          string   `json:"title"`
			PositionRow    *float64 `json:"position_row"`
			PositionCol    *float64 `json:"position_col"`
			LegendPosition string   `json:"legend_position"`
			Stacked        bool     `json:"stacked"`
			Width          *float64 `json:"width"`
			Height         *float64 `json:"height"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		var posRow, posCol int64
		if args.PositionRow != nil {
			posRow = int64(*args.PositionRow)
		}
		if args.PositionCol != nil {
			posCol = int64(*args.PositionCol)
		}
		var width, height int64
		if args.Width != nil {
			width = int64(*args.Width)
		}
		if args.Height != nil {
			height = int64(*args.Height)
		}
		chartResp, err := h.client.CreateChart(args.SpreadsheetID, int64(args.SheetID), args.ChartType, args.DataRange, args.Title, posRow, posCol, args.LegendPosition, args.Stacked, width, height)
		if err != nil {
			return nil, err
		}
		result := map[string]interface{}{
			"status": "chart_created",
		}
		if chartResp.Chart != nil {
			result["chartId"] = chartResp.Chart.ChartId
		}
		return result, nil

	case "sheets_freeze_columns":
		var args struct {
			SpreadsheetID string   `json:"spreadsheet_id"`
			SheetID       float64  `json:"sheet_id"`
			NumColumns    *float64 `json:"num_columns"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		numCols := int64(1) // default
		if args.NumColumns != nil {
			numCols = int64(*args.NumColumns)
		}
		if err := h.client.FreezeColumns(args.SpreadsheetID, int64(args.SheetID), numCols); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":        "frozen",
			"frozenColumns": numCols,
		}, nil

	case "sheets_add_named_range":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			Name          string  `json:"name"`
			SheetID       float64 `json:"sheet_id"`
			Range         string  `json:"range"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		namedRange, err := h.client.AddNamedRange(args.SpreadsheetID, args.Name, int64(args.SheetID), args.Range)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":       "named_range_created",
			"namedRangeId": namedRange.NamedRangeId,
			"name":         namedRange.Name,
		}, nil

	case "sheets_duplicate_sheet":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			SheetID       float64 `json:"sheet_id"`
			NewName       string  `json:"new_name"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		props, err := h.client.DuplicateSheet(args.SpreadsheetID, int64(args.SheetID), args.NewName)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":  "duplicated",
			"sheetId": props.SheetId,
			"title":   props.Title,
			"index":   props.Index,
		}, nil

	case "sheets_add_protected_range":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			SheetID       float64 `json:"sheet_id"`
			Range         string  `json:"range"`
			Description   string  `json:"description"`
			WarningOnly   bool    `json:"warning_only"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		protectedRange, err := h.client.AddProtectedRange(args.SpreadsheetID, int64(args.SheetID), args.Range, args.Description, args.WarningOnly)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":           "protected",
			"protectedRangeId": protectedRange.ProtectedRangeId,
			"range":            args.Range,
			"warningOnly":      args.WarningOnly,
		}, nil

	case "sheets_list_charts":
		var args struct {
			SpreadsheetID string   `json:"spreadsheet_id"`
			SheetID       *float64 `json:"sheet_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		var sheetID *int64
		if args.SheetID != nil {
			id := int64(*args.SheetID)
			sheetID = &id
		}
		charts, err := h.client.ListCharts(args.SpreadsheetID, sheetID)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"charts": charts,
			"count":  len(charts),
		}, nil

	case "sheets_delete_chart":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			ChartID       float64 `json:"chart_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		if err := h.client.DeleteChart(args.SpreadsheetID, int64(args.ChartID)); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":  "deleted",
			"chartId": int64(args.ChartID),
		}, nil

	case "sheets_update_chart_position":
		var args struct {
			SpreadsheetID string   `json:"spreadsheet_id"`
			ChartID       float64  `json:"chart_id"`
			SheetID       *float64 `json:"sheet_id"`
			AnchorRow     *float64 `json:"anchor_row"`
			AnchorCol     *float64 `json:"anchor_col"`
			Width         *float64 `json:"width"`
			Height        *float64 `json:"height"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		var sheetID, anchorRow, anchorCol, width, height *int64
		if args.SheetID != nil {
			v := int64(*args.SheetID)
			sheetID = &v
		}
		if args.AnchorRow != nil {
			v := int64(*args.AnchorRow)
			anchorRow = &v
		}
		if args.AnchorCol != nil {
			v := int64(*args.AnchorCol)
			anchorCol = &v
		}
		if args.Width != nil {
			v := int64(*args.Width)
			width = &v
		}
		if args.Height != nil {
			v := int64(*args.Height)
			height = &v
		}
		if err := h.client.UpdateChartPosition(args.SpreadsheetID, int64(args.ChartID), sheetID, anchorRow, anchorCol, width, height); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":  "position_updated",
			"chartId": int64(args.ChartID),
		}, nil

	case "sheets_update_chart":
		var args struct {
			SpreadsheetID string   `json:"spreadsheet_id"`
			ChartID       float64  `json:"chart_id"`
			Title         *string  `json:"title"`
			ChartType     *string  `json:"chart_type"`
			DataRange     *string  `json:"data_range"`
			SheetID       *float64 `json:"sheet_id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		var dataRangeSheetID *int64
		if args.SheetID != nil {
			v := int64(*args.SheetID)
			dataRangeSheetID = &v
		}
		if err := h.client.UpdateChartSpec(args.SpreadsheetID, int64(args.ChartID), args.Title, args.ChartType, args.DataRange, dataRangeSheetID); err != nil {
			return nil, err
		}
		result := map[string]interface{}{
			"status":  "chart_updated",
			"chartId": int64(args.ChartID),
		}
		if args.Title != nil {
			result["newTitle"] = *args.Title
		}
		if args.ChartType != nil {
			result["newChartType"] = *args.ChartType
		}
		if args.DataRange != nil {
			result["newDataRange"] = *args.DataRange
		}
		return result, nil

	case "sheets_set_number_format":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			SheetID       float64 `json:"sheet_id"`
			Range         string  `json:"range"`
			FormatType    string  `json:"format_type"`
			Pattern       string  `json:"pattern"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		if err := h.client.SetNumberFormat(args.SpreadsheetID, int64(args.SheetID), args.Range, args.FormatType, args.Pattern); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":     "number_format_set",
			"range":      args.Range,
			"formatType": args.FormatType,
			"pattern":    args.Pattern,
		}, nil

	case "sheets_add_note":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			SheetID       float64 `json:"sheet_id"`
			Range         string  `json:"range"`
			Note          string  `json:"note"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		if err := h.client.AddNote(args.SpreadsheetID, int64(args.SheetID), args.Range, args.Note); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status": "note_added",
			"range":  args.Range,
		}, nil

	case "sheets_set_sheet_visibility":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			SheetID       float64 `json:"sheet_id"`
			Hidden        bool    `json:"hidden"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		if err := h.client.SetSheetVisibility(args.SpreadsheetID, int64(args.SheetID), args.Hidden); err != nil {
			return nil, err
		}
		status := "shown"
		if args.Hidden {
			status = "hidden"
		}
		return map[string]interface{}{
			"status": status,
		}, nil

	case "sheets_set_row_height":
		var args struct {
			SpreadsheetID string  `json:"spreadsheet_id"`
			SheetID       float64 `json:"sheet_id"`
			StartIndex    float64 `json:"start_index"`
			EndIndex      float64 `json:"end_index"`
			Height        float64 `json:"height"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		if err := h.client.SetRowHeight(args.SpreadsheetID, int64(args.SheetID), int64(args.StartIndex), int64(args.EndIndex), int64(args.Height)); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":     "row_height_set",
			"startIndex": args.StartIndex,
			"endIndex":   args.EndIndex,
			"height":     args.Height,
		}, nil

	case "sheets_batch_update_values":
		var args struct {
			SpreadsheetID string `json:"spreadsheet_id"`
			Data          []struct {
				Range  string          `json:"range"`
				Values [][]interface{} `json:"values"`
			} `json:"data"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		valueRanges := make([]sheets.ValueRange, len(args.Data))
		for i, d := range args.Data {
			valueRanges[i] = sheets.ValueRange{
				Range:  d.Range,
				Values: d.Values,
			}
		}
		resp, err := h.client.BatchUpdateValues(args.SpreadsheetID, valueRanges)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":                "values_updated",
			"spreadsheetId":         resp.SpreadsheetId,
			"totalUpdatedRows":      resp.TotalUpdatedRows,
			"totalUpdatedColumns":   resp.TotalUpdatedColumns,
			"totalUpdatedCells":     resp.TotalUpdatedCells,
			"totalUpdatedSheets":    resp.TotalUpdatedSheets,
		}, nil

	case "sheets_create_waterfall_chart":
		var args struct {
			SpreadsheetID   string    `json:"spreadsheet_id"`
			SheetID         float64   `json:"sheet_id"`
			DomainRange     string    `json:"domain_range"`
			DataRange       string    `json:"data_range"`
			Title           string    `json:"title"`
			SubtotalIndices []float64 `json:"subtotal_indices"`
			PositionRow     *float64  `json:"position_row"`
			PositionCol     *float64  `json:"position_col"`
			Width           *float64  `json:"width"`
			Height          *float64  `json:"height"`
			PositiveColor   *string   `json:"positive_color"`
			NegativeColor   *string   `json:"negative_color"`
			SubtotalColor   *string   `json:"subtotal_color"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		// Convert subtotal indices
		var subtotalIndices []int64
		for _, idx := range args.SubtotalIndices {
			subtotalIndices = append(subtotalIndices, int64(idx))
		}

		// Defaults for position and size
		var posRow, posCol int64
		if args.PositionRow != nil {
			posRow = int64(*args.PositionRow)
		}
		if args.PositionCol != nil {
			posCol = int64(*args.PositionCol)
		}
		width := int64(800)
		if args.Width != nil {
			width = int64(*args.Width)
		}
		height := int64(500)
		if args.Height != nil {
			height = int64(*args.Height)
		}

		// Parse colors with defaults
		positiveColorStr := "#4285f4"
		if args.PositiveColor != nil {
			positiveColorStr = *args.PositiveColor
		}
		positiveColor, err := hexToColor(positiveColorStr)
		if err != nil {
			return nil, fmt.Errorf("invalid positive_color: %w", err)
		}

		negativeColorStr := "#ea4335"
		if args.NegativeColor != nil {
			negativeColorStr = *args.NegativeColor
		}
		negativeColor, err := hexToColor(negativeColorStr)
		if err != nil {
			return nil, fmt.Errorf("invalid negative_color: %w", err)
		}

		subtotalColorStr := "#bfbfbf"
		if args.SubtotalColor != nil {
			subtotalColorStr = *args.SubtotalColor
		}
		subtotalColor, err := hexToColor(subtotalColorStr)
		if err != nil {
			return nil, fmt.Errorf("invalid subtotal_color: %w", err)
		}

		chartResp, err := h.client.CreateWaterfallChart(
			args.SpreadsheetID, int64(args.SheetID),
			args.DomainRange, args.DataRange, args.Title,
			subtotalIndices, posRow, posCol, width, height,
			positiveColor, negativeColor, subtotalColor,
		)
		if err != nil {
			return nil, err
		}
		result := map[string]interface{}{
			"status": "waterfall_chart_created",
		}
		if chartResp.Chart != nil {
			result["chartId"] = chartResp.Chart.ChartId
		}
		return result, nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// colorToHex converts a Google Sheets Color (float64 0-1) to hex string
func colorToHex(c *sheets.Color) string {
	if c == nil {
		return ""
	}
	r := int(c.Red * 255)
	g := int(c.Green * 255)
	b := int(c.Blue * 255)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// GetResources returns the available Sheets resources
func (h *Handler) GetResources() []server.Resource {
	return []server.Resource{}
}

// HandleResourceCall handles a resource call for Sheets service
func (h *Handler) HandleResourceCall(ctx context.Context, uri string) (interface{}, error) {
	return nil, fmt.Errorf("no resources available for sheets")
}
