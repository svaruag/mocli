package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/svaruag/mocli/internal/config"
)

func TestParseRedirectURL(t *testing.T) {
	code, state, err := ParseRedirectURL("http://localhost/cb?code=abc&state=xyz")
	if err != nil {
		t.Fatalf("ParseRedirectURL returned error: %v", err)
	}
	if code != "abc" || state != "xyz" {
		t.Fatalf("unexpected parsed values code=%q state=%q", code, state)
	}
}

func TestParseRedirectURLError(t *testing.T) {
	_, _, err := ParseRedirectURL("http://localhost/cb?error=access_denied&error_description=nope")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestStartDeviceCode(t *testing.T) {
	useMockHTTPClient(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/common/oauth2/v2.0/devicecode" {
			return nil, fmt.Errorf("unexpected path %q", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			return nil, fmt.Errorf("ParseForm: %w", err)
		}
		if got := r.Form.Get("client_id"); got != "client-1" {
			return nil, fmt.Errorf("unexpected client_id %q", got)
		}
		return jsonResponse(http.StatusOK, `{
  "device_code":"dev-code",
  "user_code":"user-code",
  "verification_uri":"https://microsoft.com/devicelogin",
  "verification_uri_complete":"https://microsoft.com/devicelogin?user_code=user-code",
  "expires_in":900,
  "interval":5,
  "message":"Sign in with the code."
}`), nil
	})

	lookup := func(key string) (string, bool) {
		if key == "MO_AUTH_BASE_URL" {
			return "https://login.test", true
		}
		return "", false
	}

	out, err := StartDeviceCode(context.Background(), config.Credentials{
		ClientID: "client-1",
		Tenant:   "common",
	}, nil, lookup)
	if err != nil {
		t.Fatalf("StartDeviceCode returned error: %v", err)
	}
	if out.DeviceCode != "dev-code" {
		t.Fatalf("unexpected device code %q", out.DeviceCode)
	}
	if out.UserCode != "user-code" {
		t.Fatalf("unexpected user code %q", out.UserCode)
	}
	if out.Interval != 5 {
		t.Fatalf("unexpected interval %d", out.Interval)
	}
}

func TestWaitForDeviceTokenAuthorizationPendingThenSuccess(t *testing.T) {
	var calls atomic.Int32
	useMockHTTPClient(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/common/oauth2/v2.0/token" {
			return nil, fmt.Errorf("unexpected path %q", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			return nil, fmt.Errorf("ParseForm: %w", err)
		}
		n := calls.Add(1)
		if n == 1 {
			return jsonResponse(http.StatusBadRequest, `{"error":"authorization_pending","error_description":"pending"}`), nil
		}
		return jsonResponse(http.StatusOK, `{
  "token_type":"Bearer",
  "scope":"openid profile",
  "expires_in":3600,
  "access_token":"access-123",
  "refresh_token":"refresh-123"
}`), nil
	})

	lookup := func(key string) (string, bool) {
		if key == "MO_AUTH_BASE_URL" {
			return "https://login.test", true
		}
		return "", false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	out, err := WaitForDeviceToken(ctx, config.Credentials{
		ClientID: "client-1",
		Tenant:   "common",
	}, DeviceCodeResponse{
		DeviceCode: "device-1",
		Interval:   1,
		ExpiresIn:  300,
	}, nil, lookup)
	if err != nil {
		t.Fatalf("WaitForDeviceToken returned error: %v", err)
	}
	if out.AccessToken != "access-123" {
		t.Fatalf("unexpected access token %q", out.AccessToken)
	}
	if calls.Load() < 2 {
		t.Fatalf("expected at least 2 token calls, got %d", calls.Load())
	}
}

func TestWaitForDeviceTokenDeclined(t *testing.T) {
	useMockHTTPClient(t, func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, `{"error":"authorization_declined","error_description":"declined"}`), nil
	})

	lookup := func(key string) (string, bool) {
		if key == "MO_AUTH_BASE_URL" {
			return "https://login.test", true
		}
		return "", false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := WaitForDeviceToken(ctx, config.Credentials{
		ClientID: "client-1",
		Tenant:   "common",
	}, DeviceCodeResponse{
		DeviceCode: "device-1",
		Interval:   1,
		ExpiresIn:  300,
	}, nil, lookup)
	if err == nil {
		t.Fatalf("expected error")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func useMockHTTPClient(t *testing.T, rt roundTripperFunc) {
	t.Helper()
	orig := newHTTPClient
	newHTTPClient = func(timeout time.Duration) *http.Client {
		return &http.Client{Transport: rt, Timeout: timeout}
	}
	t.Cleanup(func() {
		newHTTPClient = orig
	})
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
