package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types/events"
)

var reMention = regexp.MustCompile(`\B@\w+`)

// Event types for webhook payload
const (
	EventTypeMessage         = "message"
	EventTypeMessageReaction = "message.reaction"
	EventTypeMessageRevoked  = "message.revoked"
	EventTypeMessageEdited   = "message.edited"
)

// WebhookEvent is the top-level structure for webhook payloads
type WebhookEvent struct {
	Event    string         `json:"event"`
	DeviceID string         `json:"device_id"`
	Payload  map[string]any `json:"payload"`
}

type webhookContactPayload struct {
	DisplayName string `json:"displayName"`
	VCard       string `json:"vcard"`
	PhoneNumber string `json:"phone_number,omitempty"`
}

// forwardMessageToWebhook is a helper function to forward message event to webhook url
func forwardMessageToWebhook(ctx context.Context, client *whatsmeow.Client, evt *events.Message) error {
	webhookEvent, err := createWebhookEvent(ctx, client, evt)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"event":     webhookEvent.Event,
		"device_id": webhookEvent.DeviceID,
		"payload":   webhookEvent.Payload,
	}

	return forwardPayloadToConfiguredWebhooks(ctx, payload, webhookEvent.Event)
}

func isReactionMessage(evt *events.Message) bool {
	if evt == nil || evt.Message == nil {
		return false
	}

	return utils.UnwrapMessage(evt.Message).GetReactionMessage() != nil
}

func createWebhookEvent(ctx context.Context, client *whatsmeow.Client, evt *events.Message) (*WebhookEvent, error) {
	webhookEvent := &WebhookEvent{
		Event:   EventTypeMessage,
		Payload: make(map[string]any),
	}

	// Set device_id
	if client != nil && client.Store != nil && client.Store.ID != nil {
		deviceJID := NormalizeJIDFromLID(ctx, client.Store.ID.ToNonAD(), client)
		webhookEvent.DeviceID = deviceJID.ToNonAD().String()
	}

	// Determine event type and build payload
	eventType, payload, err := buildEventPayload(ctx, client, evt)
	if err != nil {
		return nil, err
	}

	webhookEvent.Event = eventType
	webhookEvent.Payload = payload

	return webhookEvent, nil
}

func buildEventPayload(ctx context.Context, client *whatsmeow.Client, evt *events.Message) (string, map[string]any, error) {
	payload := make(map[string]any)

	msg := utils.UnwrapMessage(evt.Message)

	// Common fields for all message types
	payload["id"] = evt.Info.ID
	payload["timestamp"] = evt.Info.Timestamp.Format(time.RFC3339)
	payload["is_from_me"] = evt.Info.IsFromMe

	// Build from/from_lid fields
	buildFromFields(ctx, client, evt, payload)

	// Set from_name (pushname)
	if pushname := evt.Info.PushName; pushname != "" {
		payload["from_name"] = pushname
	}

	// Check for protocol messages (revoke, edit)
	if protocolMessage := msg.GetProtocolMessage(); protocolMessage != nil {
		protocolType := protocolMessage.GetType().String()

		switch protocolType {
		case "REVOKE":
			if key := protocolMessage.GetKey(); key != nil {
				payload["revoked_message_id"] = key.GetID()
				payload["revoked_from_me"] = key.GetFromMe()
				if key.GetRemoteJID() != "" {
					payload["revoked_chat"] = key.GetRemoteJID()
				}
			}
			return EventTypeMessageRevoked, payload, nil

		case "MESSAGE_EDIT":
			if key := protocolMessage.GetKey(); key != nil {
				payload["original_message_id"] = key.GetID()
			}
			if editedMessage := protocolMessage.GetEditedMessage(); editedMessage != nil {
				if editedText := editedMessage.GetExtendedTextMessage(); editedText != nil {
					payload["body"] = editedText.GetText()
				} else if editedConv := editedMessage.GetConversation(); editedConv != "" {
					payload["body"] = editedConv
				}
			}
			return EventTypeMessageEdited, payload, nil
		}
	}

	// Check for reaction message
	if reactionMessage := msg.GetReactionMessage(); reactionMessage != nil {
		payload["reaction"] = reactionMessage.GetText()
		if key := reactionMessage.GetKey(); key != nil {
			payload["reacted_message_id"] = key.GetID()
		}
		return EventTypeMessageReaction, payload, nil
	}

	// Regular message - build body and media fields
	if err := buildMessageBody(ctx, client, evt, payload); err != nil {
		return "", nil, err
	}

	// Add optional fields
	if err := buildOptionalFields(ctx, client, evt, msg, payload); err != nil {
		return "", nil, err
	}

	return EventTypeMessage, payload, nil
}

