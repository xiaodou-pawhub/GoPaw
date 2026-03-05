package builtin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gopaw/gopaw/internal/cron"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&CronListTool{})
}

type CronListTool struct {
	service *cron.CronService
}

func (t *CronListTool) Name() string { return "cron_list" }

func (t *CronListTool) Description() string {
	return "List all active scheduled tasks."
}

func (t *CronListTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type:       "object",
		Properties: map[string]plugin.ToolProperty{},
	}
}

func (t *CronListTool) SetCronService(s *cron.CronService) {
	t.service = s
}

func (t *CronListTool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	if t.service == nil {
		return plugin.ErrorResult("cron service not initialized")
	}

	jobs := t.service.ListJobs()
	if len(jobs) == 0 {
		return plugin.NewToolResult("No scheduled jobs.")
	}

	var sb strings.Builder
	sb.WriteString("### Scheduled Jobs\n\n")
	sb.WriteString("| ID | Name | Schedule | Task | Last Run |\n")
	sb.WriteString("|----|------|----------|------|----------|\n")

	for _, job := range jobs {
		lastRun := "Never"
		if job.LastRunAt != nil {
			lastRun = job.LastRunAt.Format(time.DateTime)
		}
		// Truncate ID for readability, but keep enough for unique identification
		shortID := job.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		
		fmt.Fprintf(&sb, "| %s | %s | `%s` | %s | %s |\n", 
			shortID, job.Name, job.Schedule, job.Task, lastRun)
	}

	return plugin.NewToolResult(sb.String())
}
