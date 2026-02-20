package app

import (
	"net/http"
	"testing"
	"time"
)

func TestExtractPageToken(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "skiptoken",
			in:   "https://graph.microsoft.com/v1.0/me/messages?$skiptoken=abc123",
			want: "abc123",
		},
		{
			name: "skip",
			in:   "https://graph.microsoft.com/v1.0/me/messages?$skip=20",
			want: "20",
		},
		{
			name: "empty",
			in:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPageToken(tt.in)
			if got != tt.want {
				t.Fatalf("extractPageToken(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRetryDelayCapsRetryAfter(t *testing.T) {
	resp := &http.Response{
		Header: make(http.Header),
	}
	resp.Header.Set("Retry-After", "3600")

	got := retryDelay(resp, 1)
	if got != maxRetryAfter {
		t.Fatalf("retryDelay cap = %s, want %s", got, maxRetryAfter)
	}
}

func TestRetryDelayUsesRetryAfterWithinCap(t *testing.T) {
	resp := &http.Response{
		Header: make(http.Header),
	}
	resp.Header.Set("Retry-After", "5")

	got := retryDelay(resp, 1)
	if got != 5*time.Second {
		t.Fatalf("retryDelay = %s, want 5s", got)
	}
}
