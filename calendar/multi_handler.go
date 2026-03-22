package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"go.ngs.io/google-mcp-server/auth"
	"go.ngs.io/google-mcp-server/server"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// MultiAccountHandler implements the ServiceHandler interface with multi-account support
type MultiAccountHandler struct {
	accountManager *auth.AccountManager
}

// NewMultiAccountHandler creates a new multi-account aware Calendar handler
func NewMultiAccountHandler(accountManager *auth.AccountManager) *MultiAccountHandler {
	return &MultiAccountHandler{
		accountManager: accountManager,
	}
}

// GetTools returns the available Calendar tools
func (h *MultiAccountHandler) GetTools() []server.Tool {
	// Return the same tools as the original handler
	return []server.Tool{
		{
			Name:        "calendar_list",
			Description: "List all accessible calendars",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"account": server.AccountProperty,
				},
			},
		},
		{
			Name:        "calendar_events_list",
			Description: "List events from a calendar with optional date range",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"calendar_id": {
						Type:        "string",
						Description: "Calendar ID (use 'primary' for main calendar)",
					},
					"time_min": {
						Type:        "string",
						Description: "Start time (RFC3339 format)",
					},
					"time_max": {
						Type:        "string",
						Description: "End time (RFC3339 format)",
					},
					"max_results": {
						Type:        "number",
						Description: "Maximum number of events to return",
					},
					"account": server.AccountProperty,
				},
				Required: []string{"calendar_id"},
			},
		},
		{
			Name:        "calendar_event_create",
			Description: "Create a new calendar event",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"calendar_id": {
						Type:        "string",
						Description: "Calendar ID (use 'primary' for main calendar)",
					},
					"summary": {
						Type:        "string",
						Description: "Event title",
					},
					"description": {
						Type:        "string",
						Description: "Event description",
					},
					"location": {
						Type:        "string",
						Description: "Event location",
					},
					"start_time": {
						Type:        "string",
						Description: "Start time (RFC3339 format)",
					},
					"end_time": {
						Type:        "string",
						Description: "End time (RFC3339 format)",
					},
					"attendees": {
						Type:        "array",
						Description: "List of attendee email addresses",
						Items: &server.Property{
							Type: "string",
						},
					},
					"reminders": {
						Type:        "array",
						Description: "List of reminder times in minutes",
						Items: &server.Property{
							Type: "number",
						},
					},
					"account": server.AccountProperty,
				},
				Required: []string{"calendar_id", "summary", "start_time", "end_time"},
			},
		},
		{
			Name:        "calendar_event_update",
			Description: "Update an existing calendar event",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"calendar_id": {
						Type:        "string",
						Description: "Calendar ID",
					},
					"event_id": {
						Type:        "string",
						Description: "Event ID",
					},
					"summary": {
						Type:        "string",
						Description: "Event title",
					},
					"description": {
						Type:        "string",
						Description: "Event description",
					},
					"location": {
						Type:        "string",
						Description: "Event location",
					},
					"start_time": {
						Type:        "string",
						Description: "Start time (RFC3339 format)",
					},
					"end_time": {
						Type:        "string",
						Description: "End time (RFC3339 format)",
					},
					"account": server.AccountProperty,
				},
				Required: []string{"calendar_id", "event_id"},
			},
		},
		{
			Name:        "calendar_event_delete",
			Description: "Delete a calendar event",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"calendar_id": {
						Type:        "string",
						Description: "Calendar ID",
					},
					"event_id": {
						Type:        "string",
						Description: "Event ID",
					},
					"account": server.AccountProperty,
				},
				Required: []string{"calendar_id", "event_id"},
			},
		},
		{
			Name:        "calendar_event_get",
			Description: "Get details of a specific calendar event",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"calendar_id": {
						Type:        "string",
						Description: "Calendar ID",
					},
					"event_id": {
						Type:        "string",
						Description: "Event ID",
					},
					"account": server.AccountProperty,
				},
				Required: []string{"calendar_id", "event_id"},
			},
		},
		{
			Name:        "calendar_freebusy_query",
			Description: "Query free/busy information for calendars",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"calendar_ids": {
						Type:        "array",
						Description: "List of calendar IDs to check",
						Items: &server.Property{
							Type: "string",
						},
					},
					"time_min": {
						Type:        "string",
						Description: "Start time (RFC3339 format)",
					},
					"time_max": {
						Type:        "string",
						Description: "End time (RFC3339 format)",
					},
					"account": server.AccountProperty,
				},
				Required: []string{"calendar_ids", "time_min", "time_max"},
			},
		},
		{
			Name:        "calendar_event_search",
			Description: "Search for events in a calendar by query text",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"calendar_id": {
						Type:        "string",
						Description: "Calendar ID",
					},
					"query": {
						Type:        "string",
						Description: "Search query text",
					},
					"time_min": {
						Type:        "string",
						Description: "Start time (RFC3339 format)",
					},
					"time_max": {
						Type:        "string",
						Description: "End time (RFC3339 format)",
					},
					"account": server.AccountProperty,
				},
				Required: []string{"calendar_id", "query"},
			},
		},
		{
			Name:        "calendar_find_free_time",
			Description: "Find available time slots in a date range. Gets all events and computes free gaps between them.",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"calendar_id": {
						Type:        "string",
						Description: "Calendar ID (use 'primary' for main calendar)",
					},
					"time_min": {
						Type:        "string",
						Description: "Start of range (RFC3339 format)",
					},
					"time_max": {
						Type:        "string",
						Description: "End of range (RFC3339 format)",
					},
					"min_duration_minutes": {
						Type:        "number",
						Description: "Minimum free slot duration in minutes (default: 30)",
					},
					"working_hours_start": {
						Type:        "number",
						Description: "Start of working hours (0-23, default: 9)",
					},
					"working_hours_end": {
						Type:        "number",
						Description: "End of working hours (0-23, default: 17)",
					},
					"account": server.AccountProperty,
				},
				Required: []string{"calendar_id", "time_min", "time_max"},
			},
		},
		{
			Name:        "calendar_event_respond",
			Description: "Respond to a calendar event invitation (accept, decline, or tentative)",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"calendar_id": {
						Type:        "string",
						Description: "Calendar ID",
					},
					"event_id": {
						Type:        "string",
						Description: "Event ID",
					},
					"response": {
						Type:        "string",
						Description: "Response status",
						Enum:        []string{"accepted", "declined", "tentative"},
					},
					"account": server.AccountProperty,
				},
				Required: []string{"calendar_id", "event_id", "response"},
			},
		},
		{
			Name:        "calendar_events_list_all_accounts",
			Description: "List events from all authenticated accounts for today or specified date range",
			InputSchema: server.InputSchema{
				Type: "object",
				Properties: map[string]server.Property{
					"time_min": {
						Type:        "string",
						Description: "Start time (RFC3339 format, defaults to today)",
					},
					"time_max": {
						Type:        "string",
						Description: "End time (RFC3339 format, defaults to end of today)",
					},
					"max_results": {
						Type:        "number",
						Description: "Maximum number of events per account",
					},
				},
			},
		},
	}
}

