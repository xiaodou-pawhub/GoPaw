package builtin

import (
	"context"
	"fmt"

	"github.com/gopaw/gopaw/internal/cron"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&CronRemoveTool{})
}

type CronRemoveTool struct {
	service *cron.CronService
}

func (t *CronRemoveTool) Name() string { return "cron_remove" }

func (t *CronRemoveTool) Description() string {
	return "Cancel a scheduled task."
}

func (t *CronRemoveTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"job_id": {
				Type:        "string",
				Description: "The ID of the job to remove (see cron_list).",
			},
		},
		Required: []string{"job_id"},
	}
}

func (t *CronRemoveTool) SetCronService(s *cron.CronService) {
	t.service = s
}

func (t *CronRemoveTool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	if t.service == nil {
		return plugin.ErrorResult("cron service not initialized")
	}

	jobID, _ := args["job_id"].(string)
	if jobID == "" {
		return plugin.ErrorResult("job_id is required")
	}

	// Try to find the full ID if a short ID was provided
	// This is a simple prefix match convenience
	if len(jobID) < 36 {
		jobs := t.service.ListJobs()
		for _, j := range jobs {
			if len(j.ID) > len(jobID) && j.ID[:len(jobID)] == jobID {
				jobID = j.ID
				break
			}
		}
	}

	if err := t.service.RemoveJob(jobID); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to remove job: %v", err))
	}

	return plugin.NewToolResult(fmt.Sprintf("Job %s removed successfully.", jobID))
}
