package feishu

import (
	"encoding/json"
	"fmt"

	"github.com/gopaw/gopaw/internal/renderer"
)

// FeishuCard represents the root structure of a Feishu Interactive Card 2.0.
type FeishuCard struct {
	Schema string `json:"schema"`
	Header struct {
		Title struct {
			Tag     string `json:"tag"`
			Content string `json:"content"`
		} `json:"title"`
		Template string `json:"template,omitempty"` // blue, wathet, turquoise, green, yellow, orange, red, carmine, violet, purple, indigo, grey
	} `json:"header"`
	Body struct {
		Elements []interface{} `json:"elements"`
	} `json:"body"`
}

// BuildCard converts standard renderer blocks into a Feishu Card JSON string.
func BuildCard(title string, blocks []renderer.MessageBlock, status string) (string, error) {
	card := FeishuCard{
		Schema: "2.0",
	}

	// 1. Set Header
	card.Header.Title.Tag = "plain_text"
	card.Header.Title.Content = title
	
	switch status {
	case "processing":
		card.Header.Template = "blue"
		if title == "" {
			card.Header.Title.Content = "🤖 思考中..."
		}
	case "success":
		card.Header.Template = "green"
	case "error":
		card.Header.Template = "red"
	default:
		card.Header.Template = "indigo"
	}

	// 2. Set Body Elements
	for _, b := range blocks {
		switch b.Type {
		case renderer.BlockMarkdown:
			element := map[string]interface{}{
				"tag":     "markdown",
				"content": b.Content,
			}
			card.Body.Elements = append(card.Body.Elements, element)
		
		case renderer.BlockDivider:
			card.Body.Elements = append(card.Body.Elements, map[string]string{"tag": "hr"})
		
		case renderer.BlockImage:
			element := map[string]interface{}{
				"tag":     "img",
				"img_key": b.Content,
				"alt": map[string]string{
					"tag":     "plain_text",
					"content": b.Title,
				},
				"mode": "fit_horizontal", // Better for large images
			}
			card.Body.Elements = append(card.Body.Elements, element)
		}
	}

	if len(card.Body.Elements) == 0 {
		card.Body.Elements = append(card.Body.Elements, map[string]string{
			"tag":     "markdown",
			"content": "_[无内容]_",
		})
	}

	data, err := json.Marshal(card)
	if err != nil {
		return "", fmt.Errorf("failed to marshal feishu card: %w", err)
	}

	return string(data), nil
}

// BuildPlaceholderCard returns a minimal card for "typing" state.
func BuildPlaceholderCard() (string, error) {
	return BuildCard("🤖 思考中...", nil, "processing")
}
