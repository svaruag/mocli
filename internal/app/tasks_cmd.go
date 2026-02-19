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

func runTasks(rt *runtimeState, args []string) int {
	if len(args) == 0 || isHelpToken(args[0]) {
		_, _ = fmt.Fprint(rt.stdout, groupHelp("tasks"))
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
		return runTasksList(rt, id, rest)
	case "create":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runTasksCreate(rt, id, rest)
	case "update":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runTasksUpdate(rt, id, rest)
	case "complete":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runTasksComplete(rt, id, rest)
	case "delete":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runTasksDelete(rt, id, rest)
	default:
		return rt.failErr(usageError(fmt.Sprintf("unknown tasks subcommand %q", sub), "Run 'mo tasks help' for usage."))
	}
}

func runTasksList(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("tasks list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	max := fs.Int("max", 100, "Max tasks")
	page := fs.String("page", "", "Page token")
	listID := fs.String("list-id", "", "To Do list id")
	if err := fs.Parse(args); err != nil {
		return rt.failErr(usageError("invalid tasks list flags", "Usage: mo tasks list [--list-id ID] [--max N] [--page TOKEN]"))
	}
	if fs.NArg() != 0 {
		return rt.failErr(usageError("tasks list does not take positional arguments", "Run 'mo tasks list --help'."))
	}
	if *max <= 0 || *max > 1000 {
		return rt.failErr(usageError("--max must be between 1 and 1000", "Use a value in range 1..1000."))
	}

	resolvedListID, err := resolveTodoListID(rt, id, *listID)
	if err != nil {
		return rt.failErr(err)
	}

	q := url.Values{}
	q.Set("$top", fmt.Sprintf("%d", *max))
	if strings.TrimSpace(*page) != "" {
		q.Set("$skiptoken", strings.TrimSpace(*page))
	}

	path := "/v1.0/me/todo/lists/" + url.PathEscape(resolvedListID) + "/tasks"
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
			title, _ := it["title"].(string)
			status, _ := it["status"].(string)
			_, _ = fmt.Fprintf(rt.stdout, "%s\t%s\t%s\n", idv, status, strings.ReplaceAll(title, "\t", " "))
		}
		if next != "" {
			_, _ = fmt.Fprintf(rt.stdout, "next_page\t%s\n", next)
		}
		return exitcode.Success
	}

	return rt.writeJSON(map[string]any{"list_id": resolvedListID, "items": resp.Value, "next_page": next})
}

