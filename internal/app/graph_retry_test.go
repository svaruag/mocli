package app

import (
	"net/http"
	"testing"
	"time"
)

func TestShouldRetryStatus(t *testing.T) {
	if !shouldRetryStatus(http.StatusTooManyRequests) {
		t.Fatalf("expected 429 to be retryable")
	}
	if !shouldRetryStatus(http.StatusBadGateway) {
		t.Fatalf("expected 502 to be retryable")
	}
	if shouldRetryStatus(http.StatusForbidden) {
		t.Fatalf("did not expect 403 to be retryable")
	}
}

func TestRetryDelayUsesRetryAfter(t *testing.T) {
	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("Retry-After", "2")
	got := retryDelay(resp, 1)
	want := 2 * time.Second
	if got != want {
		t.Fatalf("retryDelay = %v, want %v", got, want)
	}
}

func TestBackoffDurationIncreases(t *testing.T) {
	a := backoffDuration(1)
	b := backoffDuration(2)
	if !(b > a) {
		t.Fatalf("expected backoff to increase: %v then %v", a, b)
	}
}
