package whatsapp

import (
	"context"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func isOwnJID(client *whatsmeow.Client, jid types.JID) bool {
	if client == nil || client.Store == nil {
		return false
	}
	jid = jid.ToNonAD()
	if own := client.Store.ID; own != nil && jid == own.ToNonAD() {
		return true
	}
	return !client.Store.LID.IsEmpty() && jid == client.Store.LID.ToNonAD()
}

func buildRepliedTo(ctx context.Context, client *whatsmeow.Client, evt *events.Message) map[string]any {
	msg := utils.UnwrapMessage(evt.Message)
	ci := utils.ReplyContextInfo(msg)
	if ci.GetStanzaID() == "" {
		return nil
	}
	quoted := utils.UnwrapMessage(ci.GetQuotedMessage())
	if quoted == nil {
		return nil
	}

	remoteJID := ci.GetRemoteJID()
	participant := ci.GetParticipant()
	replied := map[string]any{
		"id": ci.GetStanzaID(),
	}

	// Where the replied message lives: RemoteJID when WhatsApp sets it (status
	// replies, cross-chat quotes), otherwise the same chat as the reply.
	chat := evt.Info.Chat
	if remoteJID != "" {
		if jid, err := types.ParseJID(remoteJID); err == nil {
			chat = jid
		}
	}
	chatJID := chat.ToNonAD()
	if chatJID.Server == "lid" {
		replied["chat_lid"] = chatJID.String()
		chatJID = NormalizeJIDFromLID(ctx, chatJID, client).ToNonAD()
	}
	replied["chat_id"] = chatJID.String()

	if participant != "" {
		if jid, err := types.ParseJID(participant); err == nil {
			fromJID := jid.ToNonAD()
			if fromJID.Server == "lid" {
				replied["from_lid"] = fromJID.String()
				fromJID = NormalizeJIDFromLID(ctx, fromJID, client).ToNonAD()
			}
			replied["from"] = fromJID.String()
			replied["is_from_me"] = isOwnJID(client, jid)
		}
	}

	if body := utils.ExtractQuotedText(quoted); body != "" {
		replied["body"] = body
	}
	utils.PopulateQuotedSummary(quoted, replied)

	return replied
}
