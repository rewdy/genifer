package config

import (
	"context"
	"testing"
)

func TestResolveLiteral(t *testing.T) {
	cases := []string{"sk-plain", "just text", "{notadirective", "value}", "{env with space:X}"}
	for _, raw := range cases {
		got, err := Value(raw).Resolve(context.Background())
		if err != nil {
			t.Fatalf("Resolve(%q) unexpected error: %v", raw, err)
		}
		if got != raw {
			t.Errorf("Resolve(%q) = %q, want verbatim", raw, got)
		}
	}
}

func TestResolveEnv(t *testing.T) {
	t.Setenv("GENIFER_TEST_KEY", "secret-123")
	got, err := Value("{env:GENIFER_TEST_KEY}").Resolve(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "secret-123" {
		t.Errorf("got %q, want %q", got, "secret-123")
	}
}

func TestResolveEnvMissing(t *testing.T) {
	_, err := Value("{env:GENIFER_DEFINITELY_UNSET_VAR}").Resolve(context.Background())
	if err == nil {
		t.Fatal("expected error for unset env var, got nil")
	}
}

func TestResolveCommand(t *testing.T) {
	got, err := Value("{cmd:printf '  hello  '}").Resolve(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Errorf("got %q, want trimmed %q", got, "hello")
	}
}

func TestResolveCommandFailure(t *testing.T) {
	_, err := Value("{cmd:exit 3}").Resolve(context.Background())
	if err == nil {
		t.Fatal("expected error for non-zero command exit, got nil")
	}
}

func TestResolveSingleLevel(t *testing.T) {
	// A command whose output looks like a directive must not be re-resolved.
	got, err := Value("{cmd:printf '%s' '{env:HOME}'}").Resolve(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "{env:HOME}" {
		t.Errorf("got %q, want literal %q (no re-resolution)", got, "{env:HOME}")
	}
}

func TestIsCommand(t *testing.T) {
	if !Value("{cmd:echo hi}").IsCommand() {
		t.Error("expected IsCommand true for cmd directive")
	}
	if Value("{env:X}").IsCommand() {
		t.Error("expected IsCommand false for env directive")
	}
	if Value("literal").IsCommand() {
		t.Error("expected IsCommand false for literal")
	}
}
