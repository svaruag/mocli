package app

import "testing"

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
