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
	SendBlockKit(channelID, blocks []Block) error
}

// Block represents a rich text block (Slack Block Kit style).
type Block struct {
	Type string      `json:"type"`
	Text *TextBlock  `json:"text,omitempty"`
	Elements []interface{} `json:"elements,omitempty"`
}

// TextBlock represents a text element.
type TextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
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
