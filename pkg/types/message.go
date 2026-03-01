// Package types defines shared data types used across GoPaw modules.
package types

// MessageType is an enumeration of supported message content types.
type MessageType string

const (
	MsgTypeText     MessageType = "text"
	MsgTypeImage    MessageType = "image"
	MsgTypeFile     MessageType = "file"
	MsgTypeMarkdown MessageType = "markdown"
)

// Message is the unified message model exchanged between channels and the agent.
type Message struct {
	// ID is a globally unique message identifier (UUID).
	ID string
	// SessionID identifies the conversation context (e.g. channel-user pair).
	SessionID string
	// UserID is the platform-specific user identifier.
	UserID string
	// Channel is the originating channel name (e.g. "feishu", "console").
	Channel string
	// Content holds the textual payload of the message.
	Content string
	// MsgType describes the nature of the content.
	MsgType MessageType
	// Files contains any attached files.
	Files []FileAttachment
	// ReplyTo is the ID of the message being replied to, if any.
	ReplyTo string
	// Timestamp is a Unix millisecond timestamp.
	Timestamp int64
	// Metadata carries arbitrary key-value pairs for channel-specific data.
	Metadata map[string]string
}

// FileAttachment represents a file attached to a message.
type FileAttachment struct {
	// Name is the original filename.
	Name string
	// URL is where the file can be downloaded.
	URL string
	// Size is the file size in bytes.
	Size int64
	// MIMEType is the detected MIME type.
	MIMEType string
}

// Request is the normalised input handed to the agent for processing.
type Request struct {
	SessionID string
	UserID    string
	Channel   string
	Content   string
	MsgType   MessageType
	Files     []FileAttachment
	Metadata  map[string]string
}

// Response is the structured output returned by the agent after processing.
type Response struct {
	Content string
	MsgType MessageType
	Files   []FileAttachment
	Error   error
}
