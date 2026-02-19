package app

import (
	"reflect"
	"testing"
)

func TestNormalizeAuthAddArgsEmailFirstWithFlags(t *testing.T) {
	in := []string{"user@contoso.com", "--device"}
	got := normalizeOnePositionalArgs(in)
	want := []string{"--device", "user@contoso.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeOnePositionalArgs(%v) = %v, want %v", in, got, want)
	}
}

func TestNormalizeAuthAddArgsFlagFirstUnchanged(t *testing.T) {
	in := []string{"--device", "user@contoso.com"}
	got := normalizeOnePositionalArgs(in)
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("normalizeOnePositionalArgs(%v) = %v, want unchanged", in, got)
	}
}

func TestNormalizeAuthAddArgsEmailOnlyUnchanged(t *testing.T) {
	in := []string{"user@contoso.com"}
	got := normalizeOnePositionalArgs(in)
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("normalizeOnePositionalArgs(%v) = %v, want unchanged", in, got)
	}
}
