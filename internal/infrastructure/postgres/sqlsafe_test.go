package postgres

import (
	"strings"
	"testing"
)

func TestSafeIdentifierAllowed(t *testing.T) {
	allowed := map[string]struct{}{
		"created_at": {},
		"updated_at": {},
	}

	result := SafeIdentifier("created_at", allowed, "id")
	if !strings.Contains(result, "created_at") {
		t.Errorf("expected 'created_at' in result, got %q", result)
	}
}

func TestSafeIdentifierNotAllowed(t *testing.T) {
	allowed := map[string]struct{}{
		"created_at": {},
	}

	result := SafeIdentifier("malicious_column", allowed, "id")
	if strings.Contains(result, "malicious_column") {
		t.Errorf("expected fallback, got %q", result)
	}
	if !strings.Contains(result, "id") {
		t.Errorf("expected 'id' fallback in result, got %q", result)
	}
}

func TestSafeIdentifierNilAllowlist(t *testing.T) {
	result := SafeIdentifier("anything", nil, "id")
	if !strings.Contains(result, "anything") {
		t.Errorf("expected 'anything' with nil allowlist, got %q", result)
	}
}

func TestSafeOrderBy(t *testing.T) {
	allowed := map[string]struct{}{
		"created_at": {},
	}

	result := SafeOrderBy("created_at", allowed, "id")
	if !strings.HasPrefix(result, "ORDER BY ") {
		t.Errorf("expected 'ORDER BY ' prefix, got %q", result)
	}
	if !strings.Contains(result, "created_at") {
		t.Errorf("expected 'created_at' in result, got %q", result)
	}
}

func TestSafeOrderByFallback(t *testing.T) {
	allowed := map[string]struct{}{
		"created_at": {},
	}

	result := SafeOrderBy("drop_table", allowed, "id")
	if strings.Contains(result, "drop_table") {
		t.Errorf("expected fallback, got %q", result)
	}
}
