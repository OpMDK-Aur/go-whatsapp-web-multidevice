package whatsapp

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func createLoggedOutPayload(ctx context.Context, evt *events.LoggedOut, deviceID string, client *whatsmeow.Client) map[string]any {
	body := make(map[string]any)
	payload := make(map[string]any)

	payload["on_connect"] = evt.OnConnect
	payload["reason"] = evt.Reason
	payload["reason_description"] = evt.Reason.String()

	// Wrap in body structure
	body["event"] = "loggedout"
	if deviceID != "" {
		body["device_id"] = deviceID
	}
	body["timestamp"] = time.Now().Format(time.RFC3339)
	body["payload"] = payload

	return body
}

func forwardLoggedOutToWebhook(ctx context.Context, evt *events.LoggedOut, deviceID string, client *whatsmeow.Client) error {
	payload := createLoggedOutPayload(ctx, evt, deviceID, client)
	return forwardPayloadToConfiguredWebhooks(ctx, payload, "loggedout")
}
