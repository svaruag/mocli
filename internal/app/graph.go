package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/svaruag/mocli/internal/auth"
	"github.com/svaruag/mocli/internal/config"
	"github.com/svaruag/mocli/internal/secrets"
)

type identityContext struct {
	Account string
	Client  string
	Creds   config.Credentials
	Store   *secrets.Store
	Cfg     config.AppConfig
}

type graphErrorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

const (
	defaultGraphBaseURL = "https://graph.microsoft.com"
	defaultAuthBaseURL  = "https://login.microsoftonline.com"
	maxRetryAfter       = 60 * time.Second
)

func (rt *runtimeState) selectedClient(cfg config.AppConfig) string {
	client := strings.TrimSpace(rt.globals.Client)
	if client != "" {
		return client
	}
	if strings.TrimSpace(cfg.DefaultClient) != "" {
		return cfg.DefaultClient
	}
	return "default"
}

func (rt *runtimeState) selectedAccount(cfg config.AppConfig, client string) string {
	if a := strings.ToLower(strings.TrimSpace(rt.globals.Account)); a != "" {
		return a
	}
	if a := strings.ToLower(strings.TrimSpace(cfg.DefaultAccount)); a != "" {
		return a
	}
	accounts := cfg.AccountsForClient(client)
	if len(accounts) == 1 {
		return strings.ToLower(strings.TrimSpace(accounts[0].Email))
	}
	return ""
}

func (rt *runtimeState) resolveIdentity() (identityContext, error) {
	cfg, err := config.LoadAppConfig()
	if err != nil {
		return identityContext{}, fmt.Errorf("load config: %w", err)
	}
	client := rt.selectedClient(cfg)
	account := rt.selectedAccount(cfg, client)
	if account == "" {
		return identityContext{}, authRequiredError(
			"no account selected",
			"Use --account or run 'mo auth add <email>' first.",
		)
	}

	creds, err := config.LoadCredentials(client)
	if err != nil {
		return identityContext{}, authRequiredError(
			fmt.Sprintf("missing credentials for client %q", client),
			"Run 'mo --client <name> auth credentials <path>' first.",
		)
	}

	store, _, err := secrets.OpenStore(rt.lookup, cfg)
	if err != nil {
		return identityContext{}, authRequiredError(
			"could not open secrets backend",
			secretsBackendHint(err),
		)
	}

	return identityContext{
		Account: account,
		Client:  client,
		Creds:   creds,
		Store:   store,
		Cfg:     cfg,
	}, nil
}

func (rt *runtimeState) accessToken(id identityContext) (string, error) {
	tok, err := id.Store.GetToken(id.Client, id.Account)
	if err != nil {
		if errors.Is(err, secrets.ErrNotFound) {
			return "", authRequiredError(
				fmt.Sprintf("no stored token for %s", id.Account),
				"Run 'mo auth add <email>' to authorize.",
			)
		}
		return "", fmt.Errorf("read token: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	refreshed, err := auth.RefreshAccessToken(ctx, id.Creds, tok.RefreshToken, auth.DefaultScopes, rt.lookup)
	if err != nil {
		return "", authRequiredError(
			"could not refresh access token",
			err.Error(),
		)
	}

	if strings.TrimSpace(refreshed.RefreshToken) != "" && refreshed.RefreshToken != tok.RefreshToken {
		if err := id.Store.PutToken(id.Client, id.Account, secrets.Token{
			RefreshToken: refreshed.RefreshToken,
			Scope:        refreshed.Scope,
		}); err != nil {
			return "", authRequiredError(
				"could not persist refreshed token",
				fmt.Sprintf("Refresh token rotated but save failed (%v). Re-authenticate with 'mo auth add %s' after fixing secrets backend.", err, id.Account),
			)
		}
	}
	return refreshed.AccessToken, nil
}

func (rt *runtimeState) graphRequest(id identityContext, method, path string, query url.Values, body any, out any) (string, error) {
	rt.warnEndpointOverrides()

	accessToken, err := rt.accessToken(id)
	if err != nil {
		return "", err
	}

	base := strings.TrimRight(config.String(rt.lookup, "MO_GRAPH_BASE_URL", defaultGraphBaseURL), "/")
	u := base + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var reqBody io.Reader
	var payload []byte
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("marshal request body: %w", err)
		}
		payload = data
		reqBody = bytes.NewReader(payload)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	var respBody []byte
	var statusCode int
	var graphCode string
	var graphMessage string

	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if payload != nil {
			reqBody = bytes.NewReader(payload)
		}

		req, err := http.NewRequest(method, u, reqBody)
		if err != nil {
			return "", fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			if attempt == maxAttempts {
				return "", transientError("graph request failed", err.Error())
			}
			time.Sleep(backoffDuration(attempt))
			continue
		}

		respBody, _ = io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		_ = resp.Body.Close()

		statusCode = resp.StatusCode
		var env graphErrorEnvelope
		_ = json.Unmarshal(respBody, &env)
		graphCode = strings.TrimSpace(env.Error.Code)
		graphMessage = strings.TrimSpace(env.Error.Message)
		if graphMessage == "" && statusCode >= 400 {
			graphMessage = strings.TrimSpace(string(respBody))
		}

		if statusCode >= 200 && statusCode < 300 {
			break
		}

		if shouldRetryStatus(statusCode) && attempt < maxAttempts {
			time.Sleep(retryDelay(resp, attempt))
			continue
		}
		return "", mapGraphError(statusCode, graphCode, graphMessage)
	}

	nextPage := ""
	var meta struct {
		NextLink string `json:"@odata.nextLink"`
	}
	if err := json.Unmarshal(respBody, &meta); err == nil {
		nextPage = extractPageToken(meta.NextLink)
	}

	if out != nil {
		if len(respBody) == 0 {
			return nextPage, nil
		}
		if err := json.Unmarshal(respBody, out); err != nil {
			return "", fmt.Errorf("parse response: %w", err)
		}
	}

	return nextPage, nil
}

func shouldRetryStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

func retryDelay(resp *http.Response, attempt int) time.Duration {
	if resp != nil {
		if v := strings.TrimSpace(resp.Header.Get("Retry-After")); v != "" {
			if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
				delay := time.Duration(secs) * time.Second
				if delay > maxRetryAfter {
					return maxRetryAfter
				}
				return delay
			}
		}
	}
	return backoffDuration(attempt)
}

func backoffDuration(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	// 300ms, 600ms, 1200ms...
	ms := 300.0 * math.Pow(2, float64(attempt-1))
	return time.Duration(ms) * time.Millisecond
}

func (rt *runtimeState) graphGetMeEmail(accessToken string) (string, error) {
	base := strings.TrimRight(config.String(rt.lookup, "MO_GRAPH_BASE_URL", defaultGraphBaseURL), "/")
	u := base + "/v1.0/me?$select=userPrincipalName,mail"

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	httpClient := &http.Client{Timeout: 20 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("profile request failed with status %d", resp.StatusCode)
	}

	var profile struct {
		UPN  string `json:"userPrincipalName"`
		Mail string `json:"mail"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&profile); err != nil {
		return "", err
	}
	email := strings.ToLower(strings.TrimSpace(profile.UPN))
	if email == "" {
		email = strings.ToLower(strings.TrimSpace(profile.Mail))
	}
	if email == "" {
		return "", errors.New("profile response missing email")
	}
	return email, nil
}

func (rt *runtimeState) warnEndpointOverrides() {
	if rt.endpointWarningsShown {
		return
	}
	rt.endpointWarningsShown = true

	rt.warnURLOverride("MO_AUTH_BASE_URL", defaultAuthBaseURL)
	rt.warnURLOverride("MO_GRAPH_BASE_URL", defaultGraphBaseURL)
}

func (rt *runtimeState) warnURLOverride(envKey, defaultValue string) {
	current := strings.TrimSpace(config.String(rt.lookup, envKey, defaultValue))
	if current == "" {
		return
	}
	if strings.TrimRight(current, "/") == strings.TrimRight(defaultValue, "/") {
		return
	}
	_, _ = fmt.Fprintf(rt.stderr, "warning: %s is set to non-default endpoint %s\n", envKey, current)
}

func mapGraphError(status int, graphCode, graphMessage string) error {
	switch {
	case status == http.StatusUnauthorized:
		return authRequiredError("graph request unauthorized", graphMessage)
	case status == http.StatusForbidden:
		return permissionError("graph request forbidden", graphMessage)
	case status == http.StatusNotFound:
		return notFoundError("graph resource not found", graphMessage)
	case status == http.StatusTooManyRequests || status >= 500:
		return transientError("graph transient failure", graphMessage)
	default:
		msg := strings.TrimSpace(graphMessage)
		if msg == "" {
			msg = fmt.Sprintf("graph request failed with status %d", status)
		}
		return usageError(msg, graphCode)
	}
}

func extractPageToken(nextLink string) string {
	nextLink = strings.TrimSpace(nextLink)
	if nextLink == "" {
		return ""
	}
	u, err := url.Parse(nextLink)
	if err != nil {
		return nextLink
	}
	q := u.Query()
	if v := strings.TrimSpace(q.Get("$skiptoken")); v != "" {
		return v
	}
	if v := strings.TrimSpace(q.Get("$skip")); v != "" {
		return v
	}
	return nextLink
}
