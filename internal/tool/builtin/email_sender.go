package builtin

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/smtp"
	"regexp"
	"strings"

	"github.com/gopaw/gopaw/internal/settings"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/jordan-wright/email"
)

func init() {
	tool.Register(&EmailSenderTool{})
}

type EmailSenderTool struct {
	settings *settings.Store
	store    plugin.MediaStore
}

type emailConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	SSL      bool   `json:"ssl"`
}

func (t *EmailSenderTool) Name() string { return "email_sender" }

func (t *EmailSenderTool) Description() string {
	return "Send an email with optional attachments. Supports HTML body and media:// references for attachments. " +
		"Requires SMTP configuration in settings. Requires user approval."
}

func (t *EmailSenderTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"to": {
				Type:        "string",
				Description: "Recipient email address.",
			},
			"subject": {
				Type:        "string",
				Description: "Subject of the email.",
			},
			"body": {
				Type:        "string",
				Description: "Body of the email (HTML or plain text).",
			},
			"attachments": {
				Type:        "array",
				Description: "List of media:// references to attach.",
			},
		},
		Required: []string{"to", "subject", "body"},
	}
}

func (t *EmailSenderTool) SetSettingsStore(s *settings.Store) {
	t.settings = s
}

func (t *EmailSenderTool) SetMediaStore(s plugin.MediaStore) {
	t.store = s
}

func (t *EmailSenderTool) Summary(args map[string]interface{}) string {
	to, _ := args["to"].(string)
	subject, _ := args["subject"].(string)
	return fmt.Sprintf("📧 **发送邮件**\n- **收件人**: %s\n- **主题**: %s", to, subject)
}

func (t *EmailSenderTool) RequireApproval(args map[string]interface{}) bool {
	return true // Always require approval for sending emails
}

func (t *EmailSenderTool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	if t.settings == nil {
		return plugin.ErrorResult("settings store not initialized")
	}

	// 1. Load Config
	cfgJSON, err := t.settings.GetChannelConfig("email")
	if err != nil || cfgJSON == "{}" {
		return plugin.ErrorResult("Email SMTP is not configured. Please configure 'email' channel in settings.")
	}

	var cfg emailConfig
	if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to parse email config: %v", err))
	}

	// 2. Prepare Email
	to, _ := args["to"].(string)
	subject, _ := args["subject"].(string)
	body, _ := args["body"].(string)
	attRefs, _ := args["attachments"].([]interface{})

	e := email.NewEmail()
	e.From = cfg.From
	if e.From == "" {
		e.From = cfg.Username
	}
	e.To = []string{to}
	e.Subject = subject

	// 智能检测 HTML 并设置双重格式（最佳兼容性）
	if isHTML(body) {
		e.HTML = []byte(body)           // HTML 版本（富文本，现代客户端）
		e.Text = []byte(stripHTML(body)) // 纯文本版本（兼容性，老式客户端）
	} else {
		e.Text = []byte(body)           // 纯文本
	}

	// 3. Handle Attachments
	for _, ref := range attRefs {
		refStr, ok := ref.(string)
		if !ok || !strings.HasPrefix(refStr, "media://") {
			continue
		}
		if t.store == nil {
			continue
		}
		localPath, err := t.store.Resolve(refStr)
		if err != nil {
			continue
		}
		_, err = e.AttachFile(localPath)
		if err != nil {
			return plugin.ErrorResult(fmt.Sprintf("failed to attach file %s: %v", refStr, err))
		}
	}

	// 4. Send Email
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	var sendErr error
	if cfg.SSL {
		// SSL mode (port 465) - establish TLS connection first
		sendErr = sendWithSSL(e, addr, cfg.Username, cfg.Password, cfg.Host)
	} else {
		// STARTTLS mode (port 587) - use standard email.Send
		sendErr = e.Send(addr, auth)
	}

	if sendErr != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to send email: %v", sendErr))
	}

	return plugin.NewToolResult(fmt.Sprintf("Email sent successfully to %s", to))
}

// sendWithSSL sends email using SSL/TLS connection (for port 465)
func sendWithSSL(e *email.Email, addr, username, password, host string) error {
	// Create TLS configuration
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         host,
	}

	// Establish SSL connection
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("SSL connection failed: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("create SMTP client failed: %w", err)
	}
	defer client.Close()

	// Authenticate
	auth := smtp.PlainAuth("", username, password, host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Set sender
	if err := client.Mail(e.From); err != nil {
		return fmt.Errorf("set sender failed: %w", err)
	}

	// Set recipients
	for _, to := range e.To {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("set recipient failed: %w", err)
		}
	}

	// Get data writer
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("get data writer failed: %w", err)
	}

	// Write email headers
	fmt.Fprintf(w, "From: %s\r\n", e.From)
	fmt.Fprintf(w, "To: %s\r\n", e.To[0])
	fmt.Fprintf(w, "Subject: %s\r\n", e.Subject)

	// Write body (HTML or Text)
	if len(e.HTML) > 0 {
		fmt.Fprintf(w, "Content-Type: text/html; charset=UTF-8\r\n")
		fmt.Fprintf(w, "\r\n")
		if _, err := w.Write(e.HTML); err != nil {
			return fmt.Errorf("write HTML body failed: %w", err)
		}
	} else if len(e.Text) > 0 {
		fmt.Fprintf(w, "Content-Type: text/plain; charset=UTF-8\r\n")
		fmt.Fprintf(w, "\r\n")
		if _, err := w.Write(e.Text); err != nil {
			return fmt.Errorf("write text body failed: %w", err)
		}
	} else {
		return fmt.Errorf("email has no body content")
	}

	// Handle attachments - use email library's built-in method to write attachments
	for _, attachment := range e.Attachments {
		// email library handles attachment encoding internally
		if _, err := w.Write(attachment.Content); err != nil {
			return fmt.Errorf("write attachment failed: %w", err)
		}
	}

	// Close data writer
	if err := w.Close(); err != nil {
		return fmt.Errorf("close data writer failed: %w", err)
	}

	// Quit SMTP session
	return client.Quit()
}

// isHTML detects if content contains HTML tags.
func isHTML(content string) bool {
	lower := strings.ToLower(content)
	// Check for common HTML tags
	htmlTags := []string{
		"<html", "<head", "<body", "<div", "<span",
		"<h1>", "<h2>", "<h3>", "<h4>", "<h5>", "<h6>",
		"<p>", "<br", "<hr", "<ul>", "<ol>", "<li>",
		"<table", "<tr>", "<td>", "<th>",
		"<strong>", "<b>", "<em>", "<i>", "<u>",
		"<a ", "<img ", "<style", "<!doctype",
	}
	for _, tag := range htmlTags {
		if strings.Contains(lower, tag) {
			return true
		}
	}
	return false
}

// stripHTML removes HTML tags from content, returning plain text.
func stripHTML(html string) string {
	// Remove HTML tags using regex
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, "")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	// Normalize whitespace
	text = strings.TrimSpace(text)
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	return text
}
