package plugin

import "context"

// Standard Reaction types that CapabilityCoordinator can request.
// Individual plugins map these to platform-specific emoji or actions.
const (
	ReactionWait    = "wait"    // e.g., "EYES" or "👀"
	ReactionSuccess = "success" // e.g., "DONE" or "✅"
	ReactionError   = "error"   // e.g., "WRONG" or "❌"
)

// TypingCapable is an optional interface for plugins that can signal
// a "typing…" / "bot is composing" indicator to the user.
type TypingCapable interface {
	// StartTyping begins the typing indicator for chatID.
	// The returned stop function must be called when the agent finishes responding.
	StartTyping(ctx context.Context, chatID string) (stop func(), err error)
}

// ReactionCapable is an optional interface for plugins that support emoji
// reactions on individual messages.
type ReactionCapable interface {
	// AddReaction attaches a standard reaction to the message identified by messageID.
	AddReaction(ctx context.Context, chatID, messageID, reactionType string) error
	// RemoveReaction removes a previously added reaction.
	RemoveReaction(ctx context.Context, chatID, messageID, reactionType string) error
}

// PlaceholderCapable is an optional interface for plugins that can send a
// placeholder message immediately, to be replaced with the real answer later.
type PlaceholderCapable interface {
	// SendPlaceholder sends a placeholder message and returns its messageID.
	SendPlaceholder(ctx context.Context, chatID string) (messageID string, err error)
}

// MessageEditor is an optional interface for plugins that support editing a
// previously sent message in place (used to replace a placeholder).
type MessageEditor interface {
	// EditMessage replaces the content of the message identified by messageID.
	EditMessage(ctx context.Context, chatID, messageID, content string) error
}
