package whatsapp

import (
	"context"
	"time"
)

func createConnectedPayload(ctx context.Context,  deviceID, instanceID string) map[string]any {
	body := make(map[string]any)

	// Wrap in body structure
	body["event"] = "connected"
	if deviceID != "" {
		body["device_id"] = deviceID
	}
	if instanceID != "" {
		body["instance_id"] = instanceID
	}
	body["timestamp"] = time.Now().Format(time.RFC3339)

	return body
}

func forwardConnectedToWebhook(ctx context.Context, deviceID, instanceID string) error {
	payload := createConnectedPayload(ctx, deviceID, instanceID)
	return forwardPayloadToConfiguredWebhooks(ctx, payload, "connected")
}