// HandleToolCall handles a tool call
func (h *MultiAccountHandler) HandleToolCall(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	switch name {
	case "calendar_list":
		return h.handleCalendarList(ctx, arguments)
	case "calendar_events_list":
		return h.handleEventsList(ctx, arguments)
	case "calendar_event_create":
		return h.handleEventCreate(ctx, arguments)
	case "calendar_event_update":
		return h.handleEventUpdate(ctx, arguments)
	case "calendar_event_delete":
		return h.handleEventDelete(ctx, arguments)
	case "calendar_event_get":
		return h.handleEventGet(ctx, arguments)
	case "calendar_freebusy_query":
		return h.handleFreeBusyQuery(ctx, arguments)
	case "calendar_event_search":
		return h.handleEventSearch(ctx, arguments)
	case "calendar_find_free_time":
		return h.handleFindFreeTime(ctx, arguments)
	case "calendar_event_respond":
		return h.handleEventRespond(ctx, arguments)
	case "calendar_events_list_all_accounts":
		return h.handleEventsListAllAccounts(ctx, arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// getClientForAccount gets or creates a calendar client for the specified account
func (h *MultiAccountHandler) getClientForAccount(ctx context.Context, email string) (*Client, error) {

	// Get account from manager
	account, err := h.accountManager.GetAccount(email)
	if err != nil {
		if email == "" {
			account, err = h.accountManager.ResolveAccount(ctx, "")
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Create calendar service for this account
	service, err := calendar.NewService(ctx, option.WithHTTPClient(account.OAuthClient.GetHTTPClient()))
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	return &Client{service: service}, nil
}

// handleCalendarList lists calendars for the specified account
func (h *MultiAccountHandler) handleCalendarList(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var args struct {
		Account string `json:"account"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	client, err := h.getClientForAccount(ctx, args.Account)
	if err != nil {
		return nil, err
	}

	calendars, err := client.ListCalendars()
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}

	return map[string]interface{}{
		"calendars": calendars,
	}, nil
}

// handleEventsList lists events for the specified account
func (h *MultiAccountHandler) handleEventsList(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var args struct {
		CalendarID string `json:"calendar_id"`
		TimeMin    string `json:"time_min"`
		TimeMax    string `json:"time_max"`
		MaxResults int64  `json:"max_results"`
		Account    string `json:"account"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Determine which account to use based on calendar_id
	accountEmail := args.Account
	if accountEmail == "" && strings.Contains(args.CalendarID, "@") {
		// Try to match account based on calendar ID
		accounts := h.accountManager.ListAccounts()
		for _, acc := range accounts {
			if strings.Contains(args.CalendarID, acc.Email) || args.CalendarID == acc.Email {
				accountEmail = acc.Email
				break
			}
		}
	}

	client, err := h.getClientForAccount(ctx, accountEmail)
	if err != nil {
		return nil, err
	}

	// If calendar_id looks like an email and matches an account, use "primary" instead
	calendarID := args.CalendarID
	if accountEmail != "" && args.CalendarID == accountEmail {
		calendarID = "primary"
	}

	// Parse time strings
	var timeMin, timeMax time.Time
	if args.TimeMin != "" {
		timeMin, _ = time.Parse(time.RFC3339, args.TimeMin)
	}
	if args.TimeMax != "" {
		timeMax, _ = time.Parse(time.RFC3339, args.TimeMax)
	}

	events, err := client.ListEvents(calendarID, timeMin, timeMax, args.MaxResults)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	return map[string]interface{}{
		"events":  events,
		"account": accountEmail,
	}, nil
}

// handleEventCreate creates an event
func (h *MultiAccountHandler) handleEventCreate(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var args struct {
		CalendarID  string   `json:"calendar_id"`
		Summary     string   `json:"summary"`
		Description string   `json:"description"`
		Location    string   `json:"location"`
		StartTime   string   `json:"start_time"`
		EndTime     string   `json:"end_time"`
		Attendees   []string `json:"attendees"`
		Reminders   []int    `json:"reminders"`
		Account     string   `json:"account"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	client, err := h.getClientForAccount(ctx, args.Account)
	if err != nil {
		return nil, err
	}

	// Create calendar event object
	event := &calendar.Event{
		Summary:     args.Summary,
		Description: args.Description,
		Location:    args.Location,
		Start: &calendar.EventDateTime{
			DateTime: args.StartTime,
		},
		End: &calendar.EventDateTime{
			DateTime: args.EndTime,
		},
	}

	// Add attendees if provided
	if len(args.Attendees) > 0 {
		var attendees []*calendar.EventAttendee
		for _, email := range args.Attendees {
			attendees = append(attendees, &calendar.EventAttendee{
				Email: email,
			})
		}
		event.Attendees = attendees
	}

	// Add reminders if provided
	if len(args.Reminders) > 0 {
		var overrides []*calendar.EventReminder
		for _, minutes := range args.Reminders {
			overrides = append(overrides, &calendar.EventReminder{
				Method:  "popup",
				Minutes: int64(minutes),
			})
		}
		event.Reminders = &calendar.EventReminders{
			UseDefault: false,
			Overrides:  overrides,
		}
	}

	createdEvent, err := client.CreateEvent(args.CalendarID, event)
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return map[string]interface{}{
		"event":   createdEvent,
		"message": fmt.Sprintf("Event '%s' created successfully", args.Summary),
	}, nil
}

// handleEventUpdate updates an existing event
func (h *MultiAccountHandler) handleEventUpdate(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var args struct {
		CalendarID  string `json:"calendar_id"`
		EventID     string `json:"event_id"`
		Summary     string `json:"summary"`
		Description string `json:"description"`
		Location    string `json:"location"`
		StartTime   string `json:"start_time"`
		EndTime     string `json:"end_time"`
		Account     string `json:"account"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	client, err := h.getClientForAccount(ctx, args.Account)
	if err != nil {
		return nil, err
	}

	// Get existing event
	event, err := client.GetEvent(args.CalendarID, args.EventID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if args.Summary != "" {
		event.Summary = args.Summary
	}
	if args.Description != "" {
		event.Description = args.Description
	}
	if args.Location != "" {
		event.Location = args.Location
	}
	if args.StartTime != "" {
		event.Start = &calendar.EventDateTime{
			DateTime: args.StartTime,
		}
	}
	if args.EndTime != "" {
		event.End = &calendar.EventDateTime{
			DateTime: args.EndTime,
		}
	}

	updated, err := client.UpdateEvent(args.CalendarID, args.EventID, event)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"event":   formatEvent(updated),
		"account": args.Account,
		"message": "Event updated successfully",
	}, nil
}

// handleEventDelete deletes a calendar event
func (h *MultiAccountHandler) handleEventDelete(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var args struct {
		CalendarID string `json:"calendar_id"`
		EventID    string `json:"event_id"`
		Account    string `json:"account"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	client, err := h.getClientForAccount(ctx, args.Account)
	if err != nil {
		return nil, err
	}

	if err := client.DeleteEvent(args.CalendarID, args.EventID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":   "deleted",
		"event_id": args.EventID,
		"account":  args.Account,
	}, nil
}

// handleEventGet gets details of a specific event
func (h *MultiAccountHandler) handleEventGet(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var args struct {
		CalendarID string `json:"calendar_id"`
		EventID    string `json:"event_id"`
		Account    string `json:"account"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	client, err := h.getClientForAccount(ctx, args.Account)
	if err != nil {
		return nil, err
	}

	event, err := client.GetEvent(args.CalendarID, args.EventID)
	if err != nil {
		return nil, err
	}

	result := formatEvent(event)
	result["account"] = args.Account
	return result, nil
}

// handleFreeBusyQuery queries free/busy information
func (h *MultiAccountHandler) handleFreeBusyQuery(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var args struct {
		CalendarIDs []string `json:"calendar_ids"`
		TimeMin     string   `json:"time_min"`
		TimeMax     string   `json:"time_max"`
		Account     string   `json:"account"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	client, err := h.getClientForAccount(ctx, args.Account)
	if err != nil {
		return nil, err
	}

	timeMin, err := time.Parse(time.RFC3339, args.TimeMin)
	if err != nil {
		return nil, fmt.Errorf("invalid time_min format: %w", err)
	}
	timeMax, err := time.Parse(time.RFC3339, args.TimeMax)
	if err != nil {
		return nil, fmt.Errorf("invalid time_max format: %w", err)
	}

	response, err := client.QueryFreeBusy(args.CalendarIDs, timeMin, timeMax)
	if err != nil {
		return nil, err
	}

	// Format the response
	result := make(map[string]interface{})
	result["timeMin"] = response.TimeMin
	result["timeMax"] = response.TimeMax
	result["account"] = args.Account

	calendars := make(map[string]interface{})
	for id, cal := range response.Calendars {
		calData := make(map[string]interface{})
		if cal.Errors != nil {
			calData["errors"] = cal.Errors
		}
		if cal.Busy != nil {
			busy := make([]map[string]string, len(cal.Busy))
			for i, period := range cal.Busy {
				busy[i] = map[string]string{
					"start": period.Start,
					"end":   period.End,
				}
			}
			calData["busy"] = busy
		}
		calendars[id] = calData
	}
	result["calendars"] = calendars

	return result, nil
}

// handleEventSearch searches for events
func (h *MultiAccountHandler) handleEventSearch(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var args struct {
		CalendarID string `json:"calendar_id"`
		Query      string `json:"query"`
		TimeMin    string `json:"time_min"`
		TimeMax    string `json:"time_max"`
		Account    string `json:"account"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	client, err := h.getClientForAccount(ctx, args.Account)
	if err != nil {
		return nil, err
	}

	var timeMin, timeMax time.Time
	if args.TimeMin != "" {
		timeMin, _ = time.Parse(time.RFC3339, args.TimeMin)
	}
	if args.TimeMax != "" {
		timeMax, _ = time.Parse(time.RFC3339, args.TimeMax)
	}

	events, err := client.SearchEvents(args.CalendarID, args.Query, timeMin, timeMax)
	if err != nil {
		return nil, err
	}

	eventList := make([]map[string]interface{}, len(events))
	for i, event := range events {
		eventList[i] = formatEvent(event)
	}

	return map[string]interface{}{
		"events":  eventList,
		"count":   len(eventList),
		"account": args.Account,
	}, nil
}

// handleFindFreeTime finds available time slots
func (h *MultiAccountHandler) handleFindFreeTime(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var args struct {
		CalendarID         string  `json:"calendar_id"`
		TimeMin            string  `json:"time_min"`
		TimeMax            string  `json:"time_max"`
		MinDurationMinutes float64 `json:"min_duration_minutes"`
		WorkingHoursStart  float64 `json:"working_hours_start"`
		WorkingHoursEnd    float64 `json:"working_hours_end"`
		Account            string  `json:"account"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Set defaults
	if args.MinDurationMinutes == 0 {
		args.MinDurationMinutes = 30
	}
	if args.WorkingHoursStart == 0 && args.WorkingHoursEnd == 0 {
		args.WorkingHoursStart = 9
		args.WorkingHoursEnd = 17
	}

	client, err := h.getClientForAccount(ctx, args.Account)
	if err != nil {
		return nil, err
	}

	rangeStart, err := time.Parse(time.RFC3339, args.TimeMin)
	if err != nil {
		return nil, fmt.Errorf("invalid time_min format: %w", err)
	}
	rangeEnd, err := time.Parse(time.RFC3339, args.TimeMax)
	if err != nil {
		return nil, fmt.Errorf("invalid time_max format: %w", err)
	}

	// Get all events in range
	events, err := client.ListEvents(args.CalendarID, rangeStart, rangeEnd, 250)
	if err != nil {
		return nil, err
	}

	// Parse event times and find gaps
	type busySlot struct {
		start time.Time
		end   time.Time
	}
	var busySlots []busySlot
	for _, event := range events {
		if event.Start == nil || event.End == nil {
			continue
		}
		var start, end time.Time
		if event.Start.DateTime != "" {
			start, _ = time.Parse(time.RFC3339, event.Start.DateTime)
		} else if event.Start.Date != "" {
			start, _ = time.Parse("2006-01-02", event.Start.Date)
		}
		if event.End.DateTime != "" {
			end, _ = time.Parse(time.RFC3339, event.End.DateTime)
		} else if event.End.Date != "" {
			end, _ = time.Parse("2006-01-02", event.End.Date)
		}
		if !start.IsZero() && !end.IsZero() {
			busySlots = append(busySlots, busySlot{start: start, end: end})
		}
	}

	// Find free slots day by day
	minDuration := time.Duration(args.MinDurationMinutes) * time.Minute
	var freeSlots []map[string]interface{}

	for day := rangeStart; day.Before(rangeEnd); day = day.AddDate(0, 0, 1) {
		dayStart := time.Date(day.Year(), day.Month(), day.Day(), int(args.WorkingHoursStart), 0, 0, 0, day.Location())
		dayEnd := time.Date(day.Year(), day.Month(), day.Day(), int(args.WorkingHoursEnd), 0, 0, 0, day.Location())

		if dayStart.Before(rangeStart) {
			dayStart = rangeStart
		}
		if dayEnd.After(rangeEnd) {
			dayEnd = rangeEnd
		}
		if dayStart.After(dayEnd) || dayStart.Equal(dayEnd) {
			continue
		}

		// Find busy periods for this day
		cursor := dayStart
		for _, slot := range busySlots {
			if slot.end.Before(dayStart) || slot.start.After(dayEnd) {
				continue
			}
			slotStart := slot.start
			if slotStart.Before(dayStart) {
				slotStart = dayStart
			}
			// Gap before this event
			if slotStart.After(cursor) && slotStart.Sub(cursor) >= minDuration {
				freeSlots = append(freeSlots, map[string]interface{}{
					"start":            cursor.Format(time.RFC3339),
					"end":              slotStart.Format(time.RFC3339),
					"duration_minutes": slotStart.Sub(cursor).Minutes(),
				})
			}
			if slot.end.After(cursor) {
				cursor = slot.end
			}
		}
		// Gap after last event
		if cursor.Before(dayEnd) && dayEnd.Sub(cursor) >= minDuration {
			freeSlots = append(freeSlots, map[string]interface{}{
				"start":            cursor.Format(time.RFC3339),
				"end":              dayEnd.Format(time.RFC3339),
				"duration_minutes": dayEnd.Sub(cursor).Minutes(),
			})
		}
	}

	return map[string]interface{}{
		"free_slots":           freeSlots,
		"count":                len(freeSlots),
		"min_duration_minutes": args.MinDurationMinutes,
		"working_hours":        fmt.Sprintf("%d:00-%d:00", int(args.WorkingHoursStart), int(args.WorkingHoursEnd)),
		"account":              args.Account,
	}, nil
}

// handleEventRespond responds to an event invitation
func (h *MultiAccountHandler) handleEventRespond(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var args struct {
		CalendarID string `json:"calendar_id"`
		EventID    string `json:"event_id"`
		Response   string `json:"response"`
		Account    string `json:"account"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	client, err := h.getClientForAccount(ctx, args.Account)
	if err != nil {
		return nil, err
	}

	// Get the event first
	event, err := client.GetEvent(args.CalendarID, args.EventID)
	if err != nil {
		return nil, err
	}

	// Find the user's attendee entry and update response status
	// Get user's email to match the attendee entry
	accountEmail := args.Account
	if accountEmail == "" {
		acct, err := h.accountManager.ResolveAccount(ctx, "")
		if err == nil && acct != nil {
			accountEmail = acct.Email
		}
	}

	found := false
	if event.Attendees != nil {
		for _, attendee := range event.Attendees {
			if attendee.Self || (accountEmail != "" && strings.EqualFold(attendee.Email, accountEmail)) {
				attendee.ResponseStatus = args.Response
				found = true
				break
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("could not find your attendee entry in this event. You may not be invited to this event")
	}

	// Update the event
	updated, err := client.UpdateEvent(args.CalendarID, args.EventID, event)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"event":    formatEvent(updated),
		"response": args.Response,
		"account":  args.Account,
		"message":  fmt.Sprintf("Event response set to '%s'", args.Response),
	}, nil
}

// handleEventsListAllAccounts lists events from all accounts
func (h *MultiAccountHandler) handleEventsListAllAccounts(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var args struct {
		TimeMin    string `json:"time_min"`
		TimeMax    string `json:"time_max"`
		MaxResults int64  `json:"max_results"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Default to today if no time range specified
	if args.TimeMin == "" {
		now := time.Now()
		loc, _ := time.LoadLocation("Asia/Tokyo")
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		args.TimeMin = todayStart.Format(time.RFC3339)
	}
	if args.TimeMax == "" {
		minTime, _ := time.Parse(time.RFC3339, args.TimeMin)
		args.TimeMax = minTime.Add(24 * time.Hour).Format(time.RFC3339)
	}
	if args.MaxResults == 0 {
		args.MaxResults = 50
	}

	// Get all accounts
	accounts := h.accountManager.ListAccounts()
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no authenticated accounts available")
	}

	// Collect events from all accounts
	allEvents := make(map[string]interface{})

	for _, account := range accounts {
		client, err := h.getClientForAccount(ctx, account.Email)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get client for %s: %v\n", account.Email, err)
			continue
		}

		// Parse time strings for this call
		timeMin, _ := time.Parse(time.RFC3339, args.TimeMin)
		timeMax, _ := time.Parse(time.RFC3339, args.TimeMax)

		// Get events from primary calendar
		events, err := client.ListEvents("primary", timeMin, timeMax, args.MaxResults)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to list events for %s: %v\n", account.Email, err)
			continue
		}

		if len(events) > 0 {
			allEvents[account.Email] = map[string]interface{}{
				"account_name": account.Name,
				"events":       events,
			}
		}
	}

	return map[string]interface{}{
		"accounts": allEvents,
		"time_range": map[string]string{
			"start": args.TimeMin,
			"end":   args.TimeMax,
		},
		"total_accounts": len(accounts),
	}, nil
}

// GetResources returns available resources
func (h *MultiAccountHandler) GetResources() []server.Resource {
	return []server.Resource{}
}

// HandleResourceCall handles resource calls
func (h *MultiAccountHandler) HandleResourceCall(ctx context.Context, uri string) (interface{}, error) {
	return nil, fmt.Errorf("no default client available")
}
