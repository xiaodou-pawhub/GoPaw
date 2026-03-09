// Package plugin defines advanced capabilities for channel plugins.
package plugin

import "github.com/gopaw/gopaw/pkg/types"

// ReactionCapable is an optional interface for plugins that support message reactions.
type ReactionCapable interface {
	AddReaction(channelID, messageTS, reaction string) error
	RemoveReaction(channelID, messageTS, reaction string) error
	ListReactions(channelID, messageTS string) ([]string, error)
}

// MessageEditor is an optional interface for plugins that support message editing.
type MessageEditor interface {
	EditMessage(channelID, messageTS, newContent string) error
	DeleteMessage(channelID, messageTS string) error
}

// ThreadCapable is an optional interface for plugins that support threaded conversations.
type ThreadCapable interface {
	SendThreadMessage(channelID, parentTS, content string) error
	GetThreadHistory(channelID, parentTS string) ([]types.Message, error)
}

// RichTextCapable is an optional interface for plugins that support rich text formatting.
type RichTextCapable interface {
	SendMarkdown(channelID, markdown string) error
	SendHTML(channelID, html string) error
	SendBlockKit(channelID string, blocks []Block) error
}

// Block represents a rich text block (Slack Block Kit style).
type Block struct {
	Type     string      `json:"type"`
	Text     *TextBlock  `json:"text,omitempty"`
	Elements []interface{} `json:"elements,omitempty"`
	Fields   []TextField `json:"fields,omitempty"`
}

// TextBlock represents a text element.
type TextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// TextField represents a field element.
type TextField struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// InteractiveCapable is an optional interface for plugins that support interactive components.
type InteractiveCapable interface {
	SendButton(channelID, text string, buttons []Button) error
	SendSelectMenu(channelID, text string, options []SelectOption) error
	SendMultiSelectMenu(channelID, text string, options []SelectOption) error
}

// Button represents an interactive button.
type Button struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Value string `json:"value"`
	Style string `json:"style,omitempty"` // primary, danger, success
}

// SelectOption represents a select menu option.
type SelectOption struct {
	Text  string `json:"text"`
	Value string `json:"value"`
}

// FileCapable is an optional interface for plugins that support file operations.
type FileCapable interface {
	SendFile(channelID, filePath, caption string) error
	SendImage(channelID, imagePath, caption string) error
	SendVideo(channelID, videoPath, caption string) error
	SendAudio(channelID, audioPath, caption string) error
}

// MediaMessage represents a media message.
type MediaMessage struct {
	Type      string `json:"type"`      // image, video, audio, file
	URL       string `json:"url"`       // remote URL or local path
	Caption   string `json:"caption"`   // optional caption
	Thumbnail string `json:"thumbnail"` // optional thumbnail for video
}

// ApprovalUICapable is an optional interface for plugins that support interactive approval UI.
type ApprovalUI interface {
	RequestApproval(channelID, userID, toolName, description string, options []ApprovalOption) (string, error)
	UpdateApproval(messageID, action, reason string) error
}

// ApprovalOption represents an approval action option.
type ApprovalOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value"`
}

// TypingCapable is an optional interface for plugins that support typing indicators.
type TypingCapable interface {
	SendTypingIndicator(channelID string) error
}

// PlaceholderCapable is an optional interface for plugins that support placeholder messages.
type PlaceholderCapable interface {
	SendPlaceholder(channelID, placeholder string) (string, error)
	UpdatePlaceholder(messageID, newContent string) error
}
