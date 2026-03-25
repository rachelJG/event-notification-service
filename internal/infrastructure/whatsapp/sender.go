// Package whatsapp provides a WhatsApp messaging infrastructure adapter.
package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Sender implements ports.WhatsAppSender using an HTTP-based WhatsApp API.
type Sender struct {
	apiURL string
	token  string
	client *http.Client
}

// NewSender creates a new WhatsApp sender.
func NewSender(apiURL, token string) *Sender {
	return &Sender{
		apiURL: apiURL,
		token:  token,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type sendGroupMessageRequest struct {
	GroupID string `json:"group_id"`
	Message string `json:"message"`
}

// SendToGroup sends a message to a WhatsApp group via the configured API.
func (s *Sender) SendToGroup(ctx context.Context, groupID, message string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	payload, err := json.Marshal(sendGroupMessageRequest{
		GroupID: groupID,
		Message: message,
	})
	if err != nil {
		return fmt.Errorf("whatsapp marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL+"/messages/group", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("whatsapp create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("whatsapp send: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("whatsapp api error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