func buildFromFields(ctx context.Context, client *whatsmeow.Client, evt *events.Message, payload map[string]any) {
	chatJID := evt.Info.Chat.ToNonAD()
	if chatJID.Server == "lid" {
		payload["chat_lid"] = chatJID.String()
		chatJID = NormalizeJIDFromLID(ctx, chatJID, client).ToNonAD()
	}
	payload["chat_id"] = chatJID.String()

	senderJID := evt.Info.Sender
	if senderJID.Server == "lid" {
		payload["from_lid"] = senderJID.ToNonAD().String()
	}

	normalizedSenderJID := NormalizeJIDFromLID(ctx, senderJID, client)
	payload["from"] = normalizedSenderJID.ToNonAD().String()
}



func buildMessageBody(ctx context.Context, client *whatsmeow.Client, evt *events.Message, payload map[string]any) error {
	message := utils.BuildEventMessage(evt)

	// Replace LID mentions with phone numbers in text
	if message.Text != "" && client != nil && client.Store != nil && client.Store.LIDs != nil {
		tags := reMention.FindAllString(message.Text, -1)
		tagsMap := make(map[string]bool)
		for _, tag := range tags {
			tagsMap[tag] = true
		}
		for tag := range tagsMap {
			lid, err := types.ParseJID(tag[1:] + "@lid")
			if err != nil {
				logrus.Errorf("Error when parse jid: %v", err)
			} else {
				pn, err := client.Store.LIDs.GetPNForLID(ctx, lid)
				if err != nil {
					logrus.Errorf("Error when get pn for lid %s: %v", lid.ToNonAD().String(), err)
				}
				if !pn.IsEmpty() {
					message.Text = strings.Replace(message.Text, tag, fmt.Sprintf("@%s", pn.User), -1)
				}
			}
		}
		payload["body"] = message.Text
	} else if message.Text != "" {
		payload["body"] = message.Text
	}

	// Fallback: extract caption from media messages if no text body was set
	if _, hasBody := payload["body"]; !hasBody {
		msg := utils.UnwrapMessage(evt.Message)
		if caption := utils.ExtractMediaCaption(msg); caption != "" {
			payload["body"] = caption
		}
	}

	// Add reply context if present
	if message.RepliedId != "" {
		payload["replied_to_id"] = message.RepliedId
	}
	if message.QuotedMessage != "" {
		payload["quoted_body"] = message.QuotedMessage
	}

	return nil
}

func buildOptionalFields(ctx context.Context, client *whatsmeow.Client, evt *events.Message, msg *waE2E.Message, payload map[string]any) error {
	if evt.IsViewOnce {
		payload["view_once"] = true
	}

	if utils.BuildForwarded(evt) {
		payload["forwarded"] = true
	}

	if referral := utils.ExtractExternalAdReply(msg); referral != nil {
		payload["referral"] = referral
	}

	if err := buildMediaFields(ctx, client, msg, payload); err != nil {
		return err
	}

	buildOtherMessageTypes(msg, payload)
	buildConversionMessageTypes(msg, payload)

	return nil
}

// buildRichMediaPayload creates a map with all fields needed to identify and download media.
// mediaKey, fileSHA256, and fileEncSHA256 are base64-encoded.
func buildRichMediaPayload(
	url string,
	mimeType string,
	mediaKey []byte,
	fileSHA256 []byte,
	fileEncSHA256 []byte,
	fileLength uint64,
) map[string]any {
	meta := map[string]any{
		"url":       url,
		"mime_type": mimeType,
		"file_size": fileLength,
	}
	if len(mediaKey) > 0 {
		meta["media_key"] = base64.StdEncoding.EncodeToString(mediaKey)
	}
	if len(fileSHA256) > 0 {
		meta["file_sha256"] = base64.StdEncoding.EncodeToString(fileSHA256)
	}
	if len(fileEncSHA256) > 0 {
		meta["file_enc_sha256"] = base64.StdEncoding.EncodeToString(fileEncSHA256)
	}
	return meta
}