func runTasksCreate(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("tasks create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	title := fs.String("title", "", "Task title")
	body := fs.String("body", "", "Task body")
	due := fs.String("due", "", "Due timestamp (RFC3339)")
	status := fs.String("status", "", "Task status")
	importance := fs.String("importance", "", "Importance (low|normal|high)")
	listID := fs.String("list-id", "", "To Do list id")
	if err := fs.Parse(args); err != nil {
		return rt.failErr(usageError("invalid tasks create flags", "Usage: mo tasks create --title <text> [--list-id ID]"))
	}
	if fs.NArg() != 0 {
		return rt.failErr(usageError("tasks create does not take positional arguments", "Run 'mo tasks create --help'."))
	}
	if strings.TrimSpace(*title) == "" {
		return rt.failErr(usageError("--title is required", "Usage: mo tasks create --title <text> [--list-id ID]"))
	}
	resolvedListID, err := resolveTodoListID(rt, id, *listID)
	if err != nil {
		return rt.failErr(err)
	}

	payload := map[string]any{"title": *title}
	if strings.TrimSpace(*body) != "" {
		payload["body"] = map[string]any{"content": *body, "contentType": "text"}
	}
	if strings.TrimSpace(*due) != "" {
		t, err := time.Parse(time.RFC3339, *due)
		if err != nil {
			return rt.failErr(usageError("invalid --due timestamp", "Use RFC3339 format."))
		}
		payload["dueDateTime"] = map[string]any{"dateTime": t.UTC().Format(time.RFC3339), "timeZone": "UTC"}
	}
	if strings.TrimSpace(*status) != "" {
		if !validTaskStatus(*status) {
			return rt.failErr(usageError("invalid --status", "Allowed: notStarted, inProgress, completed, waitingOnOthers, deferred"))
		}
		payload["status"] = strings.TrimSpace(*status)
	}
	if strings.TrimSpace(*importance) != "" {
		imp := strings.ToLower(strings.TrimSpace(*importance))
		if imp != "low" && imp != "normal" && imp != "high" {
			return rt.failErr(usageError("invalid --importance", "Allowed: low, normal, high"))
		}
		payload["importance"] = imp
	}

	path := "/v1.0/me/todo/lists/" + url.PathEscape(resolvedListID) + "/tasks"
	var out map[string]any
	_, err = rt.graphRequest(id, "POST", path, nil, payload, &out)
	if err != nil {
		return rt.failErr(err)
	}
	return rt.writeJSON(out)
}

func runTasksUpdate(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("tasks update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	title := fs.String("title", "", "Task title")
	body := fs.String("body", "", "Task body")
	due := fs.String("due", "", "Due timestamp (RFC3339)")
	status := fs.String("status", "", "Task status")
	importance := fs.String("importance", "", "Importance (low|normal|high)")
	listID := fs.String("list-id", "", "To Do list id")
	if err := fs.Parse(normalizeOnePositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid tasks update flags", "Usage: mo tasks update <task-id> [--title ...] [--status ...]"))
	}
	if fs.NArg() != 1 {
		return rt.failErr(usageError("task id is required", "Usage: mo tasks update <task-id> [--title ...] [--status ...]"))
	}
	taskID := strings.TrimSpace(fs.Arg(0))
	if taskID == "" {
		return rt.failErr(usageError("task id is required", "Usage: mo tasks update <task-id> [--title ...] [--status ...]"))
	}
	resolvedListID, err := resolveTodoListID(rt, id, *listID)
	if err != nil {
		return rt.failErr(err)
	}

	payload := map[string]any{}
	if strings.TrimSpace(*title) != "" {
		payload["title"] = *title
	}
	if strings.TrimSpace(*body) != "" {
		payload["body"] = map[string]any{"content": *body, "contentType": "text"}
	}
	if strings.TrimSpace(*due) != "" {
		t, err := time.Parse(time.RFC3339, *due)
		if err != nil {
			return rt.failErr(usageError("invalid --due timestamp", "Use RFC3339 format."))
		}
		payload["dueDateTime"] = map[string]any{"dateTime": t.UTC().Format(time.RFC3339), "timeZone": "UTC"}
	}
	if strings.TrimSpace(*status) != "" {
		if !validTaskStatus(*status) {
			return rt.failErr(usageError("invalid --status", "Allowed: notStarted, inProgress, completed, waitingOnOthers, deferred"))
		}
		payload["status"] = strings.TrimSpace(*status)
	}
	if strings.TrimSpace(*importance) != "" {
		imp := strings.ToLower(strings.TrimSpace(*importance))
		if imp != "low" && imp != "normal" && imp != "high" {
			return rt.failErr(usageError("invalid --importance", "Allowed: low, normal, high"))
		}
		payload["importance"] = imp
	}
	if len(payload) == 0 {
		return rt.failErr(usageError("no update fields specified", "Provide at least one update flag."))
	}

	path := "/v1.0/me/todo/lists/" + url.PathEscape(resolvedListID) + "/tasks/" + url.PathEscape(taskID)
	var out map[string]any
	_, err = rt.graphRequest(id, "PATCH", path, nil, payload, &out)
	if err != nil {
		return rt.failErr(err)
	}
	if out == nil {
		return rt.writeJSON(map[string]any{"updated": true, "id": taskID, "list_id": resolvedListID})
	}
	return rt.writeJSON(out)
}

func runTasksComplete(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("tasks complete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	listID := fs.String("list-id", "", "To Do list id")
	if err := fs.Parse(normalizeOnePositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid tasks complete flags", "Usage: mo tasks complete <task-id> [--list-id ID]"))
	}
	if fs.NArg() != 1 {
		return rt.failErr(usageError("task id is required", "Usage: mo tasks complete <task-id> [--list-id ID]"))
	}
	taskID := strings.TrimSpace(fs.Arg(0))
	if taskID == "" {
		return rt.failErr(usageError("task id is required", "Usage: mo tasks complete <task-id> [--list-id ID]"))
	}
	resolvedListID, err := resolveTodoListID(rt, id, *listID)
	if err != nil {
		return rt.failErr(err)
	}

	payload := map[string]any{
		"status": "completed",
		"completedDateTime": map[string]any{
			"dateTime": time.Now().UTC().Format(time.RFC3339),
			"timeZone": "UTC",
		},
	}
	path := "/v1.0/me/todo/lists/" + url.PathEscape(resolvedListID) + "/tasks/" + url.PathEscape(taskID)
	var out map[string]any
	_, err = rt.graphRequest(id, "PATCH", path, nil, payload, &out)
	if err != nil {
		return rt.failErr(err)
	}
	if out == nil {
		return rt.writeJSON(map[string]any{"completed": true, "id": taskID, "list_id": resolvedListID})
	}
	return rt.writeJSON(out)
}

func runTasksDelete(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("tasks delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	listID := fs.String("list-id", "", "To Do list id")
	if err := fs.Parse(normalizeOnePositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid tasks delete flags", "Usage: mo tasks delete <task-id> [--list-id ID]"))
	}
	if fs.NArg() != 1 {
		return rt.failErr(usageError("task id is required", "Usage: mo tasks delete <task-id> [--list-id ID]"))
	}
	taskID := strings.TrimSpace(fs.Arg(0))
	if taskID == "" {
		return rt.failErr(usageError("task id is required", "Usage: mo tasks delete <task-id> [--list-id ID]"))
	}
	resolvedListID, err := resolveTodoListID(rt, id, *listID)
	if err != nil {
		return rt.failErr(err)
	}
	ok, err := confirmAction(rt, "Delete task?")
	if err != nil {
		return rt.failErr(err)
	}
	if !ok {
		return rt.writeJSON(map[string]any{"deleted": false, "id": taskID, "list_id": resolvedListID})
	}

	path := "/v1.0/me/todo/lists/" + url.PathEscape(resolvedListID) + "/tasks/" + url.PathEscape(taskID)
	_, err = rt.graphRequest(id, "DELETE", path, nil, nil, nil)
	if err != nil {
		return rt.failErr(err)
	}
	return rt.writeJSON(map[string]any{"deleted": true, "id": taskID, "list_id": resolvedListID})
}

func resolveTodoListID(rt *runtimeState, id identityContext, listID string) (string, error) {
	if strings.TrimSpace(listID) != "" {
		return strings.TrimSpace(listID), nil
	}
	q := url.Values{}
	q.Set("$top", "1")
	var resp struct {
		Value []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"value"`
	}
	_, err := rt.graphRequest(id, "GET", "/v1.0/me/todo/lists", q, nil, &resp)
	if err != nil {
		return "", err
	}
	if len(resp.Value) == 0 || strings.TrimSpace(resp.Value[0].ID) == "" {
		return "", notFoundError("no To Do lists found", "Create a list in Microsoft To Do first, or pass --list-id.")
	}
	return resp.Value[0].ID, nil
}

func validTaskStatus(v string) bool {
	switch strings.TrimSpace(v) {
	case "notStarted", "inProgress", "completed", "waitingOnOthers", "deferred":
		return true
	default:
		return false
	}
}
