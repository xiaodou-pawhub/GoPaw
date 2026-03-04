// Package renderer provides cross-platform message rendering abstractions.
package renderer

import (
	"regexp"
	"strings"
)

// BlockType identifies the type of a message content block.
type BlockType string

const (
	BlockMarkdown BlockType = "markdown"
	BlockImage    BlockType = "image"
	BlockDivider  BlockType = "divider"
	BlockHeader   BlockType = "header"
)

// MessageBlock represents a single atomic unit of a message (e.g. a paragraph of text, an image).
type MessageBlock struct {
	Type BlockType
	// Content holds the primary data (text for markdown, URL/ref for image).
	Content string
	// Title is used for headers or image captions.
	Title string
}

// ParseMarkdown converts a raw markdown string into a list of MessageBlocks.
// Currently, it handles simple text blocks and splits them by double newlines or horizontal rules.
// It also normalises some markdown syntax for better compatibility with IM platforms.
func ParseMarkdown(raw string) []MessageBlock {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	// Normalise markdown for IM platforms (e.g. Feishu cards don't support # headers inside markdown tags)
	raw = NormaliseMarkdown(raw)

	// In a full implementation, we might use a proper markdown parser (goldmark).
	// For now, we'll treat the entire thing as one markdown block for simplicity,
	// but structured so that it can be split into multiple blocks in the future.
	return []MessageBlock{
		{
			Type:    BlockMarkdown,
			Content: raw,
		},
	}
}

// NormaliseMarkdown converts standard markdown into a dialect that is more
// compatible with IM platform card systems (like Feishu/DingTalk).
func NormaliseMarkdown(text string) string {
	// 1. Convert headers to bold (Feishu cards don't support # in markdown elements)
	reHeader := regexp.MustCompile(`(?m)^#{1,6}\s+(.*)$`)
	text = reHeader.ReplaceAllString(text, "**$1**")

	// 2. Escape HTML tags if any (depending on platform needs)
	// text = strings.ReplaceAll(text, "<", "&lt;")
	// text = strings.ReplaceAll(text, ">", "&gt;")

	return strings.TrimSpace(text)
}
