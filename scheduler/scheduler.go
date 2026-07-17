package scheduler

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/Hex-4/bop/ai"
	"github.com/Hex-4/bop/tools"
	"github.com/Hex-4/bop/triggers"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	Cron               *cron.Cron
	Jobs               map[string]Job
	sessions           *triggers.SessionStore
	ExtraToolsProvider func(channelID string) map[string]tools.Tool
	Agent              *ai.Agent
}

type Job struct {
	ID          string       `json:"id"`
	CronExpr    string       `json:"cronExpr,omitempty"` // Only for recurring jobs
	CronEntryID cron.EntryID `json:"cronEntryId,omitempty"`
	FireAt      time.Time    `json:"fireAt,omitempty"` // Only for one-shots
	Prompt      string       `json:"prompt"`
	SessionID   string       `json:"sessionId"`
	Silent      bool         `json:"silent"`
}

func NewScheduler(agent *ai.Agent) *Scheduler {
	sessions := triggers.SessionStore{}

	return &Scheduler{
		Cron:     cron.New(),
		Jobs:     make(map[string]Job),
		sessions: &sessions,
		Agent:    agent,
	}
}

func generateJobID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b) // e.g. "a3f1b20c"
}

func (s *Scheduler) AddCron(expression string, prompt string, sessionID string, silent bool) (string, error) {
	jobID := generateJobID()
	wrappedPrompt := "The following is a background cron job, not a live user message. Your history may include past iterations of this job. Job ID: " + jobID + ". Use tools as normal. Execute the following: " + prompt

	var cronFunc func()
	if silent {
		cronFunc = func() {
			prompt := s.Agent.SystemPrompt()
			niceTimeString := time.Now().Format("January 1 2006 at 15:04:05 MST")
			prompt = prompt + " (current time: " + niceTimeString + ")"

			history, _ := s.sessions.Load(jobID)
			history = append(history, ai.Message{Role: "system", Content: wrappedPrompt})

			messages := []ai.Message{
				{Role: "system", Content: prompt},
			}
			messages = append(messages, history...)

			newMessages, err := s.Agent.Ask(messages, nil)
			if err != nil {
				fmt.Printf("cron job failed: %v\n", err)
			}
			history = append(history, newMessages...)
			s.sessions.Save(jobID, history)
		}
	} else {
		cronFunc = func() {
			prompt := s.Agent.SystemPrompt()
			niceTimeString := time.Now().Format("January 1 2006 at 15:04:05 MST")
			prompt = prompt + " (current time: " + niceTimeString + ")"

			history, _ := s.sessions.Load(jobID)
			history = append(history, ai.Message{Role: "system", Content: wrappedPrompt})

			messages := []ai.Message{
				{Role: "system", Content: prompt},
			}
			messages = append(messages, history...)

			channelID := strings.TrimPrefix(sessionID, "discord:")

			newMessages, err := s.Agent.Ask(messages, s.ExtraToolsProvider(channelID))
			if err != nil {
				fmt.Printf("cron job failed: %v\n", err)
				return
			}
			history = append(history, newMessages...)
			s.sessions.Save(jobID, history)
		}
	}
	entryID, err := s.Cron.AddFunc(expression, cronFunc)
	if err != nil {
		return "", fmt.Errorf("invalid cron expression: %w", err)
	}
	s.Jobs[jobID] = Job{
		CronExpr:    expression,
		ID:          jobID,
		CronEntryID: entryID,
		Prompt:      prompt,
		SessionID:   sessionID,
		Silent:      silent,
	}
	return jobID, nil
}

func (s *Scheduler) AddOneShot(fireAt time.Time, prompt string, sessionID string, silent bool) string {
	jobID := generateJobID()
	wrappedPrompt := "The following is a scheduled one-shot job, not a live user message. Job ID: " + jobID + ". Use tools as normal. Execute the following: " + prompt

	var oneShotFunc func()
	if silent {
		oneShotFunc = func() {
			prompt := s.Agent.SystemPrompt()
			niceTimeString := time.Now().Format("January 1 2006 at 15:04:05 MST")
			prompt = prompt + " (current time: " + niceTimeString + ")"

			messages := []ai.Message{
				{Role: "system", Content: prompt},
			}
			history, _ := s.sessions.Load(jobID)
			history = append(history, ai.Message{Role: "system", Content: wrappedPrompt})
			messages = append(messages, history...)
			newMessages, err := s.Agent.Ask(messages, nil)
			if err != nil {
				fmt.Printf("one-shot job failed: %v\n", err)
			}
			history = append(history, newMessages...)
			s.sessions.Save(jobID, history)
			delete(s.Jobs, jobID)
		}
	} else {
		oneShotFunc = func() {
			prompt := s.Agent.SystemPrompt()
			niceTimeString := time.Now().Format("January 1 2006 at 15:04:05 MST")
			prompt = prompt + " (current time: " + niceTimeString + ")"

			messages := []ai.Message{
				{Role: "system", Content: prompt},
			}
			history, _ := s.sessions.Load(jobID)
			history = append(history, ai.Message{Role: "system", Content: wrappedPrompt})
			messages = append(messages, history...)

			channelID := strings.TrimPrefix(sessionID, "discord:")

			newMessages, err := s.Agent.Ask(messages, s.ExtraToolsProvider(channelID))

			if err != nil {
				fmt.Printf("one-shot job failed: %v\n", err)
				return
			}
			history = append(history, newMessages...)
			s.sessions.Save(jobID, history)
			delete(s.Jobs, jobID)
		}
	}
	time.AfterFunc(time.Until(fireAt), oneShotFunc)

	s.Jobs[jobID] = Job{
		ID:        jobID,
		FireAt:    fireAt,
		Prompt:    prompt,
		SessionID: sessionID,
		Silent:    silent,
	}
	return jobID
}

func (s *Scheduler) RemoveJob(jobID string) error {
	job, ok := s.Jobs[jobID]
	if !ok {
		return fmt.Errorf("Could not find job ID")
	}
	if job.CronExpr != "" {
		s.Cron.Remove(job.CronEntryID)
	}
	delete(s.Jobs, jobID)
	return nil
}