func buildMediaFields(ctx context.Context, client *whatsmeow.Client, msg *waE2E.Message, payload map[string]any) error {
	if audioMedia := msg.GetAudioMessage(); audioMedia != nil {
		if config.WhatsappAutoDownloadMedia {
			extracted, err := utils.ExtractMedia(ctx, client, config.PathMedia, audioMedia)
			if err != nil {
				logrus.Errorf("Failed to download audio: %v", err)
				return pkgError.WebhookError(fmt.Sprintf("Failed to download audio: %v", err))
			}
			payload["audio"] = extracted.MediaPath
		} else {
			meta := buildRichMediaPayload(
				audioMedia.GetURL(),
				audioMedia.GetMimetype(),
				audioMedia.GetMediaKey(),
				audioMedia.GetFileSHA256(),
				audioMedia.GetFileEncSHA256(),
				audioMedia.GetFileLength(),
			)
			meta["seconds"] = audioMedia.GetSeconds();
			meta["is_ptt"] = audioMedia.GetPTT()
			payload["audio"] = meta
		}
	}

	if documentMedia := msg.GetDocumentMessage(); documentMedia != nil {
		if config.WhatsappAutoDownloadMedia {
			extracted, err := utils.ExtractMedia(ctx, client, config.PathMedia, documentMedia)
			if err != nil {
				logrus.Errorf("Failed to download document: %v", err)
				return pkgError.WebhookError(fmt.Sprintf("Failed to download document: %v", err))
			}
			payload["document"] = buildAutoDownloadPayload(extracted)
		} else {
			meta := buildRichMediaPayload(
				documentMedia.GetURL(),
				documentMedia.GetMimetype(),
				documentMedia.GetMediaKey(),
				documentMedia.GetFileSHA256(),
				documentMedia.GetFileEncSHA256(),
				documentMedia.GetFileLength(),
			)
			meta["filename"] = documentMedia.GetFileName()
			if cap := documentMedia.GetCaption(); cap != "" {
				meta["caption"] = cap
			}
			payload["document"] = meta
		}
	}

	if imageMedia := msg.GetImageMessage(); imageMedia != nil {
		if config.WhatsappAutoDownloadMedia {
			extracted, err := utils.ExtractMedia(ctx, client, config.PathMedia, imageMedia)
			if err != nil {
				logrus.Errorf("Failed to download image: %v", err)
				return pkgError.WebhookError(fmt.Sprintf("Failed to download image: %v", err))
			}
			payload["image"] = buildAutoDownloadPayload(extracted)
		} else {
			meta := buildRichMediaPayload(
				imageMedia.GetURL(),
				imageMedia.GetMimetype(),
				imageMedia.GetMediaKey(),
				imageMedia.GetFileSHA256(),
				imageMedia.GetFileEncSHA256(),
				imageMedia.GetFileLength(),
			)
			if thumbnail := imageMedia.GetJPEGThumbnail(); thumbnail != nil {
				meta["jpeg_thumbnail"] = base64.StdEncoding.EncodeToString(thumbnail)
			}
			if cap := imageMedia.GetCaption(); cap != "" {
				meta["caption"] = cap
			}
			payload["image"] = meta
		}
	}

	if stickerMedia := msg.GetStickerMessage(); stickerMedia != nil {
		if config.WhatsappAutoDownloadMedia {
			extracted, err := utils.ExtractMedia(ctx, client, config.PathMedia, stickerMedia)
			if err != nil {
				logrus.Errorf("Failed to download sticker: %v", err)
				return pkgError.WebhookError(fmt.Sprintf("Failed to download sticker: %v", err))
			}
			payload["sticker"] = extracted.MediaPath
		} else {
			meta := buildRichMediaPayload(
				stickerMedia.GetURL(),
				stickerMedia.GetMimetype(),
				stickerMedia.GetMediaKey(),
				stickerMedia.GetFileSHA256(),
				stickerMedia.GetFileEncSHA256(),
				stickerMedia.GetFileLength(),
			)
			payload["sticker"] = meta
		}
	}

	if videoMedia := msg.GetVideoMessage(); videoMedia != nil {
		if config.WhatsappAutoDownloadMedia {
			extracted, err := utils.ExtractMedia(ctx, client, config.PathMedia, videoMedia)
			if err != nil {
				logrus.Errorf("Failed to download video: %v", err)
				return pkgError.WebhookError(fmt.Sprintf("Failed to download video: %v", err))
			}
			payload["video"] = buildAutoDownloadPayload(extracted)
		} else {
			meta := buildRichMediaPayload(
				videoMedia.GetURL(),
				videoMedia.GetMimetype(),
				videoMedia.GetMediaKey(),
				videoMedia.GetFileSHA256(),
				videoMedia.GetFileEncSHA256(),
				videoMedia.GetFileLength(),
			)
			if thumbnail := videoMedia.GetJPEGThumbnail(); thumbnail != nil {
				meta["jpeg_thumbnail"] = base64.StdEncoding.EncodeToString(thumbnail)
			}
			if cap := videoMedia.GetCaption(); cap != "" {
				meta["caption"] = cap
			}
			payload["video"] = meta
		}
	}

	if ptvMedia := msg.GetPtvMessage(); ptvMedia != nil {
		if config.WhatsappAutoDownloadMedia {
			extracted, err := utils.ExtractMedia(ctx, client, config.PathMedia, ptvMedia)
			if err != nil {
				logrus.Errorf("Failed to download video note: %v", err)
				return pkgError.WebhookError(fmt.Sprintf("Failed to download video note: %v", err))
			}
			payload["video_note"] = buildAutoDownloadPayload(extracted)
		} else {
			meta := buildRichMediaPayload(
				ptvMedia.GetURL(),
				ptvMedia.GetMimetype(),
				ptvMedia.GetMediaKey(),
				ptvMedia.GetFileSHA256(),
				ptvMedia.GetFileEncSHA256(),
				ptvMedia.GetFileLength(),
			)
			if cap := ptvMedia.GetCaption(); cap != "" {
				meta["caption"] = cap
			}
			payload["video_note"] = meta
		}
	}

	return nil
}

