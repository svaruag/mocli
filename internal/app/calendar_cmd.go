package app

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/svaruag/mocli/internal/exitcode"
)

func runCalendar(rt *runtimeState, args []string) int {
	if len(args) == 0 || isHelpToken(args[0]) {
		_, _ = fmt.Fprint(rt.stdout, groupHelp("calendar"))
		return exitcode.Success
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	rest := args[1:]
	switch sub {
	case "list":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runCalendarList(rt, id, rest)
	case "create":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runCalendarCreate(rt, id, rest)
	case "update":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runCalendarUpdate(rt, id, rest)
	case "delete":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runCalendarDelete(rt, id, rest)
	default:
		return rt.failErr(usageError(fmt.Sprintf("unknown calendar subcommand %q", sub), "Run 'mo calendar help' for usage."))
	}
}

func runCalendarList(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("calendar list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	max := fs.Int("max", 50, "Max events")
	page := fs.String("page", "", "Page token")
	from := fs.String("from", "", "Start RFC3339")
	to := fs.String("to", "", "End RFC3339")
	if err := fs.Parse(args); err != nil {
		return rt.failErr(usageError("invalid calendar list flags", "Usage: mo calendar list [--from RFC3339 --to RFC3339] [--max N]"))
	}
	if fs.NArg() != 0 {
		return rt.failErr(usageError("calendar list does not take positional arguments", "Run 'mo calendar list --help'."))
	}
	if *max <= 0 || *max > 1000 {
		return rt.failErr(usageError("--max must be between 1 and 1000", "Use a value in range 1..1000."))
	}
	if strings.TrimSpace(*from) != "" {
		if _, err := time.Parse(time.RFC3339, *from); err != nil {
			return rt.failErr(usageError("invalid --from timestamp", "Use RFC3339 format."))
		}
	}
	if strings.TrimSpace(*to) != "" {
		if _, err := time.Parse(time.RFC3339, *to); err != nil {
			return rt.failErr(usageError("invalid --to timestamp", "Use RFC3339 format."))
		}
	}
	if (strings.TrimSpace(*from) == "") != (strings.TrimSpace(*to) == "") {
		return rt.failErr(usageError("--from and --to must be provided together", "Provide both --from and --to for calendarView queries."))
	}

	path := "/v1.0/me/events"
	q := url.Values{}
	q.Set("$top", fmt.Sprintf("%d", *max))
	q.Set("$select", "id,subject,start,end,location,webLink")
	q.Set("$orderby", "start/dateTime")
	if strings.TrimSpace(*from) != "" {
		path = "/v1.0/me/calendarView"
		q.Set("startDateTime", strings.TrimSpace(*from))
		q.Set("endDateTime", strings.TrimSpace(*to))
	}
	if strings.TrimSpace(*page) != "" {
		q.Set("$skiptoken", strings.TrimSpace(*page))
	}

	var resp struct {
		Value []map[string]any `json:"value"`
	}
	next, err := rt.graphRequest(id, "GET", path, q, nil, &resp)
	if err != nil {
		return rt.failErr(err)
	}

	if rt.globals.Plain {
		for _, it := range resp.Value {
			idv, _ := it["id"].(string)
			subj, _ := it["subject"].(string)
			_, _ = fmt.Fprintf(rt.stdout, "%s\t%s\n", idv, strings.ReplaceAll(subj, "\t", " "))
		}
		if next != "" {
			_, _ = fmt.Fprintf(rt.stdout, "next_page\t%s\n", next)
		}
		return exitcode.Success
	}

	return rt.writeJSON(map[string]any{"items": resp.Value, "next_page": next})
}

func runCalendarCreate(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("calendar create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	summary := fs.String("summary", "", "Event summary")
	from := fs.String("from", "", "Start RFC3339")
	to := fs.String("to", "", "End RFC3339")
	description := fs.String("description", "", "Description")
	location := fs.String("location", "", "Location")
	attendees := fs.String("attendees", "", "Comma-separated attendees")
	if err := fs.Parse(args); err != nil {
		return rt.failErr(usageError("invalid calendar create flags", "Usage: mo calendar create --summary <text> --from <rfc3339> --to <rfc3339>"))
	}
	if fs.NArg() != 0 {
		return rt.failErr(usageError("calendar create does not take positional arguments", "Run 'mo calendar create --help'."))
	}
	if strings.TrimSpace(*summary) == "" || strings.TrimSpace(*from) == "" || strings.TrimSpace(*to) == "" {
		return rt.failErr(usageError("--summary, --from, and --to are required", "Usage: mo calendar create --summary <text> --from <rfc3339> --to <rfc3339>"))
	}
	fromTime, err := time.Parse(time.RFC3339, *from)
	if err != nil {
		return rt.failErr(usageError("invalid --from timestamp", "Use RFC3339 format."))
	}
	toTime, err := time.Parse(time.RFC3339, *to)
	if err != nil {
		return rt.failErr(usageError("invalid --to timestamp", "Use RFC3339 format."))
	}
	if !fromTime.Before(toTime) {
		return rt.failErr(usageError("--from must be before --to", "Provide a valid event time range."))
	}

	payload := map[string]any{
		"subject": *summary,
		"start": map[string]any{
			"dateTime": fromTime.UTC().Format(time.RFC3339),
			"timeZone": "UTC",
		},
		"end": map[string]any{
			"dateTime": toTime.UTC().Format(time.RFC3339),
			"timeZone": "UTC",
		},
	}
	if strings.TrimSpace(*description) != "" {
		payload["body"] = map[string]any{"contentType": "Text", "content": *description}
	}
	if strings.TrimSpace(*location) != "" {
		payload["location"] = map[string]any{"displayName": *location}
	}
	if strings.TrimSpace(*attendees) != "" {
		recipients := emailRecipients(*attendees)
		at := make([]map[string]any, 0, len(recipients))
		for _, r := range recipients {
			at = append(at, map[string]any{"emailAddress": r["emailAddress"], "type": "required"})
		}
		payload["attendees"] = at
	}

	var out map[string]any
	_, err = rt.graphRequest(id, "POST", "/v1.0/me/events", nil, payload, &out)
	if err != nil {
		return rt.failErr(err)
	}
	return rt.writeJSON(out)
}

func runCalendarUpdate(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("calendar update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	summary := fs.String("summary", "", "Event summary")
	from := fs.String("from", "", "Start RFC3339")
	to := fs.String("to", "", "End RFC3339")
	description := fs.String("description", "", "Description")
	location := fs.String("location", "", "Location")
	attendees := fs.String("attendees", "", "Comma-separated attendees")
	if err := fs.Parse(normalizeOnePositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid calendar update flags", "Usage: mo calendar update <event-id> [--summary ...] [--from ...] [--to ...]"))
	}
	if fs.NArg() != 1 {
		return rt.failErr(usageError("event id is required", "Usage: mo calendar update <event-id> [--summary ...] [--from ...] [--to ...]"))
	}
	eventID := strings.TrimSpace(fs.Arg(0))
	if eventID == "" {
		return rt.failErr(usageError("event id is required", "Usage: mo calendar update <event-id> [--summary ...] [--from ...] [--to ...]"))
	}

	payload := map[string]any{}
	if strings.TrimSpace(*summary) != "" {
		payload["subject"] = *summary
	}
	if strings.TrimSpace(*from) != "" {
		t, err := time.Parse(time.RFC3339, *from)
		if err != nil {
			return rt.failErr(usageError("invalid --from timestamp", "Use RFC3339 format."))
		}
		payload["start"] = map[string]any{"dateTime": t.UTC().Format(time.RFC3339), "timeZone": "UTC"}
	}
	if strings.TrimSpace(*to) != "" {
		t, err := time.Parse(time.RFC3339, *to)
		if err != nil {
			return rt.failErr(usageError("invalid --to timestamp", "Use RFC3339 format."))
		}
		payload["end"] = map[string]any{"dateTime": t.UTC().Format(time.RFC3339), "timeZone": "UTC"}
	}
	if strings.TrimSpace(*description) != "" {
		payload["body"] = map[string]any{"contentType": "Text", "content": *description}
	}
	if strings.TrimSpace(*location) != "" {
		payload["location"] = map[string]any{"displayName": *location}
	}
	if strings.TrimSpace(*attendees) != "" {
		recipients := emailRecipients(*attendees)
		at := make([]map[string]any, 0, len(recipients))
		for _, r := range recipients {
			at = append(at, map[string]any{"emailAddress": r["emailAddress"], "type": "required"})
		}
		payload["attendees"] = at
	}
	if len(payload) == 0 {
		return rt.failErr(usageError("no update fields specified", "Provide at least one update flag."))
	}

	var out map[string]any
	_, err := rt.graphRequest(id, "PATCH", "/v1.0/me/events/"+url.PathEscape(eventID), nil, payload, &out)
	if err != nil {
		return rt.failErr(err)
	}
	if out == nil {
		return rt.writeJSON(map[string]any{"updated": true, "id": eventID})
	}
	return rt.writeJSON(out)
}

func runCalendarDelete(rt *runtimeState, id identityContext, args []string) int {
	if len(args) != 1 {
		return rt.failErr(usageError("event id is required", "Usage: mo calendar delete <event-id>"))
	}
	eventID := strings.TrimSpace(args[0])
	if eventID == "" {
		return rt.failErr(usageError("event id is required", "Usage: mo calendar delete <event-id>"))
	}
	ok, err := confirmAction(rt, "Delete calendar event?")
	if err != nil {
		return rt.failErr(err)
	}
	if !ok {
		return rt.writeJSON(map[string]any{"deleted": false, "id": eventID})
	}

	_, err = rt.graphRequest(id, "DELETE", "/v1.0/me/events/"+url.PathEscape(eventID), nil, nil, nil)
	if err != nil {
		return rt.failErr(err)
	}
	return rt.writeJSON(map[string]any{"deleted": true, "id": eventID})
}
