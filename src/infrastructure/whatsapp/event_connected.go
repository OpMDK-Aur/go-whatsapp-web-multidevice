package whatsapp

import (
	"context"
	"time"
)

func createConnectedPayload(ctx context.Context,  deviceID, deviceJID string) map[string]any {
	body := make(map[string]any)

	// Wrap in body structure
	body["event"] = "connected"
	if deviceID != "" {
		body["device_id"] = deviceID
	}
	if deviceJID != "" {
		body["device_jid"] = deviceJID
	}
	body["timestamp"] = time.Now().Format(time.RFC3339)

	return body
}

func forwardConnectedToWebhook(ctx context.Context, deviceID, deviceJID string) error {
	payload := createConnectedPayload(ctx, deviceID, deviceJID)
	return forwardPayloadToConfiguredWebhooks(ctx, payload, "connected")
}