// buildAutoDownloadPayload builds the media payload for auto-downloaded media.
// Returns just the path string if no caption (backward compatible), or a map with path+caption.
func buildAutoDownloadPayload(extracted utils.ExtractedMedia) any {
	if extracted.Caption != "" {
		return map[string]any{
			"path":    extracted.MediaPath,
			"caption": extracted.Caption,
		}
	}
	return extracted.MediaPath
}

func buildOtherMessageTypes(msg *waE2E.Message, payload map[string]any) {
	if contactMessage := msg.GetContactMessage(); contactMessage != nil {
		payload["contact"] = buildWebhookContactPayload(contactMessage)
	}

	if contactsArrayMessage := msg.GetContactsArrayMessage(); contactsArrayMessage != nil {
		payload["contacts_array"] = buildWebhookContactsArrayPayload(contactsArrayMessage.GetContacts())
	}

	if listMessage := msg.GetListMessage(); listMessage != nil {
		payload["list"] = listMessage
	}

	if liveLocationMessage := msg.GetLiveLocationMessage(); liveLocationMessage != nil {
		payload["live_location"] = liveLocationMessage
	}

	if locationMessage := msg.GetLocationMessage(); locationMessage != nil {
		payload["location"] = locationMessage
	}

	if orderMessage := msg.GetOrderMessage(); orderMessage != nil {
		payload["order"] = orderMessage
	}
}

func buildWebhookContactPayload(contact *waE2E.ContactMessage) webhookContactPayload {
	if contact == nil {
		return webhookContactPayload{}
	}

	vcard := contact.GetVcard()
	return webhookContactPayload{
		DisplayName: contact.GetDisplayName(),
		VCard:       vcard,
		PhoneNumber: utils.ExtractPhoneFromVCard(vcard),
	}
}

