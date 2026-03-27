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

	"github.com/rachelJG/event-notification-service/internal/pkg/apperror"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
	ctx, span := otel.Tracer("whatsapp").Start(ctx, "WhatsAppSender.SendToGroup",
		trace.WithAttributes(
			attribute.String("whatsapp.group_id", groupID),
		),
	)
	defer span.End()

	select {
	case <-ctx.Done():
		span.SetStatus(codes.Error, "context canceled")
		return ctx.Err()
	default:
	}

	payload, err := json.Marshal(sendGroupMessageRequest{
		GroupID: groupID,
		Message: message,
	})
	if err != nil {
		span.SetStatus(codes.Error, "marshal failed")
		span.RecordError(err)
		return apperror.Internal("whatsapp marshal request failed", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL+"/messages/group", bytes.NewReader(payload))
	if err != nil {
		span.SetStatus(codes.Error, "create request failed")
		span.RecordError(err)
		return apperror.Internal("whatsapp create request failed", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		span.SetStatus(codes.Error, "send failed")
		span.RecordError(err)
		return apperror.Unavailable("whatsapp send failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		apiErr := apperror.Unavailable(fmt.Sprintf("whatsapp api error (status %d): %s", resp.StatusCode, string(body)), nil)
		span.SetStatus(codes.Error, "api error")
		span.RecordError(apiErr)
		return apiErr
	}

	return nil
}
