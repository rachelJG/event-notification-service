package logger

import (
	"testing"
)

func TestNewDevelopment(t *testing.T) {
	l, err := New("development", "debug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewProduction(t *testing.T) {
	l, err := New("production", "info")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewInvalidLevelFallsBack(t *testing.T) {
	l, err := New("production", "invalid-level")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil logger with fallback level")
	}
}
