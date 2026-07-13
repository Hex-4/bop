package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Hex-4/bop/config"
	"github.com/Hex-4/bop/tools"
)

type Agent struct {
	ActiveModel string
	Config      *config.Config
	Sessions    map[string]*Session
}

type Sink interface {
	Send(text string) error
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

func (a *Agent) Ask(messages []Message, toolList map[string]tools.Tool) ([]Message, error) {

	// generate tool schemas
	toolSchemas := tools.NewSchemaList(toolList)

	newMessages := []Message{}

	//// tools ////

	for i := 0; i < a.Config.Agent.MaxToolCalls; i++ {
		request := ChatRequest{
			Model:    a.ActiveModel,
			Messages: append(newMessages, messages...),
			Tools:    toolSchemas,
		}

		response, err := a.callModel(request)
		if err != nil {
			return newMessages, err
		}
		newMessages = append(newMessages, response)

		if response.ToolCalls == nil {
			return newMessages, nil
		} else {
			/// the tool loop! ///
			for _, tc := range response.ToolCalls {
				tool, ok := toolList[tc.Function.Name]

				if !ok {
					errorMessage := Message{Role: "tool", Content: "Tool not found: " + tc.Function.Name, ToolCallID: tc.ID}
					newMessages = append(newMessages, errorMessage)
					continue
				}

				var args map[string]any
				err := json.Unmarshal([]byte(tc.Function.Arguments), &args)
				if err != nil {
					errorMessage := Message{Role: "tool", Content: "Error parsing arguments: " + err.Error(), ToolCallID: tc.ID}
					newMessages = append(newMessages, errorMessage)
					continue
				}

				result, err := tool.Execute(args)
				if err != nil {
					errorMessage := Message{Role: "tool", Content: "Error executing tool: " + err.Error(), ToolCallID: tc.ID}
					newMessages = append(newMessages, errorMessage)
					continue
				}
				newMessages = append(newMessages, Message{Role: "tool", Content: result, ToolCallID: tc.ID})

			}
		}
	}

	return newMessages, fmt.Errorf("Too many tool calls")
}

func (a *Agent) SystemPrompt() string {

	systemPrompt := "You are a Bop agent, a helpful assistant with useful tools, a design that lets you help out right on time, without being asked, and a personality that keeps things as real as a friend. You are just one agent of possibly many that work together to form a complete experience that the user talks to. WARNING: Bop takes a NON-STANDARD approach to user communication. YOU MUST USE THE SEND_MESSAGE TOOL TO COMMUNICATE WITH THE USER. ANY CONTENT YOU OUTPUT WILL NOT BE FORWARDED."
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
