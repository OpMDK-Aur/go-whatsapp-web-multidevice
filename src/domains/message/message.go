package message

type GenericResponse struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}

type RevokeRequest struct {
	MessageID string `json:"message_id" uri:"message_id"`
	Phone     string `json:"phone" form:"phone"`
}

type DeleteRequest struct {
	MessageID string `json:"message_id" uri:"message_id"`
	Phone     string `json:"phone" form:"phone"`
}

type ReactionRequest struct {
	MessageID string `json:"message_id" form:"message_id"`
	Phone     string `json:"phone" form:"phone"`
	Emoji     string `json:"emoji" form:"emoji"`
}

type UpdateMessageRequest struct {
	MessageID string `json:"message_id" uri:"message_id"`
	Message   string `json:"message" form:"message"`
	Phone     string `json:"phone" form:"phone"`
}

type MarkAsReadRequest struct {
	MessageID string `json:"message_id" uri:"message_id"`
	Phone     string `json:"phone" form:"phone"`
}

type MarkAsReadBulkRequest struct {
	Phone      string   `json:"phone" form:"phone"`
	MessageIDs []string `json:"message_ids" form:"message_ids"`
}

type MarkAsReadBulkResponse struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type StarRequest struct {
	MessageID string `json:"message_id" uri:"message_id"`
	Phone     string `json:"phone" form:"phone"`
	IsStarred bool   `json:"is_starred"`
}

type DownloadMediaRequest struct {
	MessageID string `json:"message_id" uri:"message_id"`
	Phone     string `json:"phone" form:"phone"`
}

type DownloadMediaResponse struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
	MediaType string `json:"media_type"`
	Filename  string `json:"filename"`
	FilePath  string `json:"file_path"`
	FileSize  int64  `json:"file_size"`
}

// holds decrypted media bytes and metadata for streaming to HTTP clients.
type GetMediaData struct {
	Data      []byte
	MimeType  string
	Filename  string
	MediaType string
	FileSize  int64
}

type DeleteMediaRequest struct {
	MessageID string `json:"message_id" uri:"message_id"`
	Phone     string `json:"phone" form:"phone"`
	FilePath  string `json:"file_path" form:"file_path"`
}
