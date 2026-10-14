package utils

import (
	"cmp"
	"encoding/base64"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

type contactPayload struct {
	DisplayName string `json:"displayName"`
	VCard       string `json:"vcard"`
}

type locationPayload struct {
	DegreesLatitude  float64 `json:"degreesLatitude,omitempty"`
	DegreesLongitude float64 `json:"degreesLongitude,omitempty"`
	Name             string  `json:"name,omitempty"`
	Address          string  `json:"address,omitempty"`
	JPEGThumbnail    string  `json:"jpeg_thumbnail,omitempty"`
	IsLive           bool    `json:"isLive,omitempty"`
}

func PopulateQuotedSummary(msg *waE2E.Message, payload map[string]any) {
	if audio := msg.GetAudioMessage(); audio != nil {
		payload["type"] = "audio"
		payload["audio"] = audio.GetSeconds()
	} else if document := msg.GetDocumentMessage(); document != nil {
		payload["type"] = "document"
		payload["document"] = cmp.Or(document.GetTitle(), document.GetFileName())
	} else if img := msg.GetImageMessage(); img != nil {
		payload["type"] = "image"
		if thumb := img.GetJPEGThumbnail(); thumb != nil {
			payload["image"] = base64.StdEncoding.EncodeToString(thumb)
		}
	} else if sticker := msg.GetStickerMessage(); sticker != nil {
		payload["type"] = "sticker"
		if thumb := sticker.GetPngThumbnail(); thumb != nil {
			payload["sticker"] = base64.StdEncoding.EncodeToString(thumb)
		}
	} else if vid := msg.GetVideoMessage(); vid != nil {
		payload["type"] = "video"
		if thumb := vid.GetJPEGThumbnail(); thumb != nil {
			payload["video"] = base64.StdEncoding.EncodeToString(thumb)
		}
	} else if ptv := msg.GetPtvMessage(); ptv != nil {
		payload["type"] = "video_note"
		if thumb := ptv.GetJPEGThumbnail(); thumb != nil {
			payload["video_note"] = base64.StdEncoding.EncodeToString(thumb)
		}
	} else if contact := msg.GetContactMessage(); contact != nil {
		payload["type"] = "contact"
		payload["contact"] = contactPayload{
			DisplayName: contact.GetDisplayName(),
			VCard:       contact.GetVcard(),
		}
	} else if contacts := msg.GetContactsArrayMessage(); contacts != nil {
		payload["type"] = "contacts_array"
		c := contacts.GetContacts()
		payload["contacts_array"] = len(c)
	} else if location := msg.GetLocationMessage(); location != nil {
		payload["type"] = "location"
		loc := locationPayload{
			DegreesLatitude:  location.GetDegreesLatitude(),
			DegreesLongitude: location.GetDegreesLongitude(),
			Name:             location.GetName(),
			Address:          location.GetAddress(),
			IsLive:           location.GetIsLive(),
		}
		if thumb := location.GetJPEGThumbnail(); thumb != nil {
			loc.JPEGThumbnail = base64.StdEncoding.EncodeToString(thumb)
		}
		payload["location"] = loc
	} else if msg.GetLiveLocationMessage() != nil {
		payload["type"] = "live_location"
	} else if poll, _ := ExtractPollCreationMessage(msg); poll != nil {
		payload["type"] = "poll"
		payload["poll"] = poll.GetName()
	} else if event := msg.GetEventMessage(); event != nil {
		payload["type"] = "event"
		payload["event"] = event.GetName()
	} else if msg.GetConversation() != "" || msg.GetExtendedTextMessage() != nil {
		payload["type"] = "text"
	} else if msg.GetListMessage() != nil {
		payload["type"] = "list"
	} else if msg.GetButtonsMessage() != nil {
		payload["type"] = "buttons"
	} else if msg.GetTemplateMessage() != nil ||
		msg.GetHighlyStructuredMessage() != nil {
		payload["type"] = "template"
	} else if msg.GetProductMessage() != nil {
		payload["type"] = "product"
	} else if msg.GetOrderMessage() != nil {
		payload["type"] = "order"
	} else if msg.GetInteractiveMessage() != nil {
		payload["type"] = "interactive"
	} else if msg.GetListResponseMessage() != nil ||
		msg.GetButtonsResponseMessage() != nil ||
		msg.GetTemplateButtonReplyMessage() != nil ||
		msg.GetInteractiveResponseMessage() != nil {
		payload["type"] = "selection"
	} else if msg.GetGroupInviteMessage() != nil {
		payload["type"] = "group_invite"
	} else {
		payload["type"] = "unknown"
	}
}

// extractQuotedText returns user text or caption from msg without synthesizing
// fallback strings for contacts or locations.
func ExtractQuotedText(msg *waE2E.Message) string {
	if t := msg.GetConversation(); t != "" {
		return t
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		if t := cmp.Or(ext.GetText(), ext.GetMatchedText()); t != "" {
			return t
		}
	}
	if cap := ExtractMediaCaption(msg); cap != "" {
		return cap
	}
	return extractBusinessMessageText(msg)
}

func ReplyContextInfo(msg *waE2E.Message) *waE2E.ContextInfo {
	return cmp.Or(
		ExtractContextInfo(msg),
		msg.GetButtonsResponseMessage().GetContextInfo(),
		msg.GetListResponseMessage().GetContextInfo(),
		msg.GetTemplateButtonReplyMessage().GetContextInfo(),
		msg.GetInteractiveResponseMessage().GetContextInfo(),
		msg.GetContactsArrayMessage().GetContextInfo(),
		msg.GetPollCreationMessage().GetContextInfo(),
		msg.GetPollCreationMessageV2().GetContextInfo(),
		msg.GetPollCreationMessageV3().GetContextInfo(),
	)
}
