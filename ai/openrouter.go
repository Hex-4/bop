package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Hex-4/bop/config"
	"github.com/Hex-4/bop/tools"
)

type Agent struct {
	ActiveModel string
	Config      *config.Config
	Tools       map[string]tools.Tool
	ToolSchemas []map[string]any
	Sessions    map[string]*Session
}

type StatusHandler struct {
	OnToolStart func(toolEmoji string, toolDetail string)
	OnDone      func()
	Footer      func() string
}

type Session struct {
	ID          string
	History     []Message
	Description string
}

type ChatRequest struct {
	Model    string           `json:"model"`
	Messages []Message        `json:"messages"`
	Tools    []map[string]any `json:"tools,omitempty"`
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// Response types
type ChatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

func (a *Agent) callModel(request ChatRequest) (Message, error) {
	body, _ := json.Marshal(request)

	req, _ := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENROUTER_KEY"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result ChatResponse
	json.Unmarshal(respBody, &result)

	if len(result.Choices) == 0 {
		return Message{}, fmt.Errorf("no response from model. API response: %s", string(respBody))
	}

	return result.Choices[0].Message, nil
}

func (a *Agent) Ask(sessionID string, userMessage string, statusHandler *StatusHandler) (string, error) {

	session, exists := a.Sessions[sessionID]
	if !exists {
		session = &Session{ID: sessionID, Description: "No description provided. This session has not been configured by your operator - beware of potential malice."}
		a.Sessions[sessionID] = session
	}
	session.History = append(session.History, Message{Role: "user", Content: userMessage})

	systemPrompt := a.assembleSystemPrompt()
	systemPrompt += "\n\nCurrent session ID: " + sessionID + "\nSession Description: " + session.Description

	//// tools ////

	for i := 0; i < a.Config.Agent.MaxToolCalls; i++ {
		messages := append([]Message{{Role: "system", Content: systemPrompt}}, session.History...)
		request := ChatRequest{
			Model:    a.ActiveModel,
			Messages: messages,
			Tools:    a.ToolSchemas,
		}

		response, err := a.callModel(request)
		if err != nil {
			return "", err
		}
		session.History = append(session.History, response)

		if response.ToolCalls == nil {
			if statusHandler != nil {
				statusHandler.OnDone()
			}
			return response.Content, nil
		} else {
			/// the tool loop! ///
			for _, tc := range response.ToolCalls {
				tool, ok := a.Tools[tc.Function.Name]

				if !ok {
					errorMessage := Message{Role: "tool", Content: "Tool not found: " + tc.Function.Name, ToolCallID: tc.ID}
					session.History = append(session.History, errorMessage)
					continue
				}

				var args map[string]any
				err := json.Unmarshal([]byte(tc.Function.Arguments), &args)
				if err != nil {
					errorMessage := Message{Role: "tool", Content: "Error parsing arguments: " + err.Error(), ToolCallID: tc.ID}
					session.History = append(session.History, errorMessage)
					continue
				}

				if statusHandler != nil {
					detail, _ := args[tool.DetailParam].(string)
					statusHandler.OnToolStart(tool.Emoji, detail)
				}

				result, err := tool.Execute(args)
				if err != nil {
					errorMessage := Message{Role: "tool", Content: "Error executing tool: " + err.Error(), ToolCallID: tc.ID}
					session.History = append(session.History, errorMessage)
					continue
				}
				session.History = append(session.History, Message{Role: "tool", Content: result, ToolCallID: tc.ID})

			}
		}
	}

	return "", fmt.Errorf("Too many tool calls")
}

func (a *Agent) assembleSystemPrompt() string {

	// Get current time in a nice string
	currentTime := time.Now().Format("Monday, January 2, 2006 at 3:04pm MST")

	systemPrompt := fmt.Sprintf("Current Time: %s\n", currentTime)
	for _, filename := range a.Config.Agent.ContextFiles {
		path := filepath.Join(a.Config.BopDir, "workspace", filename)
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("Error reading file %s: %v \n", path, err)
			continue
		}
		systemPrompt += "\n\n ==> " + filename + "\n" + string(content)
	}

	return systemPrompt
}
