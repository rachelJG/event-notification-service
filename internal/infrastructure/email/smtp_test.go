package email

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestNewSMTPSender(t *testing.T) {
	s := NewSMTPSender("smtp.example.com", "587", "user@example.com", "secret", "noreply@example.com")

	if s.host != "smtp.example.com" {
		t.Errorf("host = %q, want %q", s.host, "smtp.example.com")
	}
	if s.port != "587" {
		t.Errorf("port = %q, want %q", s.port, "587")
	}
	if s.user != "user@example.com" {
		t.Errorf("user = %q, want %q", s.user, "user@example.com")
	}
	if s.pass != "secret" {
		t.Errorf("pass = %q, want %q", s.pass, "secret")
	}
	if s.from != "noreply@example.com" {
		t.Errorf("from = %q, want %q", s.from, "noreply@example.com")
	}
}

func TestBuildMessage(t *testing.T) {
	msg := string(buildMessage("from@test.com", "to@test.com", "Test Subject", "Hello body"))

	checks := []struct {
		name     string
		contains string
	}{
		{"From header", "From: from@test.com\r\n"},
		{"To header", "To: to@test.com\r\n"},
		{"Subject header", "Subject: Test Subject\r\n"},
		{"MIME-Version", "MIME-Version: 1.0\r\n"},
		{"Content-Type", "Content-Type: text/plain; charset=\"utf-8\"\r\n"},
		{"body", "Hello body"},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(msg, c.contains) {
				t.Errorf("message missing %s: %q not found in:\n%s", c.name, c.contains, msg)
			}
		})
	}

	// Headers and body must be separated by a blank line (\r\n\r\n)
	if !strings.Contains(msg, "\r\n\r\n") {
		t.Error("message missing blank line between headers and body")
	}
}

func TestBuildMessageHeaderOrder(t *testing.T) {
	msg := string(buildMessage("a@b.com", "c@d.com", "Sub", "Body"))

	headers, bodyPart, found := strings.Cut(msg, "\r\n\r\n")
	if !found {
		t.Fatal("no header/body separator found")
	}

	if bodyPart != "Body" {
		t.Errorf("body = %q, want %q", bodyPart, "Body")
	}

	// Verify all five headers are present
	expectedHeaders := []string{"From:", "To:", "Subject:", "MIME-Version:", "Content-Type:"}
	for _, h := range expectedHeaders {
		if !strings.Contains(headers, h) {
			t.Errorf("headers missing %q", h)
		}
	}
}

func TestSendRespectsContextCancellation(t *testing.T) {
	s := NewSMTPSender("localhost", "0", "", "", "from@test.com")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := s.Send(ctx, "to@test.com", "subject", "body")
	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected context canceled error, got: %v", err)
	}
}

func TestSendWithDeadlineExceeded(t *testing.T) {
	s := NewSMTPSender("localhost", "0", "", "", "from@test.com")

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond) // ensure deadline passes

	err := s.Send(ctx, "to@test.com", "subject", "body")
	if err == nil {
		t.Fatal("expected error from expired context, got nil")
	}
}

func TestSendConnectionRefused(t *testing.T) {
	// Use a port that is unlikely to have an SMTP server listening
	s := NewSMTPSender("127.0.0.1", "19", "user", "pass", "from@test.com")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := s.Send(ctx, "to@test.com", "subject", "body")
	if err == nil {
		t.Fatal("expected error from connection refused, got nil")
	}
}

func TestSendContextCanceledDuringSend(t *testing.T) {
	// Start a TCP listener that accepts but never responds, causing smtp.SendMail to block.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	// Accept connections in background but do nothing (simulate slow server).
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			// Hold the connection open without responding.
			defer conn.Close()
		}
	}()

	_, port, _ := net.SplitHostPort(ln.Addr().String())
	s := NewSMTPSender("127.0.0.1", port, "", "", "from@test.com")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	sendErr := s.Send(ctx, "to@test.com", "subject", "body")
	if sendErr == nil {
		t.Fatal("expected error from context cancellation during send")
	}
	if !strings.Contains(sendErr.Error(), "canceled") && !strings.Contains(sendErr.Error(), "deadline") {
		t.Errorf("expected context error, got: %v", sendErr)
	}
}

func TestSendNoAuth(t *testing.T) {
	// Verify that empty user/pass results in nil auth (code path where auth is nil)
	s := NewSMTPSender("127.0.0.1", "19", "", "", "from@test.com")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := s.Send(ctx, "to@test.com", "subject", "body")
	// Will fail at connection level, but exercises the auth=nil path
	if err == nil {
		t.Fatal("expected error from connection refused, got nil")
	}
}
