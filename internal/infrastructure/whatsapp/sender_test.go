package whatsapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendToGroupSuccess(t *testing.T) {
	t.Parallel()
	var received sendGroupMessageRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/messages/group" {
			t.Errorf("expected /messages/group, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := NewSender(server.URL, "test-token")
	err := s.SendToGroup(context.Background(), "group-123", "Hello group")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if received.GroupID != "group-123" {
		t.Errorf("expected group-123, got %s", received.GroupID)
	}
	if received.Message != "Hello group" {
		t.Errorf("expected 'Hello group', got %s", received.Message)
	}
}

func TestSendToGroupAPIError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal"}`))
	}))
	defer server.Close()

	s := NewSender(server.URL, "test-token")
	err := s.SendToGroup(context.Background(), "group-123", "Hello")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestSendToGroupInvalidURL(t *testing.T) {
	t.Parallel()
	s := NewSender("://invalid-url", "token")
	err := s.SendToGroup(context.Background(), "group-123", "Hello")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestSendToGroupConnectionRefused(t *testing.T) {
	t.Parallel()
	s := NewSender("http://127.0.0.1:1", "token")
	err := s.SendToGroup(context.Background(), "group-123", "Hello")
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestSendToGroupContextCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := NewSender("http://localhost:9999", "token")
	err := s.SendToGroup(ctx, "group-123", "Hello")
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}
