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

// PeerKind describes the type of conversation context.
type PeerKind string

const (
	PeerDirect  PeerKind = "direct"
	PeerGroup   PeerKind = "group"
	PeerChannel PeerKind = "channel"
)

// Message is the unified message model exchanged between channels and the agent.
type Message struct {
	// ID is a globally unique message identifier (UUID).
	ID string
	// SessionID identifies the conversation context (e.g. channel-user pair).
	SessionID string
	// ChatID identifies the specific chat room or group (if applicable).
	ChatID string
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
	// IsMentioned indicates whether the bot was @mentioned (relevant in group chats).
	IsMentioned bool
	// ThreadID is the thread or topic ID (e.g., Feishu group thread).
	ThreadID string
	// PeerKind describes the conversation type: direct, group, or channel.
	PeerKind PeerKind
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
	SessionID   string
	UserID      string
	ChatID      string // platform-level chat room ID (e.g. Feishu oc_xxx); used for send-back routing
	Channel     string
	Content     string
	MsgType     MessageType
	Files       []FileAttachment
	Metadata    map[string]string
	IsMentioned bool
}

// Response is the structured output returned by the agent after processing.
type Response struct {
	Content string
	MsgType MessageType
	Files   []FileAttachment
	Error   error
}
