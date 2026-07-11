package scheduler

import (
	"fmt"
	"strings"
	"time"

	"github.com/Hex-4/bop/tools"
)

func newCreateCron(s *Scheduler) tools.Tool {
	return tools.Tool{
		Name:        "create_cron",
		Description: "Create a new, recurring cron job",
		Emoji:       "⏰",
		Parameters: map[string]tools.Parameter{
			"schedule": {
				Type:        "string",
				Description: "Standard 5-field cron expression (minute hour day month weekday). Examples: '0 21 * * *' (daily 9pm), '*/30 * * * *' (every 30 min), '0 9 * * 1-5' (weekdays 9am). Runs in local time.",
				Required:    true,
			},
			"prompt": {
				Type:        "string",
				Description: "The instruction to send to the agent when the job fires. Write it as a standalone task — it will run as an automated job, not a live conversation.",
				Required:    true,
			},
			"session_id": {
				Type:        "string",
				Description: "The session ID to send the result to. Use the current session ID unless the user specifies otherwise.",
				Required:    true,
			},
			"silent": {
				Type:        "boolean",
				Description: "If true, the job runs but the response is not sent to Discord. Useful for background tasks like updating memory files.",
				Required:    false,
			},
		},
		DetailParam: "prompt",
		Execute: func(args map[string]any) (string, error) {
			silent, _ := args["silent"].(bool)
			id, err := s.AddCron(tools.ArgString(args, "schedule"), tools.ArgString(args, "prompt"), tools.ArgString(args, "session_id"), silent)
			if err != nil {
				return "Error creating cron job: " + err.Error(), nil
			}
			return "Job created with ID: " + id, nil
		},
	}
}

func newScheduleOnce(s *Scheduler) tools.Tool {
	return tools.Tool{
		Name:        "schedule_once",
		Description: "Schedule a one-time job",
		Emoji:       "⏰",
		Parameters: map[string]tools.Parameter{
			"fire_at": {
				Type:        "string",
				Description: "When to fire, in YYYY-MM-DDTHH:MM:SS format, local time (e.g. 2026-04-12T21:00:00).",
				Required:    true,
			},
			"prompt": {
				Type:        "string",
				Description: "The instruction to send to the agent when the job fires. Write it as a standalone task — it will run as an automated job, not a live conversation.",
				Required:    true,
			},
			"session_id": {
				Type:        "string",
				Description: "The session ID to send the result to. Use the current session ID unless the user specifies otherwise.",
				Required:    true,
			},
			"silent": {
				Type:        "boolean",
				Description: "If true, the job runs but the response is not sent to Discord. Useful for background tasks like updating memory files.",
				Required:    false,
			},
		},
		DetailParam: "prompt",
		Execute: func(args map[string]any) (string, error) {
			timeStr := tools.ArgString(args, "fire_at")
			fireAt, err := time.ParseInLocation("2006-01-02T15:04:05", timeStr, time.Local)
			if err != nil {
				return "Error parsing fire time: " + err.Error(), nil
			}
			silent, _ := args["silent"].(bool)
			id := s.AddOneShot(fireAt, tools.ArgString(args, "prompt"), tools.ArgString(args, "session_id"), silent)
			return "Job created with ID: " + id, nil
		},
	}
}

func newRemoveJob(s *Scheduler) tools.Tool {
	return tools.Tool{
		Name:        "remove_job",
		Description: "Remove a job by ID",
		Emoji:       "❌",
		Parameters: map[string]tools.Parameter{
			"job_id": {
				Type:        "string",
				Description: "The ID of the job to remove. Use list_jobs to find this.",
				Required:    true,
			},
		},
		DetailParam: "job_id",
		Execute: func(args map[string]any) (string, error) {
			id := tools.ArgString(args, "job_id")
			err := s.RemoveJob(id)
			if err != nil {
				return "Error removing job: " + err.Error(), nil
			}
			return "Job removed", nil
		},
	}
}

func newListJobs(s *Scheduler) tools.Tool {
	return tools.Tool{
		Name:        "list_jobs",
		Description: "List all scheduled jobs (recurring cron and one-shot). Returns job IDs needed for remove_job.",
		Emoji:       "📋",
		Parameters:  map[string]tools.Parameter{},
		DetailParam: "",
		Execute: func(args map[string]any) (string, error) {
			if len(s.Jobs) == 0 {
				return "No jobs found", nil
			}
			var lines []string
			for _, job := range s.Jobs {
				if job.CronExpr != "" {
					lines = append(lines, fmt.Sprintf("%s: %s (cron: %s)", job.ID, job.Prompt, job.CronExpr))
				} else {
					lines = append(lines, fmt.Sprintf("%s: %s (one-shot: %s)", job.ID, job.Prompt, job.FireAt.Format(time.RFC3339)))
				}
			}
			return strings.Join(lines, "\n"), nil
		},
	}
}

func (s *Scheduler) Tools() []tools.Tool {
	return []tools.Tool{
		newCreateCron(s),
		newScheduleOnce(s),
		newRemoveJob(s),
		newListJobs(s),
	}
}
