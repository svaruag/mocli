package app

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/svaruag/mocli/internal/config"
	"github.com/svaruag/mocli/internal/exitcode"
)

const driveSmallUploadLimitBytes = 250 * 1024 * 1024

func runDrive(rt *runtimeState, args []string) int {
	if len(args) == 0 || isHelpToken(args[0]) {
		_, _ = fmt.Fprint(rt.stdout, groupHelp("drive"))
		return exitcode.Success
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	rest := args[1:]
	switch sub {
	case "ls":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runDriveList(rt, id, rest)
	case "search":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runDriveSearch(rt, id, rest)
	case "get":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runDriveGet(rt, id, rest)
	case "upload":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runDriveUpload(rt, id, rest)
	case "download":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runDriveDownload(rt, id, rest)
	case "mkdir":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runDriveMkdir(rt, id, rest)
	case "rename":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runDriveRename(rt, id, rest)
	case "move":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runDriveMove(rt, id, rest)
	case "delete":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runDriveDelete(rt, id, rest)
	case "permissions":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runDrivePermissions(rt, id, rest)
	case "share":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runDriveShare(rt, id, rest)
	case "unshare":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runDriveUnshare(rt, id, rest)
	case "comments":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runDriveComments(rt, id, rest)
	case "comment":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runDriveComment(rt, id, rest)
	case "drives":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runDriveDrives(rt, id, rest)
	case "shared":
		id, err := rt.resolveIdentity()
		if err != nil {
			return rt.failErr(err)
		}
		return runDriveShared(rt, id, rest)
	default:
		return rt.failErr(usageError(fmt.Sprintf("unknown drive subcommand %q", sub), "Run 'mo drive help' for usage."))
	}
}

func runDriveList(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("drive ls", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	parent := fs.String("parent", "", "Parent folder item id")
	max := fs.Int("max", 100, "Max items")
	page := fs.String("page", "", "Page token")
	drive := fs.String("drive", "", "Drive container id")
	if err := fs.Parse(args); err != nil {
		return rt.failErr(usageError("invalid drive ls flags", "Usage: mo drive ls [--parent ID] [--max N] [--page TOKEN] [--drive DRIVE_ID]"))
	}
	if fs.NArg() != 0 {
		return rt.failErr(usageError("drive ls does not take positional arguments", "Run 'mo drive ls --help'."))
	}
	if *max <= 0 || *max > 1000 {
		return rt.failErr(usageError("--max must be between 1 and 1000", "Use a value in range 1..1000."))
	}

	base := driveBasePath(*drive)
	path := base + "/root/children"
	if strings.TrimSpace(*parent) != "" {
		path = base + "/items/" + url.PathEscape(strings.TrimSpace(*parent)) + "/children"
	}

	q := url.Values{}
	q.Set("$top", strconv.Itoa(*max))
	q.Set("$orderby", "name")
	q.Set("$select", "id,name,size,createdDateTime,lastModifiedDateTime,webUrl,file,folder,parentReference,deleted")
	if strings.TrimSpace(*page) != "" {
		q.Set("$skiptoken", strings.TrimSpace(*page))
	}

	var resp struct {
		Value []map[string]any `json:"value"`
	}
	next, err := rt.graphRequest(id, http.MethodGet, path, q, nil, &resp)
	if err != nil {
		return rt.failErr(err)
	}

	if rt.globals.Plain {
		for _, it := range resp.Value {
			_, _ = fmt.Fprintf(rt.stdout, "%s\t%s\t%s\t%d\n", asString(it["id"]), driveKind(it), strings.ReplaceAll(asString(it["name"]), "\t", " "), asInt64(it["size"]))
		}
		if next != "" {
			_, _ = fmt.Fprintf(rt.stdout, "next_page\t%s\n", next)
		}
		return exitcode.Success
	}

	return rt.writeJSON(map[string]any{"items": resp.Value, "next_page": next, "drive": strings.TrimSpace(*drive)})
}

func runDriveSearch(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("drive search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	max := fs.Int("max", 100, "Max items")
	page := fs.String("page", "", "Page token")
	drive := fs.String("drive", "", "Drive container id")
	if err := fs.Parse(normalizeOnePositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid drive search flags", "Usage: mo drive search <text> [--max N] [--page TOKEN] [--drive DRIVE_ID]"))
	}
	if fs.NArg() != 1 {
		return rt.failErr(usageError("search text is required", "Usage: mo drive search <text> [--max N] [--page TOKEN] [--drive DRIVE_ID]"))
	}
	if *max <= 0 || *max > 1000 {
		return rt.failErr(usageError("--max must be between 1 and 1000", "Use a value in range 1..1000."))
	}
	text := strings.TrimSpace(fs.Arg(0))
	if text == "" {
		return rt.failErr(usageError("search text is required", "Usage: mo drive search <text> [--max N] [--page TOKEN] [--drive DRIVE_ID]"))
	}

	base := driveBasePath(*drive)
	path := base + "/root/search(q='" + url.PathEscape(strings.ReplaceAll(text, "'", "''")) + "')"
	q := url.Values{}
	q.Set("$top", strconv.Itoa(*max))
	q.Set("$select", "id,name,size,createdDateTime,lastModifiedDateTime,webUrl,file,folder,parentReference,deleted")
	if strings.TrimSpace(*page) != "" {
		q.Set("$skiptoken", strings.TrimSpace(*page))
	}

	var resp struct {
		Value []map[string]any `json:"value"`
	}
	next, err := rt.graphRequest(id, http.MethodGet, path, q, nil, &resp)
	if err != nil {
		return rt.failErr(err)
	}
	return rt.writeJSON(map[string]any{"items": resp.Value, "next_page": next, "query": text, "drive": strings.TrimSpace(*drive)})
}

func runDriveGet(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("drive get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	drive := fs.String("drive", "", "Drive container id")
	if err := fs.Parse(normalizeOnePositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid drive get flags", "Usage: mo drive get <item-id> [--drive DRIVE_ID]"))
	}
	if fs.NArg() != 1 {
		return rt.failErr(usageError("item id is required", "Usage: mo drive get <item-id> [--drive DRIVE_ID]"))
	}
	itemID := strings.TrimSpace(fs.Arg(0))
	if itemID == "" {
		return rt.failErr(usageError("item id is required", "Usage: mo drive get <item-id> [--drive DRIVE_ID]"))
	}

	q := url.Values{}
	q.Set("$select", "id,name,size,createdDateTime,lastModifiedDateTime,webUrl,file,folder,parentReference,deleted")
	var out map[string]any
	_, err := rt.graphRequest(id, http.MethodGet, driveItemPath(*drive, itemID), q, nil, &out)
	if err != nil {
		return rt.failErr(err)
	}
	return rt.writeJSON(out)
}

func runDriveUpload(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("drive upload", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	parent := fs.String("parent", "", "Parent folder item id")
	name := fs.String("name", "", "Uploaded file name")
	conflict := fs.String("conflict", "fail", "Conflict behavior: fail|rename|replace")
	drive := fs.String("drive", "", "Drive container id")
	if err := fs.Parse(normalizeOnePositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid drive upload flags", "Usage: mo drive upload <local-path> [--parent ID] [--name NAME] [--conflict fail|rename|replace] [--drive DRIVE_ID]"))
	}
	if fs.NArg() != 1 {
		return rt.failErr(usageError("local path is required", "Usage: mo drive upload <local-path> [--parent ID] [--name NAME] [--conflict fail|rename|replace] [--drive DRIVE_ID]"))
	}
	localPath := strings.TrimSpace(fs.Arg(0))
	if localPath == "" {
		return rt.failErr(usageError("local path is required", "Usage: mo drive upload <local-path> [--parent ID] [--name NAME] [--conflict fail|rename|replace] [--drive DRIVE_ID]"))
	}
	behavior := strings.ToLower(strings.TrimSpace(*conflict))
	if behavior != "fail" && behavior != "rename" && behavior != "replace" {
		return rt.failErr(usageError("invalid --conflict", "Allowed values: fail, rename, replace"))
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		return rt.failErr(usageError("failed to read local file", err.Error()))
	}
	if len(data) > driveSmallUploadLimitBytes {
		return rt.failErr(usageError("file too large for simple upload", "Limit is 250MB for this command. Resumable upload is planned."))
	}

	resolvedName := strings.TrimSpace(*name)
	if resolvedName == "" {
		resolvedName = filepath.Base(localPath)
	}
	if resolvedName == "" || resolvedName == "." || resolvedName == string(os.PathSeparator) {
		return rt.failErr(usageError("invalid upload name", "Provide --name for this path."))
	}

	base := driveBasePath(*drive)
	createPath := base + "/root/children"
	if strings.TrimSpace(*parent) != "" {
		createPath = base + "/items/" + url.PathEscape(strings.TrimSpace(*parent)) + "/children"
	}
	payload := map[string]any{
		"name":                              resolvedName,
		"file":                              map[string]any{},
		"@microsoft.graph.conflictBehavior": behavior,
	}
	var created map[string]any
	_, err = rt.graphRequest(id, http.MethodPost, createPath, nil, payload, &created)
	if err != nil {
		return rt.failErr(err)
	}
	createdID := asString(created["id"])
	if createdID == "" {
		return rt.failErr(transientError("upload initialization failed", "graph response missing created item id"))
	}

	uploadPath := base + "/items/" + url.PathEscape(createdID) + "/content"
	rawResp, err := rt.driveRawRequest(id, http.MethodPut, uploadPath, nil, bytes.NewReader(data), "application/octet-stream")
	if err != nil {
		return rt.failErr(err)
	}
	defer rawResp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(rawResp.Body, 4<<20))
	if rawResp.StatusCode < 200 || rawResp.StatusCode >= 300 {
		return rt.failErr(graphErrorFromBody(rawResp.StatusCode, body))
	}

	var out map[string]any
	if len(body) > 0 {
		if err := json.Unmarshal(body, &out); err != nil {
			return rt.failErr(transientError("upload completed but response parse failed", err.Error()))
		}
	}
	if out == nil {
		out = created
	}
	out["local_path"] = localPath
	out["conflict"] = behavior
	return rt.writeJSON(out)
}

func runDriveDownload(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("drive download", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	outPath := fs.String("out", "", "Output file path")
	drive := fs.String("drive", "", "Drive container id")
	if err := fs.Parse(normalizeOnePositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid drive download flags", "Usage: mo drive download <item-id> [--out PATH] [--drive DRIVE_ID]"))
	}
	if fs.NArg() != 1 {
		return rt.failErr(usageError("item id is required", "Usage: mo drive download <item-id> [--out PATH] [--drive DRIVE_ID]"))
	}
	itemID := strings.TrimSpace(fs.Arg(0))
	if itemID == "" {
		return rt.failErr(usageError("item id is required", "Usage: mo drive download <item-id> [--out PATH] [--drive DRIVE_ID]"))
	}

	var meta map[string]any
	_, err := rt.graphRequest(id, http.MethodGet, driveItemPath(*drive, itemID), nil, nil, &meta)
	if err != nil {
		return rt.failErr(err)
	}
	name := asString(meta["name"])
	dest, err := resolveDriveDownloadPath(strings.TrimSpace(*outPath), name, itemID)
	if err != nil {
		return rt.failErr(usageError("invalid --out path", err.Error()))
	}
	if mkErr := os.MkdirAll(filepath.Dir(dest), 0o755); mkErr != nil {
		return rt.failErr(transientError("failed to create output directory", mkErr.Error()))
	}

	rawResp, err := rt.driveRawRequest(id, http.MethodGet, driveItemPath(*drive, itemID)+"/content", nil, nil, "")
	if err != nil {
		return rt.failErr(err)
	}
	defer rawResp.Body.Close()
	if rawResp.StatusCode < 200 || rawResp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(rawResp.Body, 4<<20))
		return rt.failErr(graphErrorFromBody(rawResp.StatusCode, body))
	}

	tmp := dest + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return rt.failErr(transientError("failed to create output file", err.Error()))
	}
	written, copyErr := io.Copy(f, rawResp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return rt.failErr(transientError("download failed", copyErr.Error()))
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return rt.failErr(transientError("failed to finalize output file", closeErr.Error()))
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return rt.failErr(transientError("failed to place output file", err.Error()))
	}

	return rt.writeJSON(map[string]any{"downloaded": true, "id": itemID, "path": dest, "bytes": written})
}

func runDriveMkdir(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("drive mkdir", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	parent := fs.String("parent", "", "Parent folder item id")
	drive := fs.String("drive", "", "Drive container id")
	if err := fs.Parse(normalizeOnePositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid drive mkdir flags", "Usage: mo drive mkdir <name> [--parent ID] [--drive DRIVE_ID]"))
	}
	if fs.NArg() != 1 {
		return rt.failErr(usageError("folder name is required", "Usage: mo drive mkdir <name> [--parent ID] [--drive DRIVE_ID]"))
	}
	name := strings.TrimSpace(fs.Arg(0))
	if name == "" {
		return rt.failErr(usageError("folder name is required", "Usage: mo drive mkdir <name> [--parent ID] [--drive DRIVE_ID]"))
	}

	base := driveBasePath(*drive)
	path := base + "/root/children"
	if strings.TrimSpace(*parent) != "" {
		path = base + "/items/" + url.PathEscape(strings.TrimSpace(*parent)) + "/children"
	}
	payload := map[string]any{
		"name":                              name,
		"folder":                            map[string]any{},
		"@microsoft.graph.conflictBehavior": "fail",
	}
	var out map[string]any
	_, err := rt.graphRequest(id, http.MethodPost, path, nil, payload, &out)
	if err != nil {
		return rt.failErr(err)
	}
	return rt.writeJSON(out)
}

func runDriveRename(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("drive rename", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	drive := fs.String("drive", "", "Drive container id")
	if err := fs.Parse(normalizeTwoPositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid drive rename flags", "Usage: mo drive rename <item-id> <new-name> [--drive DRIVE_ID]"))
	}
	if fs.NArg() != 2 {
		return rt.failErr(usageError("item id and new name are required", "Usage: mo drive rename <item-id> <new-name> [--drive DRIVE_ID]"))
	}
	itemID := strings.TrimSpace(fs.Arg(0))
	newName := strings.TrimSpace(fs.Arg(1))
	if itemID == "" || newName == "" {
		return rt.failErr(usageError("item id and new name are required", "Usage: mo drive rename <item-id> <new-name> [--drive DRIVE_ID]"))
	}

	payload := map[string]any{"name": newName}
	var out map[string]any
	_, err := rt.graphRequest(id, http.MethodPatch, driveItemPath(*drive, itemID), nil, payload, &out)
	if err != nil {
		return rt.failErr(err)
	}
	if out == nil {
		return rt.writeJSON(map[string]any{"renamed": true, "id": itemID, "name": newName})
	}
	return rt.writeJSON(out)
}

func runDriveMove(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("drive move", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	parent := fs.String("parent", "", "Destination parent folder item id")
	drive := fs.String("drive", "", "Drive container id")
	if err := fs.Parse(normalizeOnePositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid drive move flags", "Usage: mo drive move <item-id> --parent <dest-id> [--drive DRIVE_ID]"))
	}
	if fs.NArg() != 1 {
		return rt.failErr(usageError("item id is required", "Usage: mo drive move <item-id> --parent <dest-id> [--drive DRIVE_ID]"))
	}
	itemID := strings.TrimSpace(fs.Arg(0))
	dest := strings.TrimSpace(*parent)
	if itemID == "" || dest == "" {
		return rt.failErr(usageError("item id and --parent are required", "Usage: mo drive move <item-id> --parent <dest-id> [--drive DRIVE_ID]"))
	}

	payload := map[string]any{"parentReference": map[string]any{"id": dest}}
	var out map[string]any
	_, err := rt.graphRequest(id, http.MethodPatch, driveItemPath(*drive, itemID), nil, payload, &out)
	if err != nil {
		return rt.failErr(err)
	}
	if out == nil {
		return rt.writeJSON(map[string]any{"moved": true, "id": itemID, "parent": dest})
	}
	return rt.writeJSON(out)
}

func runDriveDelete(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("drive delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	permanent := fs.Bool("permanent", false, "Permanently delete")
	drive := fs.String("drive", "", "Drive container id")
	if err := fs.Parse(normalizeOnePositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid drive delete flags", "Usage: mo drive delete <item-id> [--permanent] [--drive DRIVE_ID]"))
	}
	if fs.NArg() != 1 {
		return rt.failErr(usageError("item id is required", "Usage: mo drive delete <item-id> [--permanent] [--drive DRIVE_ID]"))
	}
	itemID := strings.TrimSpace(fs.Arg(0))
	if itemID == "" {
		return rt.failErr(usageError("item id is required", "Usage: mo drive delete <item-id> [--permanent] [--drive DRIVE_ID]"))
	}
	ok, err := confirmAction(rt, "Delete drive item?")
	if err != nil {
		return rt.failErr(err)
	}
	if !ok {
		return rt.writeJSON(map[string]any{"deleted": false, "id": itemID, "permanent": *permanent})
	}

	if *permanent {
		_, err = rt.graphRequest(id, http.MethodPost, driveItemPath(*drive, itemID)+"/permanentDelete", nil, map[string]any{}, nil)
	} else {
		_, err = rt.graphRequest(id, http.MethodDelete, driveItemPath(*drive, itemID), nil, nil, nil)
	}
	if err != nil {
		return rt.failErr(err)
	}
	return rt.writeJSON(map[string]any{"deleted": true, "id": itemID, "permanent": *permanent})
}

func runDrivePermissions(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("drive permissions", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	max := fs.Int("max", 100, "Max permissions")
	page := fs.String("page", "", "Page token")
	drive := fs.String("drive", "", "Drive container id")
	if err := fs.Parse(normalizeOnePositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid drive permissions flags", "Usage: mo drive permissions <item-id> [--max N] [--page TOKEN] [--drive DRIVE_ID]"))
	}
	if fs.NArg() != 1 {
		return rt.failErr(usageError("item id is required", "Usage: mo drive permissions <item-id> [--max N] [--page TOKEN] [--drive DRIVE_ID]"))
	}
	if *max <= 0 || *max > 1000 {
		return rt.failErr(usageError("--max must be between 1 and 1000", "Use a value in range 1..1000."))
	}
	itemID := strings.TrimSpace(fs.Arg(0))
	if itemID == "" {
		return rt.failErr(usageError("item id is required", "Usage: mo drive permissions <item-id> [--max N] [--page TOKEN] [--drive DRIVE_ID]"))
	}

	q := url.Values{}
	q.Set("$top", strconv.Itoa(*max))
	if strings.TrimSpace(*page) != "" {
		q.Set("$skiptoken", strings.TrimSpace(*page))
	}
	var out struct {
		Value []map[string]any `json:"value"`
	}
	next, err := rt.graphRequest(id, http.MethodGet, driveItemPath(*drive, itemID)+"/permissions", q, nil, &out)
	if err != nil {
		return rt.failErr(err)
	}
	return rt.writeJSON(map[string]any{"item_id": itemID, "items": out.Value, "next_page": next})
}

func runDriveShare(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("drive share", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	to := fs.String("to", "", "Share target: user|domain|anyone")
	email := fs.String("email", "", "Recipient email (required for --to user)")
	domain := fs.String("domain", "", "Domain (required for --to domain)")
	role := fs.String("role", "read", "Role: read|write")
	sendInvite := fs.Bool("send-invite", false, "Send invitation email when supported")
	drive := fs.String("drive", "", "Drive container id")
	if err := fs.Parse(normalizeOnePositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid drive share flags", "Usage: mo drive share <item-id> --to user|domain|anyone [--email ...] [--domain ...] --role read|write [--send-invite] [--drive DRIVE_ID]"))
	}
	if fs.NArg() != 1 {
		return rt.failErr(usageError("item id is required", "Usage: mo drive share <item-id> --to user|domain|anyone [--email ...] [--domain ...] --role read|write [--send-invite] [--drive DRIVE_ID]"))
	}
	itemID := strings.TrimSpace(fs.Arg(0))
	if itemID == "" {
		return rt.failErr(usageError("item id is required", "Usage: mo drive share <item-id> --to user|domain|anyone [--email ...] [--domain ...] --role read|write [--send-invite] [--drive DRIVE_ID]"))
	}
	shareTo := strings.ToLower(strings.TrimSpace(*to))
	if shareTo != "user" && shareTo != "domain" && shareTo != "anyone" {
		return rt.failErr(usageError("invalid --to", "Allowed values: user, domain, anyone"))
	}
	shareRole := strings.ToLower(strings.TrimSpace(*role))
	if shareRole != "read" && shareRole != "write" {
		return rt.failErr(usageError("invalid --role", "Allowed values: read, write"))
	}

	baseItem := driveItemPath(*drive, itemID)
	switch shareTo {
	case "user":
		em := strings.TrimSpace(*email)
		if em == "" {
			return rt.failErr(usageError("--email is required for --to user", "Usage: mo drive share <item-id> --to user --email <addr> --role read|write"))
		}
		payload := map[string]any{
			"requireSignIn":              true,
			"sendInvitation":             *sendInvite,
			"roles":                      []string{shareRole},
			"recipients":                 []map[string]any{{"email": em}},
			"message":                    "",
			"retainInheritedPermissions": true,
		}
		var out map[string]any
		_, err := rt.graphRequest(id, http.MethodPost, baseItem+"/invite", nil, payload, &out)
		if err != nil {
			return rt.failErr(err)
		}
		if out == nil {
			out = map[string]any{}
		}
		out["shared"] = true
		out["to"] = shareTo
		out["role"] = shareRole
		out["item_id"] = itemID
		return rt.writeJSON(out)
	case "domain", "anyone":
		scope := "anonymous"
		if shareTo == "domain" {
			if strings.TrimSpace(*domain) == "" {
				return rt.failErr(usageError("--domain is required for --to domain", "Usage: mo drive share <item-id> --to domain --domain <value> --role read|write"))
			}
			scope = "organization"
		}
		typeVal := "view"
		if shareRole == "write" {
			typeVal = "edit"
		}
		payload := map[string]any{"type": typeVal, "scope": scope}
		var out map[string]any
		_, err := rt.graphRequest(id, http.MethodPost, baseItem+"/createLink", nil, payload, &out)
		if err != nil {
			return rt.failErr(err)
		}
		if out == nil {
			out = map[string]any{}
		}
		out["shared"] = true
		out["to"] = shareTo
		out["role"] = shareRole
		out["item_id"] = itemID
		if shareTo == "domain" {
			out["target_domain"] = strings.TrimSpace(*domain)
			out["note"] = "Graph createLink with organization scope is tenant-scoped; exact domain targeting may depend on tenant policy."
		}
		return rt.writeJSON(out)
	default:
		return rt.failErr(usageError("invalid --to", "Allowed values: user, domain, anyone"))
	}
}

func runDriveUnshare(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("drive unshare", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	drive := fs.String("drive", "", "Drive container id")
	if err := fs.Parse(normalizeTwoPositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid drive unshare flags", "Usage: mo drive unshare <item-id> <permission-id> [--drive DRIVE_ID]"))
	}
	if fs.NArg() != 2 {
		return rt.failErr(usageError("item id and permission id are required", "Usage: mo drive unshare <item-id> <permission-id> [--drive DRIVE_ID]"))
	}
	itemID := strings.TrimSpace(fs.Arg(0))
	permID := strings.TrimSpace(fs.Arg(1))
	if itemID == "" || permID == "" {
		return rt.failErr(usageError("item id and permission id are required", "Usage: mo drive unshare <item-id> <permission-id> [--drive DRIVE_ID]"))
	}
	ok, err := confirmAction(rt, "Remove sharing permission?")
	if err != nil {
		return rt.failErr(err)
	}
	if !ok {
		return rt.writeJSON(map[string]any{"unshared": false, "item_id": itemID, "permission_id": permID})
	}

	path := driveItemPath(*drive, itemID) + "/permissions/" + url.PathEscape(permID)
	_, err = rt.graphRequest(id, http.MethodDelete, path, nil, nil, nil)
	if err != nil {
		return rt.failErr(err)
	}
	return rt.writeJSON(map[string]any{"unshared": true, "item_id": itemID, "permission_id": permID})
}

func runDriveComments(rt *runtimeState, _ identityContext, args []string) int {
	fs := flag.NewFlagSet("drive comments", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	_ = fs.Int("max", 100, "Max comments")
	_ = fs.String("page", "", "Page token")
	_ = fs.String("drive", "", "Drive container id")
	if err := fs.Parse(normalizeOnePositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid drive comments flags", "Usage: mo drive comments <item-id> [--max N] [--page TOKEN] [--drive DRIVE_ID]"))
	}
	if fs.NArg() != 1 {
		return rt.failErr(usageError("item id is required", "Usage: mo drive comments <item-id> [--max N] [--page TOKEN] [--drive DRIVE_ID]"))
	}
	return rt.failErr(notImplementedError(
		"drive comments are not implemented",
		"Microsoft Graph v1.0 does not provide general file-comments endpoints for drive items. This command is reserved for future support.",
	))
}

func runDriveComment(rt *runtimeState, id identityContext, args []string) int {
	if len(args) == 0 || isHelpToken(args[0]) {
		return rt.failErr(usageError("comment subcommand is required", "Usage: mo drive comment add|delete ..."))
	}
	sub := strings.ToLower(strings.TrimSpace(args[0]))
	rest := args[1:]
	switch sub {
	case "add":
		return runDriveCommentAdd(rt, id, rest)
	case "delete":
		return runDriveCommentDelete(rt, id, rest)
	default:
		return rt.failErr(usageError(fmt.Sprintf("unknown drive comment subcommand %q", sub), "Usage: mo drive comment add|delete ..."))
	}
}

func runDriveCommentAdd(rt *runtimeState, _ identityContext, args []string) int {
	fs := flag.NewFlagSet("drive comment add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	_ = fs.String("text", "", "Comment text")
	_ = fs.String("drive", "", "Drive container id")
	if err := fs.Parse(normalizeOnePositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid drive comment add flags", "Usage: mo drive comment add <item-id> --text <value> [--drive DRIVE_ID]"))
	}
	if fs.NArg() != 1 {
		return rt.failErr(usageError("item id is required", "Usage: mo drive comment add <item-id> --text <value> [--drive DRIVE_ID]"))
	}
	return rt.failErr(notImplementedError(
		"drive comments are not implemented",
		"Microsoft Graph v1.0 does not provide general file-comments endpoints for drive items. This command is reserved for future support.",
	))
}

func runDriveCommentDelete(rt *runtimeState, _ identityContext, args []string) int {
	fs := flag.NewFlagSet("drive comment delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	_ = fs.String("drive", "", "Drive container id")
	if err := fs.Parse(normalizeTwoPositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid drive comment delete flags", "Usage: mo drive comment delete <item-id> <comment-id> [--drive DRIVE_ID]"))
	}
	if fs.NArg() != 2 {
		return rt.failErr(usageError("item id and comment id are required", "Usage: mo drive comment delete <item-id> <comment-id> [--drive DRIVE_ID]"))
	}
	return rt.failErr(notImplementedError(
		"drive comments are not implemented",
		"Microsoft Graph v1.0 does not provide general file-comments endpoints for drive items. This command is reserved for future support.",
	))
}

func runDriveDrives(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("drive drives", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	max := fs.Int("max", 100, "Max drives")
	page := fs.String("page", "", "Page token")
	if err := fs.Parse(args); err != nil {
		return rt.failErr(usageError("invalid drive drives flags", "Usage: mo drive drives [--max N] [--page TOKEN]"))
	}
	if fs.NArg() != 0 {
		return rt.failErr(usageError("drive drives does not take positional arguments", "Run 'mo drive drives --help'."))
	}
	if *max <= 0 || *max > 1000 {
		return rt.failErr(usageError("--max must be between 1 and 1000", "Use a value in range 1..1000."))
	}

	q := url.Values{}
	q.Set("$top", strconv.Itoa(*max))
	if strings.TrimSpace(*page) != "" {
		q.Set("$skiptoken", strings.TrimSpace(*page))
	}
	var out struct {
		Value []map[string]any `json:"value"`
	}
	next, err := rt.graphRequest(id, http.MethodGet, "/v1.0/me/drives", q, nil, &out)
	if err != nil {
		return rt.failErr(err)
	}
	return rt.writeJSON(map[string]any{"items": out.Value, "next_page": next})
}

func runDriveShared(rt *runtimeState, id identityContext, args []string) int {
	fs := flag.NewFlagSet("drive shared", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	max := fs.Int("max", 100, "Max shared items")
	page := fs.String("page", "", "Page token")
	if err := fs.Parse(args); err != nil {
		return rt.failErr(usageError("invalid drive shared flags", "Usage: mo drive shared [--max N] [--page TOKEN]"))
	}
	if fs.NArg() != 0 {
		return rt.failErr(usageError("drive shared does not take positional arguments", "Run 'mo drive shared --help'."))
	}
	if *max <= 0 || *max > 1000 {
		return rt.failErr(usageError("--max must be between 1 and 1000", "Use a value in range 1..1000."))
	}

	q := url.Values{}
	q.Set("$top", strconv.Itoa(*max))
	if strings.TrimSpace(*page) != "" {
		q.Set("$skiptoken", strings.TrimSpace(*page))
	}
	var out struct {
		Value []map[string]any `json:"value"`
	}
	next, err := rt.graphRequest(id, http.MethodGet, "/v1.0/me/drive/sharedWithMe", q, nil, &out)
	if err != nil {
		return rt.failErr(err)
	}
	return rt.writeJSON(map[string]any{
		"items":     out.Value,
		"next_page": next,
		"warning":   "Microsoft Graph sharedWithMe is deprecated (announced 2025-05-08) and may degrade until retirement.",
	})
}

func driveBasePath(driveID string) string {
	trimmed := strings.TrimSpace(driveID)
	if trimmed == "" {
		return "/v1.0/me/drive"
	}
	return "/v1.0/drives/" + url.PathEscape(trimmed)
}

func driveItemPath(driveID, itemID string) string {
	return driveBasePath(driveID) + "/items/" + url.PathEscape(strings.TrimSpace(itemID))
}

func driveKind(item map[string]any) string {
	if _, ok := item["folder"]; ok {
		return "folder"
	}
	if _, ok := item["file"]; ok {
		return "file"
	}
	return "item"
}

func asString(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func asInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case json.Number:
		n, _ := x.Int64()
		return n
	default:
		return 0
	}
}

func resolveDriveDownloadPath(outPath, itemName, itemID string) (string, error) {
	name := strings.TrimSpace(itemName)
	if name == "" {
		name = strings.TrimSpace(itemID)
	}
	if name == "" {
		name = "download.bin"
	}
	if strings.TrimSpace(outPath) == "" {
		return name, nil
	}
	if st, err := os.Stat(outPath); err == nil && st.IsDir() {
		return filepath.Join(outPath, name), nil
	}
	return outPath, nil
}

func (rt *runtimeState) driveRawRequest(id identityContext, method, path string, query url.Values, body io.Reader, contentType string) (*http.Response, error) {
	accessToken, err := rt.accessToken(id)
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(config.String(rt.lookup, "MO_GRAPH_BASE_URL", "https://graph.microsoft.com"), "/")
	u := base + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(contentType) != "" {
		req.Header.Set("Content-Type", contentType)
	}

	httpClient := &http.Client{Timeout: 60 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, transientError("graph request failed", err.Error())
	}
	return resp, nil
}

func graphErrorFromBody(status int, body []byte) error {
	var env graphErrorEnvelope
	_ = json.Unmarshal(body, &env)
	code := strings.TrimSpace(env.Error.Code)
	msg := strings.TrimSpace(env.Error.Message)
	if msg == "" {
		msg = strings.TrimSpace(string(body))
	}
	return mapGraphError(status, code, msg)
}
