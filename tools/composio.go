package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func CreateComposioSession() (string, error) {
	body := map[string]any{
		"user_id": "bop",
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://backend.composio.dev/api/v3/tool_router/session", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", os.Getenv("COMPOSIO_KEY"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	sessionID, ok := result["session_id"].(string)
	if !ok {
		return "", fmt.Errorf("session_id not found")
	}

	return sessionID, nil
}

func FetchComposioSchemas(sessionID string) ([]map[string]any, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://backend.composio.dev/api/v3/tool_router/session/%s/tools", sessionID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", os.Getenv("COMPOSIO_KEY"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

func NewComposioToolSlice(sessionID string, schemas []map[string]any) ([]Tool, error) {
	tools := make([]Tool, 0, len(schemas))
	for _, schema := range schemas {
		slug := schema["slug"].(string)
		tool := Tool{
			Name:        slug,
			Description: schema["description"].(string),
			Emoji:       "🖇️",
			RawSchema: map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        slug,
					"description": schema["description"].(string),
					"parameters":  schema["input_parameters"].(map[string]any),
				},
			},
			Execute: func(args map[string]any) (string, error) {
				body := map[string](any){
					"slug":      slug,
					"arguments": args,
				}

				jsonBody, err := json.Marshal(body)
				if err != nil {
					return "Tool call error: Could not encode arguments into JSON: " + err.Error(), nil
				}

				req, err := http.NewRequest("POST", fmt.Sprintf("https://backend.composio.dev/api/v3/tool_router/session/%s/execute_meta", sessionID), bytes.NewBuffer(jsonBody))
				if err != nil {
					return "Tool call error: Could not create request: " + err.Error(), nil
				}
				req.Header.Set("x-api-key", os.Getenv("COMPOSIO_KEY"))
				req.Header.Set("Content-Type", "application/json")

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return "Tool call error: Could not send request: " + err.Error(), nil
				}
				defer resp.Body.Close()

				var result struct {
					Data  map[string]any `json:"data"`
					Error string         `json:"error"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					return "Tool call error: Could not decode response: " + err.Error(), nil
				}
				if result.Error != "" {
					return "Tool call error: " + result.Error, nil
				}
				jsonData, err := json.Marshal(result.Data)
				if err != nil {
					return "Tool call error: Could not encode result into JSON: " + err.Error(), nil
				}
				return string(jsonData), nil
			},
		}
		tools = append(tools, tool)
	}
	return tools, nil
}
