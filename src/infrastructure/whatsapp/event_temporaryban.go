package whatsapp

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

func createTemporaryBanPayload(ctx context.Context, evt *events.TemporaryBan, deviceID string) map[string]any {
	body := make(map[string]any)
	payload := make(map[string]any)

	payload["expire"] = evt.Expire
	payload["code"] = evt.Code
	payload["code_description"] = evt.Code.String()

	// Wrap in body structure
	body["event"] = "temporaryban"
	if deviceID != "" {
		body["device_id"] = deviceID
	}
	body["timestamp"] = time.Now().Format(time.RFC3339)
	body["payload"] = payload

	return body
}

func forwardTemporaryBanToWebhook(ctx context.Context, evt *events.TemporaryBan, deviceID string) error {
	payload := createTemporaryBanPayload(ctx, evt, deviceID)
	return forwardPayloadToConfiguredWebhooks(ctx, payload, "temporaryban")
}
