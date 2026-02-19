package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/svaruag/mocli/internal/auth"
	"github.com/svaruag/mocli/internal/config"
	"github.com/svaruag/mocli/internal/exitcode"
	"github.com/svaruag/mocli/internal/secrets"
)

func runAuth(rt *runtimeState, args []string) int {
	if len(args) == 0 || isHelpToken(args[0]) {
		_, _ = fmt.Fprint(rt.stdout, groupHelp("auth"))
		return exitcode.Success
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	rest := args[1:]
	switch sub {
	case "credentials":
		return runAuthCredentials(rt, rest)
	case "add":
		return runAuthAdd(rt, rest)
	case "status":
		return runAuthStatus(rt)
	case "list":
		return runAuthList(rt)
	case "remove":
		return runAuthRemove(rt, rest)
	default:
		return rt.fail("unknown_command", fmt.Sprintf("unknown auth subcommand %q", sub), "Run 'mo auth help' for usage.", exitcode.UsageError)
	}
}

func runAuthCredentials(rt *runtimeState, args []string) int {
	if len(args) > 0 && strings.EqualFold(args[0], "list") {
		return runAuthCredentialsList(rt)
	}

	fs := flag.NewFlagSet("auth credentials", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	clientID := fs.String("client-id", "", "Override client_id")
	tenant := fs.String("tenant", "", "Override tenant")
	if err := fs.Parse(normalizeOnePositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid auth credentials flags", "Usage: mo auth credentials <path> [--client-id ...] [--tenant ...]"))
	}
	if fs.NArg() != 1 {
		return rt.failErr(usageError("credentials path is required", "Usage: mo auth credentials <path>"))
	}

	path := fs.Arg(0)
	data, err := os.ReadFile(path)
	if err != nil {
		return rt.failErr(usageError("failed to read credentials file", err.Error()))
	}

	cred, err := config.ParseCredentialsJSON(data)
	if err != nil {
		return rt.failErr(usageError("invalid credentials file", err.Error()))
	}
	if strings.TrimSpace(*clientID) != "" {
		cred.ClientID = strings.TrimSpace(*clientID)
	}
	if strings.TrimSpace(*tenant) != "" {
		cred.Tenant = strings.TrimSpace(*tenant)
	}
	cred.Normalize()
	if err := cred.Validate(); err != nil {
		return rt.failErr(usageError("invalid credentials", err.Error()))
	}

	cfg, err := config.LoadAppConfig()
	if err != nil {
		return rt.failErr(err)
	}
	client := rt.selectedClient(cfg)
	if err := config.SaveCredentials(client, cred); err != nil {
		return rt.failErr(err)
	}
	if cfg.DefaultClient == "" {
		cfg.DefaultClient = client
		_ = config.SaveAppConfig(cfg)
	}

	return rt.writeJSON(map[string]any{
		"saved":     true,
		"client":    client,
		"client_id": cred.ClientID,
		"tenant":    cred.Tenant,
	})
}

func runAuthCredentialsList(rt *runtimeState) int {
	base, err := config.BaseDir()
	if err != nil {
		return rt.failErr(err)
	}
	patterns := []string{
		filepath.Join(base, "credentials.json"),
		filepath.Join(base, "credentials-*.json"),
	}

	paths := []string{}
	for _, p := range patterns {
		matches, _ := filepath.Glob(p)
		paths = append(paths, matches...)
	}
	sort.Strings(paths)

	type item struct {
		Client   string `json:"client"`
		Tenant   string `json:"tenant"`
		ClientID string `json:"client_id"`
		Path     string `json:"path"`
	}
	items := make([]item, 0, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		cred, err := config.ParseCredentialsJSON(b)
		if err != nil {
			continue
		}
		client := "default"
		baseName := filepath.Base(p)
		if strings.HasPrefix(baseName, "credentials-") && strings.HasSuffix(baseName, ".json") {
			client = strings.TrimSuffix(strings.TrimPrefix(baseName, "credentials-"), ".json")
		}
		items = append(items, item{
			Client:   client,
			Tenant:   cred.Tenant,
			ClientID: cred.ClientID,
			Path:     p,
		})
	}

	return rt.writeJSON(map[string]any{"items": items})
}

func runAuthAdd(rt *runtimeState, args []string) int {
	fs := flag.NewFlagSet("auth add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	device := fs.Bool("device", false, "Device code flow")
	timeout := fs.Duration("timeout", 2*time.Minute, "Browser flow timeout")
	forceConsent := fs.Bool("force-consent", false, "Force consent prompt")
	if err := fs.Parse(normalizeOnePositionalArgs(args)); err != nil {
		return rt.failErr(usageError("invalid auth add flags", "Usage: mo auth add <email> [--device]"))
	}
	if fs.NArg() != 1 {
		return rt.failErr(usageError("email is required", "Usage: mo auth add <email>"))
	}
	email := strings.ToLower(strings.TrimSpace(fs.Arg(0)))
	if email == "" || !strings.Contains(email, "@") {
		return rt.failErr(usageError("invalid email", "Use a valid account email."))
	}

	cfg, err := config.LoadAppConfig()
	if err != nil {
		return rt.failErr(err)
	}
	client := rt.selectedClient(cfg)
	creds, err := config.LoadCredentials(client)
	if err != nil {
		return rt.failErr(authRequiredError(
			fmt.Sprintf("missing credentials for client %q", client),
			"Run 'mo auth credentials <path>' first.",
		))
	}
	store, backendInfo, err := secrets.OpenStore(rt.lookup, cfg)
	if err != nil {
		return rt.failErr(authRequiredError("could not open secrets backend", secretsBackendHint(err)))
	}
	if *device {
		return runAuthAddDeviceFlow(rt, cfg, store, backendInfo, creds, client, email)
	}

	return runAuthAddBrowserFlow(rt, cfg, store, backendInfo, creds, client, email, *forceConsent, *timeout)
}

func runAuthAddDeviceFlow(rt *runtimeState, cfg config.AppConfig, store *secrets.Store, backendInfo secrets.BackendInfo, creds config.Credentials, client, email string) int {
	startCtx, startCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer startCancel()

	start, err := auth.StartDeviceCode(startCtx, creds, auth.DefaultScopes, rt.lookup)
	if err != nil {
		return rt.failErr(authRequiredError("device authorization setup failed", err.Error()))
	}

	if strings.TrimSpace(start.Message) != "" {
		_, _ = fmt.Fprintln(rt.stderr, strings.TrimSpace(start.Message))
	} else {
		_, _ = fmt.Fprintf(rt.stderr, "Open %s and enter code %s\n", start.VerificationURI, start.UserCode)
	}

	waitFor := time.Duration(start.ExpiresIn+30) * time.Second
	if waitFor < time.Minute {
		waitFor = time.Minute
	}
	waitCtx, waitCancel := context.WithTimeout(context.Background(), waitFor)
	defer waitCancel()

	tok, err := auth.WaitForDeviceToken(waitCtx, creds, start, auth.DefaultScopes, rt.lookup)
	if err != nil {
		return rt.failErr(authRequiredError("device authorization failed", err.Error()))
	}
	return finalizeAuthAdd(rt, cfg, store, backendInfo, client, email, tok)
}

func runAuthAddBrowserFlow(rt *runtimeState, cfg config.AppConfig, store *secrets.Store, backendInfo secrets.BackendInfo, creds config.Credentials, client, email string, forceConsent bool, timeout time.Duration) int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return rt.failErr(err)
	}
	defer listener.Close()

	redirectURI := "http://" + listener.Addr().String() + "/oauth2/callback"
	session, err := auth.StartSession(creds, email, client, redirectURI, forceConsent, auth.DefaultScopes, rt.lookup)
	if err != nil {
		return rt.failErr(err)
	}

	resultCh := make(chan struct {
		Code  string
		State string
		Err   error
	}, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/callback", func(w http.ResponseWriter, r *http.Request) {
		code, state, err := auth.ParseRedirectURL("http://localhost" + r.URL.RequestURI())
		if err != nil {
			http.Error(w, "Authorization failed", http.StatusBadRequest)
			select {
			case resultCh <- struct {
				Code  string
				State string
				Err   error
			}{Err: err}:
			default:
			}
			return
		}
		_, _ = io.WriteString(w, "Authorization complete. You can close this tab.\n")
		select {
		case resultCh <- struct {
			Code  string
			State string
			Err   error
		}{Code: code, State: state}:
		default:
		}
	})

	srv := &http.Server{Handler: mux}
	go func() {
		_ = srv.Serve(listener)
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	if err := openBrowser(session.AuthURL); err != nil {
		_, _ = fmt.Fprintf(rt.stderr, "Could not open browser automatically: %v\n", err)
		return rt.failErr(authRequiredError("browser open failed", "Use --device on headless systems, or open a browser-capable environment."))
	}

	select {
	case result := <-resultCh:
		if result.Err != nil {
			return rt.failErr(authRequiredError("authorization callback failed", result.Err.Error()))
		}
		if result.State != session.State {
			return rt.failErr(usageError("state mismatch", "Restart auth flow and use the latest redirect URL."))
		}
		tok, err := auth.ExchangeAuthCode(context.Background(), creds, result.Code, session.RedirectURI, session.Verifier, auth.DefaultScopes, rt.lookup)
		if err != nil {
			return rt.failErr(authRequiredError("auth code exchange failed", err.Error()))
		}
		return finalizeAuthAdd(rt, cfg, store, backendInfo, client, email, tok)
	case <-time.After(timeout):
		return rt.failErr(usageError("authorization timed out", "Re-run auth add and complete browser sign-in, or use --device."))
	}
}

func finalizeAuthAdd(rt *runtimeState, cfg config.AppConfig, store *secrets.Store, backendInfo secrets.BackendInfo, client, requestedEmail string, tok auth.TokenResponse) int {
	if strings.TrimSpace(tok.RefreshToken) == "" {
		return rt.failErr(authRequiredError(
			"token response did not include a refresh token",
			"Re-run with --force-consent and ensure offline_access scope is granted.",
		))
	}

	resolvedEmail := requestedEmail
	if profileEmail, err := rt.graphGetMeEmail(tok.AccessToken); err == nil {
		resolvedEmail = profileEmail
	}

	if err := store.PutToken(client, resolvedEmail, secrets.Token{
		RefreshToken: tok.RefreshToken,
		Scope:        tok.Scope,
	}); err != nil {
		return rt.failErr(err)
	}

	cfg.UpsertAccount(client, resolvedEmail)
	if strings.TrimSpace(cfg.DefaultClient) == "" {
		cfg.DefaultClient = client
	}
	if strings.TrimSpace(cfg.DefaultAccount) == "" {
		cfg.DefaultAccount = resolvedEmail
	}
	if err := config.SaveAppConfig(cfg); err != nil {
		return rt.failErr(err)
	}

	return rt.writeJSON(map[string]any{
		"authorized": true,
		"account":    resolvedEmail,
		"client":     client,
		"scope":      tok.Scope,
		"backend":    backendInfo,
	})
}

func runAuthStatus(rt *runtimeState) int {
	cfg, err := config.LoadAppConfig()
	if err != nil {
		return rt.failErr(err)
	}

	info, resolveErr := secrets.ResolveBackend(rt.lookup, cfg)
	client := rt.selectedClient(cfg)
	account := rt.selectedAccount(cfg, client)

	hasCredentials := true
	credentialsSource := "file"
	if _, err := config.LoadCredentials(client); err != nil {
		hasCredentials = false
		credentialsSource = "missing"
	}
	tokenAvailable := false
	storeError := ""
	if account != "" {
		store, _, err := secrets.OpenStore(rt.lookup, cfg)
		if err != nil {
			storeError = err.Error()
		} else if _, err := store.GetToken(client, account); err == nil {
			tokenAvailable = true
		}
	}

	result := map[string]any{
		"client":             client,
		"account":            account,
		"default_account":    cfg.DefaultAccount,
		"default_client":     cfg.DefaultClient,
		"has_credentials":    hasCredentials,
		"credentials_source": credentialsSource,
		"token_available":    tokenAvailable,
		"accounts":           cfg.Accounts,
	}
	if resolveErr == nil {
		result["keyring_backend"] = info
	} else {
		result["keyring_backend_error"] = resolveErr.Error()
	}
	if storeError != "" {
		result["store_error"] = storeError
	}
	return rt.writeJSON(result)
}

func runAuthList(rt *runtimeState) int {
	cfg, err := config.LoadAppConfig()
	if err != nil {
		return rt.failErr(err)
	}
	client := rt.selectedClient(cfg)
	type item struct {
		Email   string `json:"email"`
		Client  string `json:"client"`
		Default bool   `json:"default"`
	}
	items := make([]item, 0, len(cfg.Accounts))
	for _, a := range cfg.Accounts {
		items = append(items, item{
			Email:   a.Email,
			Client:  a.Client,
			Default: a.Email == cfg.DefaultAccount,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Client == items[j].Client {
			return items[i].Email < items[j].Email
		}
		return items[i].Client < items[j].Client
	})
	return rt.writeJSON(map[string]any{
		"items":           items,
		"selected_client": client,
	})
}

func runAuthRemove(rt *runtimeState, args []string) int {
	if len(args) != 1 {
		return rt.failErr(usageError("email is required", "Usage: mo auth remove <email>"))
	}
	email := strings.ToLower(strings.TrimSpace(args[0]))
	if email == "" {
		return rt.failErr(usageError("invalid email", "Usage: mo auth remove <email>"))
	}

	cfg, err := config.LoadAppConfig()
	if err != nil {
		return rt.failErr(err)
	}
	client := rt.selectedClient(cfg)
	store, _, err := secrets.OpenStore(rt.lookup, cfg)
	if err != nil {
		return rt.failErr(authRequiredError("could not open secrets backend", secretsBackendHint(err)))
	}
	if err := store.DeleteToken(client, email); err != nil && !errors.Is(err, secrets.ErrNotFound) {
		return rt.failErr(err)
	}

	cfg.RemoveAccount(client, email)
	if err := config.SaveAppConfig(cfg); err != nil {
		return rt.failErr(err)
	}

	return rt.writeJSON(map[string]any{
		"removed": true,
		"email":   email,
		"client":  client,
	})
}

func openBrowser(targetURL string) error {
	commands := [][]string{
		{"xdg-open", targetURL},
		{"open", targetURL},
		{"cmd", "/c", "start", targetURL},
	}
	for _, c := range commands {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		cmd := exec.Command(c[0], c[1:]...)
		if err := cmd.Start(); err == nil {
			return nil
		}
	}
	return errors.New("could not open browser automatically")
}