func buildWebhookContactsArrayPayload(contacts []*waE2E.ContactMessage) []webhookContactPayload {
	result := make([]webhookContactPayload, 0, len(contacts))
	for _, contact := range contacts {
		if contact == nil {
			continue
		}
		result = append(result, buildWebhookContactPayload(contact))
	}
	return result
}


func buildWebhookContactPayload(contact *waE2E.ContactMessage) webhookContactPayload {
	if contact == nil {
		return webhookContactPayload{}
	}

	vcard := contact.GetVcard()
	return webhookContactPayload{
		DisplayName: contact.GetDisplayName(),
		VCard:       vcard,
		PhoneNumber: utils.ExtractPhoneFromVCard(vcard),
	}
}

func buildWebhookContactsArrayPayload(contacts []*waE2E.ContactMessage) []webhookContactPayload {
	result := make([]webhookContactPayload, 0, len(contacts))
	for _, contact := range contacts {
		if contact == nil {
			continue
		}
		result = append(result, buildWebhookContactPayload(contact))
	}
	return result
}

/*
func buildConversionMessageTypes(msg *waE2E.Message, payload map[string]any) {
	extendedMessage := msg.GetExtendedTextMessage();
	if extendedMessage == nil {
		return
	} 
	if conversionSource := extendedMessage.ContextInfo.GetConversionSource(); conversionSource != "" {
		payload["conversion_source"] = conversionSource
	}
	if conversionData := extendedMessage.ContextInfo.GetConversionData(); conversionData != nil {
		payload["conversion_data"] = base64.StdEncoding.EncodeToString(conversionData)
	}
	if ctwaPayload := extendedMessage.ContextInfo.GetConversionData(); ctwaPayload != nil {
		payload["ctwa_payload"] = base64.StdEncoding.EncodeToString(ctwaPayload)
	}
	if entryPointConversionApp := extendedMessage.ContextInfo.GetEntryPointConversionApp(); entryPointConversionApp != "" {
		payload["entry_point_conversion_app"] = entryPointConversionApp
	}
	if entryPointConversionSource := extendedMessage.ContextInfo.GetEntryPointConversionSource(); entryPointConversionSource != "" {
		payload["entry_point_conversion_source"] = entryPointConversionSource
		payload["entry_point_conversion_delay_seconds"] = extendedMessage.ContextInfo.GetEntryPointConversionDelaySeconds()
	}

	if utm := extendedMessage.ContextInfo.GetUtm(); utm != nil {
		utmPayload := make(map[string]any)
		utmPayload["source"] = utm.GetUtmSource()
		utmPayload["campaign"] = utm.GetUtmCampaign()
		payload["utm"] = utmPayload
	}

	if externalAdReply := extendedMessage.ContextInfo.GetExternalAdReply(); externalAdReply != nil {
		adReplyPayload := make(map[string]any)

		adReplyPayload["title"] = externalAdReply.GetTitle()
		//adReplyPayload["body"] = externalAdReply.GetBody()
		//adReplyPayload["ad_context_preview_dismissed"] = externalAdReply.GetAdContextPreviewDismissed()
		//adReplyPayload["wtwa_ad_format"] = externalAdReply.GetWtwaAdFormat()

		adReplyPayload["media_type"] = externalAdReply.GetMediaType()
		adReplyPayload["media_url"] = externalAdReply.GetMediaURL()
		//adReplyPayload["original_image_url"] = externalAdReply.GetOriginalImageURL()

		if thumbnail:= externalAdReply.GetThumbnail(); thumbnail != nil {
			adReplyPayload["thumbnail"] = base64.StdEncoding.EncodeToString(thumbnail)
		} else {
			adReplyPayload["thumbnail"] = ""
		}
		
		//adReplyPayload["greeting_message_body"] = externalAdReply.GetGreetingMessageBody()

		adReplyPayload["source_app"] = externalAdReply.GetSourceApp()
		adReplyPayload["source_id"] = externalAdReply.GetSourceID()
		adReplyPayload["source_type"] = externalAdReply.GetSourceType()
		adReplyPayload["source_url"] = externalAdReply.GetSourceURL()

		
		payload["external_ad_reply"] = adReplyPayload
	}
}
*/
