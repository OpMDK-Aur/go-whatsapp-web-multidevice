# Changes made
## Enriched webhook media payloads (event_message.go)

When WhatsappAutoDownloadMedia is disabled, media messages now include all fields needed to download the media independently:

{
  "media_type": "audio",
  "audio": {
    "url": "https://mmg.whatsapp.net/...",
    "direct_path": "/v/t62.7114-24/...",
    "mime_type": "audio/ogg; codecs=opus",
    "file_size": 185380,
    "media_key": "ljYL0Xirah9x4Vc+C4rIlu7EAh4DzDHSpMy6TI9Zn6A=",
    "file_sha256": "zZC05cr/...",
    "file_enc_sha256": "v8pwawgQ5...",
    "is_ptt": true
  }
}

Each media type includes its own extra fields:
- audio: is_ptt
- image/video/video_note: caption
- document: filename, caption

A top-level media_type field is now always set (e.g. "image", "audio", "document", etc.) for all media messages, regardless of whether auto-download is enabled.


## New streaming download endpoint

GET /message/:message_id/download/stream?phone=<chat_jid>

Downloads and decrypts the media using the stored metadata, then streams the raw bytes directly to the HTTP client — no file is written to disk. Response headers include:
- Content-Type (auto-detected from file bytes)
- Content-Disposition: attachment; filename="..."
- Content-Length
- X-Media-Type (e.g. image, audio, etc.)
