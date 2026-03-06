package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/smtp"
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
	
	// Detect if body is HTML
	if strings.Contains(strings.ToLower(body), "<html>") || strings.Contains(strings.ToLower(body), "</div>") {
		e.HTML = []byte(body)
	} else {
		e.Text = []byte(body)
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

	// 4. Send
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	// Note: Standard library net/smtp doesn't handle SSL (Port 465) directly easily. 
	// For now we use the library's Send method which handles standard SMTP.
	// Future refinement: add TLS/SSL specific handling if port is 465.
	err = e.Send(addr, auth)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to send email: %v", err))
	}

	return plugin.NewToolResult(fmt.Sprintf("Email sent successfully to %s", to))
}
