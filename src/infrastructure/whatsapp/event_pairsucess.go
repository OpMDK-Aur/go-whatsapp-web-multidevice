package whatsapp

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

func createPairSuccessPayload(ctx context.Context, evt *events.PairSuccess, deviceID string) map[string]any {
	body := make(map[string]any)
	payload := make(map[string]any)

	payload["BusinessName"] = evt.BusinessName
	payload["Platfrom"] = evt.Platform

	// Wrap in body structure
	body["event"] = "pairsuccess"
	if deviceID != "" {
		body["device_id"] = deviceID
	}
	body["timestamp"] = time.Now().Format(time.RFC3339)
	body["payload"] = payload

	return body
}

func forwardPairSuccessToWebhook(ctx context.Context, evt *events.PairSuccess, deviceID string) error {
	payload := createPairSuccessPayload(ctx, evt, deviceID)
	return forwardPayloadToConfiguredWebhooks(ctx, payload, "pairsuccess")
}
