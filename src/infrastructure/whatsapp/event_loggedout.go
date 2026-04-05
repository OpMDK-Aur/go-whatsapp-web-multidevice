package whatsapp

import (
	"context"
	"time"
	"go.mau.fi/whatsmeow/types/events"
)

func createLoggedOutPayload(
	ctx context.Context,
	evt *events.LoggedOut,
	deviceID string,
	instanceID string,
) map[string]any {
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
	if instanceID != "" {
		body["instance_id"] = instanceID
	}
	body["timestamp"] = time.Now().Format(time.RFC3339)
	body["payload"] = payload

	return body
}

func forwardLoggedOutToWebhook(
	ctx context.Context,
	evt *events.LoggedOut,
	deviceID string,
	instanceID string,
) error {
	payload := createLoggedOutPayload(ctx, evt, deviceID, instanceID)
	return forwardPayloadToConfiguredWebhooks(ctx, payload, "loggedout")
}
