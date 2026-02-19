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

func runMail(rt *runtimeState, args []string) int {
	if len(args) == 0 || isHelpToken(args[0]) {
		_, _ = fmt.Fprint(rt.stdout, groupHelp("mail"))
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
		return runMailList(rt, id, rest)
	case "get":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runMailGet(rt, id, rest)
	case "send":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runMailSend(rt, id, rest)
	default:
		return rt.failErr(usageError(fmt.Sprintf("unknown mail subcommand %q", sub), "Run 'mo mail help' for usage."))
	}
}

func runMailList(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("mail list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	max := fs.Int("max", 20, "Max messages")
	page := fs.String("page", "", "Page token")
	from := fs.String("from", "", "Filter from received time (RFC3339)")
	to := fs.String("to", "", "Filter to received time (RFC3339)")
	folder := fs.String("folder", "", "Mail folder id/name")
	if err := fs.Parse(args); err != nil {
		return rt.failErr(usageError("invalid mail list flags", "Usage: mo mail list [--max N] [--page TOKEN] [--from RFC3339] [--to RFC3339]"))
	}
	if fs.NArg() != 0 {
		return rt.failErr(usageError("mail list does not take positional arguments", "Run 'mo mail list --help'."))
	}
	if *max <= 0 || *max > 1000 {
		return rt.failErr(usageError("--max must be between 1 and 1000", "Use a value in range 1..1000."))
	}
	if strings.TrimSpace(*from) != "" {
		if _, err := time.Parse(time.RFC3339, *from); err != nil {
			return rt.failErr(usageError("invalid --from timestamp", "Use RFC3339 format, e.g. 2026-02-18T00:00:00Z."))
		}
	}
	if strings.TrimSpace(*to) != "" {
		if _, err := time.Parse(time.RFC3339, *to); err != nil {
			return rt.failErr(usageError("invalid --to timestamp", "Use RFC3339 format, e.g. 2026-02-19T00:00:00Z."))
		}
	}

	path := "/v1.0/me/messages"
	if strings.TrimSpace(*folder) != "" {
		path = "/v1.0/me/mailFolders/" + url.PathEscape(strings.TrimSpace(*folder)) + "/messages"
	}

	q := url.Values{}
	q.Set("$top", fmt.Sprintf("%d", *max))
	q.Set("$orderby", "receivedDateTime desc")
	q.Set("$select", "id,subject,from,receivedDateTime,isRead,bodyPreview")
	if strings.TrimSpace(*page) != "" {
		q.Set("$skiptoken", strings.TrimSpace(*page))
	}
	filters := make([]string, 0, 2)
	if strings.TrimSpace(*from) != "" {
		filters = append(filters, "receivedDateTime ge "+strings.TrimSpace(*from))
	}
	if strings.TrimSpace(*to) != "" {
		filters = append(filters, "receivedDateTime le "+strings.TrimSpace(*to))
	}
	if len(filters) > 0 {
		q.Set("$filter", strings.Join(filters, " and "))
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
			recv, _ := it["receivedDateTime"].(string)
			_, _ = fmt.Fprintf(rt.stdout, "%s\t%s\t%s\n", idv, recv, strings.ReplaceAll(subj, "\t", " "))
		}
		if strings.TrimSpace(next) != "" {
			_, _ = fmt.Fprintf(rt.stdout, "next_page\t%s\n", next)
		}
		return exitcode.Success
	}

	return rt.writeJSON(map[string]any{"items": resp.Value, "next_page": next})
}

func runMailGet(rt *runtimeState, id identityContext, args []string) int {
	if len(args) != 1 {
		return rt.failErr(usageError("message id is required", "Usage: mo mail get <message-id>"))
	}
	msgID := strings.TrimSpace(args[0])
	if msgID == "" {
		return rt.failErr(usageError("message id is required", "Usage: mo mail get <message-id>"))
	}

	q := url.Values{}
	q.Set("$select", "id,subject,from,toRecipients,ccRecipients,bccRecipients,body,bodyPreview,receivedDateTime,sentDateTime,isRead,internetMessageId")
	var out map[string]any
	_, err := rt.graphRequest(id, "GET", "/v1.0/me/messages/"+url.PathEscape(msgID), q, nil, &out)
	if err != nil {
		return rt.failErr(err)
	}
	return rt.writeJSON(out)
}

func runMailSend(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("mail send", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	to := fs.String("to", "", "Comma-separated recipients")
	cc := fs.String("cc", "", "Comma-separated cc recipients")
	bcc := fs.String("bcc", "", "Comma-separated bcc recipients")
	subject := fs.String("subject", "", "Email subject")
	body := fs.String("body", "", "Email body")
	html := fs.Bool("body-html", false, "Send body as HTML")
	saveSent := fs.Bool("save-to-sent", true, "Save to sent items")
	if err := fs.Parse(args); err != nil {
		return rt.failErr(usageError("invalid mail send flags", "Usage: mo mail send --to <emails> --subject <text> --body <text>"))
	}
	if fs.NArg() != 0 {
		return rt.failErr(usageError("mail send does not take positional arguments", "Run 'mo mail send --help'."))
	}
	if strings.TrimSpace(*to) == "" {
		return rt.failErr(usageError("--to is required", "Usage: mo mail send --to <emails> --subject <text> --body <text>"))
	}
	if strings.TrimSpace(*subject) == "" {
		return rt.failErr(usageError("--subject is required", "Usage: mo mail send --to <emails> --subject <text> --body <text>"))
	}
	if strings.TrimSpace(*body) == "" {
		return rt.failErr(usageError("--body is required", "Usage: mo mail send --to <emails> --subject <text> --body <text>"))
	}

	payload := map[string]any{
		"message": map[string]any{
			"subject": *subject,
			"body": map[string]any{
				"contentType": func() string {
					if *html {
						return "HTML"
					}
					return "Text"
				}(),
				"content": *body,
			},
			"toRecipients":  emailRecipients(*to),
			"ccRecipients":  emailRecipients(*cc),
			"bccRecipients": emailRecipients(*bcc),
		},
		"saveToSentItems": *saveSent,
	}

	_, err := rt.graphRequest(id, "POST", "/v1.0/me/sendMail", nil, payload, nil)
	if err != nil {
		return rt.failErr(err)
	}
	return rt.writeJSON(map[string]any{"status": "sent"})
}

func emailRecipients(csv string) []map[string]any {
	parts := strings.Split(csv, ",")
	out := make([]map[string]any, 0, len(parts))
	for _, part := range parts {
		email := strings.TrimSpace(part)
		if email == "" {
			continue
		}
		out = append(out, map[string]any{
			"emailAddress": map[string]any{"address": email},
		})
	}
	return out
}
