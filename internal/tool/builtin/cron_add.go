package builtin

import (
	"context"
	"fmt"

	"github.com/gopaw/gopaw/internal/cron"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&CronAddTool{})
}

type CronAddTool struct {
	service *cron.CronService
	channel string
	chatID  string
}

func (t *CronAddTool) Name() string { return "cron_add" }

func (t *CronAddTool) Description() string {
	return "Schedule a recurring task. The agent will execute the 'task' instruction periodically. " +
		"Schedule format is standard Cron: 'Second Minute Hour Dom Month Dow' (e.g. '0 30 9 * * *' for daily 9:30:00)."
}

func (t *CronAddTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"name": {
				Type:        "string",
				Description: "A short name for the task (e.g., 'Daily Report').",
			},
			"schedule": {
				Type:        "string",
				Description: "Cron expression (6 fields: s m h d m w). Example: '0 */10 * * * *' (every 10 min).",
			},
			"task": {
				Type:        "string",
				Description: "The natural language instruction for the agent to execute (e.g., 'Check GitHub trending and summarize').",
			},
		},
		Required: []string{"name", "schedule", "task"},
	}
}

func (t *CronAddTool) SetCronService(s *cron.CronService) {
	t.service = s
}

func (t *CronAddTool) SetContext(channel, chatID, session, user string) {
	t.channel = channel
	t.chatID = chatID // This is the real platform ChatID (e.g. oc_xxx)
}

func (t *CronAddTool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	if t.service == nil {
		return plugin.ErrorResult("cron service not initialized")
	}

	name, _ := args["name"].(string)
	schedule, _ := args["schedule"].(string)
	task, _ := args["task"].(string)

	if name == "" || schedule == "" || task == "" {
		return plugin.ErrorResult("name, schedule, and task are required")
	}

	// Use the agent's ID as target_id so the cron job runs with the correct agent.
	// Falls back to chatID (platform room ID) if agentID is not injected.
	targetID := t.chatID
	if agentID, ok := ctx.Value(tool.ContextKeyAgentID).(string); ok && agentID != "" {
		targetID = agentID
	}

	job, err := t.service.AddJob(name, schedule, task, t.channel, targetID)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to schedule job: %v", err))
	}

	return plugin.NewToolResult(fmt.Sprintf("Job '%s' (ID: %s) scheduled successfully. Next run determined by schedule '%s'.", job.Name, job.ID, job.Schedule))
}
