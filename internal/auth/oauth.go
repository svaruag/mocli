package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/svaruag/mocli/internal/config"
)

var DefaultScopes = []string{
	"openid",
	"profile",
	"offline_access",
	"User.Read",
	"Mail.Read",
	"Mail.Send",
	"Calendars.ReadWrite",
	"Tasks.ReadWrite",
	"Files.ReadWrite",
}

var newHTTPClient = func(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

type Session struct {
	State       string    `json:"state"`
	Verifier    string    `json:"verifier"`
	RedirectURI string    `json:"redirect_uri"`
	AuthURL     string    `json:"auth_url"`
	Email       string    `json:"email"`
	Client      string    `json:"client"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type TokenResponse struct {
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
	Message                 string `json:"message"`
}

type oauthErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type OAuthError struct {
	Code        string
	Description string
}

func (e *OAuthError) Error() string {
	if strings.TrimSpace(e.Description) == "" {
		return "oauth error: " + e.Code
	}
	return "oauth error: " + e.Code + " (" + e.Description + ")"
}

func StartSession(creds config.Credentials, email, client, redirectURI string, forceConsent bool, scopes []string, lookup config.LookupFunc) (Session, error) {
	if strings.TrimSpace(redirectURI) == "" {
		return Session{}, errors.New("redirect URI is required")
	}
	if len(scopes) == 0 {
		scopes = DefaultScopes
	}

	state, err := randToken(24)
	if err != nil {
		return Session{}, fmt.Errorf("generate state: %w", err)
	}
	verifier, err := randToken(48)
	if err != nil {
		return Session{}, fmt.Errorf("generate verifier: %w", err)
	}
	challenge := codeChallenge(verifier)

	authBase := strings.TrimRight(config.String(lookup, "MO_AUTH_BASE_URL", "https://login.microsoftonline.com"), "/")
	tenant := creds.Tenant
	if strings.TrimSpace(tenant) == "" {
		tenant = "common"
	}
	authorizeEndpoint := authBase + "/" + url.PathEscape(tenant) + "/oauth2/v2.0/authorize"

	v := url.Values{}
	v.Set("client_id", creds.ClientID)
	v.Set("response_type", "code")
	v.Set("redirect_uri", redirectURI)
	v.Set("response_mode", "query")
	v.Set("scope", strings.Join(scopes, " "))
	v.Set("state", state)
	v.Set("code_challenge", challenge)
	v.Set("code_challenge_method", "S256")
	v.Set("login_hint", email)
	if forceConsent {
		v.Set("prompt", "consent")
	}

	return Session{
		State:       state,
		Verifier:    verifier,
		RedirectURI: redirectURI,
		AuthURL:     authorizeEndpoint + "?" + v.Encode(),
		Email:       strings.ToLower(strings.TrimSpace(email)),
		Client:      strings.TrimSpace(client),
		ExpiresAt:   time.Now().UTC().Add(10 * time.Minute),
	}, nil
}

func StartDeviceCode(ctx context.Context, creds config.Credentials, scopes []string, lookup config.LookupFunc) (DeviceCodeResponse, error) {
	if len(scopes) == 0 {
		scopes = DefaultScopes
	}

	authBase := strings.TrimRight(config.String(lookup, "MO_AUTH_BASE_URL", "https://login.microsoftonline.com"), "/")
	tenant := creds.Tenant
	if strings.TrimSpace(tenant) == "" {
		tenant = "common"
	}
	deviceEndpoint := authBase + "/" + url.PathEscape(tenant) + "/oauth2/v2.0/devicecode"

	form := url.Values{}
	form.Set("client_id", creds.ClientID)
	form.Set("scope", strings.Join(scopes, " "))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deviceEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return DeviceCodeResponse{}, fmt.Errorf("create device code request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpClient := newHTTPClient(20 * time.Second)
	resp, err := httpClient.Do(req)
	if err != nil {
		return DeviceCodeResponse{}, fmt.Errorf("device code request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var oauthErr oauthErrorResponse
		if err := json.Unmarshal(body, &oauthErr); err == nil && oauthErr.Error != "" {
			return DeviceCodeResponse{}, &OAuthError{Code: oauthErr.Error, Description: oauthErr.ErrorDescription}
		}
		return DeviceCodeResponse{}, fmt.Errorf("device code request failed with status %d", resp.StatusCode)
	}

	var out DeviceCodeResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return DeviceCodeResponse{}, fmt.Errorf("parse device code response: %w", err)
	}
	if strings.TrimSpace(out.DeviceCode) == "" || strings.TrimSpace(out.UserCode) == "" {
		return DeviceCodeResponse{}, errors.New("device code response missing required fields")
	}
	if out.Interval <= 0 {
		out.Interval = 5
	}
	if out.ExpiresIn <= 0 {
		out.ExpiresIn = 900
	}
	return out, nil
}

func ExchangeAuthCode(ctx context.Context, creds config.Credentials, code, redirectURI, verifier string, scopes []string, lookup config.LookupFunc) (TokenResponse, error) {
	if strings.TrimSpace(code) == "" {
		return TokenResponse{}, errors.New("authorization code is required")
	}
	if strings.TrimSpace(verifier) == "" {
		return TokenResponse{}, errors.New("code verifier is required")
	}
	if len(scopes) == 0 {
		scopes = DefaultScopes
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", creds.ClientID)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", verifier)
	form.Set("scope", strings.Join(scopes, " "))

	return exchangeToken(ctx, creds, form, lookup)
}

func ExchangeDeviceCode(ctx context.Context, creds config.Credentials, deviceCode string, scopes []string, lookup config.LookupFunc) (TokenResponse, error) {
	if strings.TrimSpace(deviceCode) == "" {
		return TokenResponse{}, errors.New("device code is required")
	}
	if len(scopes) == 0 {
		scopes = DefaultScopes
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	form.Set("client_id", creds.ClientID)
	form.Set("device_code", deviceCode)
	form.Set("scope", strings.Join(scopes, " "))

	return exchangeToken(ctx, creds, form, lookup)
}

func WaitForDeviceToken(ctx context.Context, creds config.Credentials, start DeviceCodeResponse, scopes []string, lookup config.LookupFunc) (TokenResponse, error) {
	interval := time.Duration(start.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := time.Now().UTC().Add(time.Duration(start.ExpiresIn) * time.Second)
	if start.ExpiresIn <= 0 {
		deadline = time.Now().UTC().Add(15 * time.Minute)
	}

	for {
		if time.Now().UTC().After(deadline) {
			return TokenResponse{}, errors.New("device code expired before authorization completed")
		}
		tok, err := ExchangeDeviceCode(ctx, creds, start.DeviceCode, scopes, lookup)
		if err == nil {
			return tok, nil
		}

		var oauthErr *OAuthError
		if !errors.As(err, &oauthErr) {
			return TokenResponse{}, err
		}
		switch oauthErr.Code {
		case "authorization_pending":
			// Keep polling.
		case "slow_down":
			interval += 5 * time.Second
		case "authorization_declined":
			return TokenResponse{}, errors.New("device authorization was declined")
		case "expired_token":
			return TokenResponse{}, errors.New("device code expired")
		case "bad_verification_code":
			return TokenResponse{}, errors.New("device code was rejected")
		default:
			return TokenResponse{}, err
		}

		select {
		case <-ctx.Done():
			return TokenResponse{}, ctx.Err()
		case <-time.After(interval):
		}
	}
}

func RefreshAccessToken(ctx context.Context, creds config.Credentials, refreshToken string, scopes []string, lookup config.LookupFunc) (TokenResponse, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return TokenResponse{}, errors.New("refresh token is required")
	}
	if len(scopes) == 0 {
		scopes = DefaultScopes
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", creds.ClientID)
	form.Set("refresh_token", refreshToken)
	form.Set("scope", strings.Join(scopes, " "))

	return exchangeToken(ctx, creds, form, lookup)
}

func exchangeToken(ctx context.Context, creds config.Credentials, form url.Values, lookup config.LookupFunc) (TokenResponse, error) {
	authBase := strings.TrimRight(config.String(lookup, "MO_AUTH_BASE_URL", "https://login.microsoftonline.com"), "/")
	tenant := creds.Tenant
	if strings.TrimSpace(tenant) == "" {
		tenant = "common"
	}
	tokenEndpoint := authBase + "/" + url.PathEscape(tenant) + "/oauth2/v2.0/token"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResponse{}, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpClient := newHTTPClient(20 * time.Second)
	resp, err := httpClient.Do(req)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var oauthErr oauthErrorResponse
		if err := json.Unmarshal(body, &oauthErr); err == nil && oauthErr.Error != "" {
			return TokenResponse{}, &OAuthError{Code: oauthErr.Error, Description: oauthErr.ErrorDescription}
		}
		return TokenResponse{}, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return TokenResponse{}, fmt.Errorf("parse token response: %w", err)
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return TokenResponse{}, errors.New("token response missing access_token")
	}
	return token, nil
}

func ParseRedirectURL(raw string) (code string, state string, err error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", "", fmt.Errorf("parse redirect url: %w", err)
	}
	q := u.Query()
	if e := strings.TrimSpace(q.Get("error")); e != "" {
		return "", "", fmt.Errorf("authorization failed: %s (%s)", e, q.Get("error_description"))
	}
	code = strings.TrimSpace(q.Get("code"))
	state = strings.TrimSpace(q.Get("state"))
	if code == "" {
		return "", "", errors.New("redirect URL does not include authorization code")
	}
	if state == "" {
		return "", "", errors.New("redirect URL does not include state")
	}
	return code, state, nil
}

func randToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func codeChallenge(verifier string) string {
	d := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(d[:])
}
