package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
)

func newWebSearch(workspaceDir string) Tool {
	return Tool{
		Name:        "web_search",
		Description: "Search the web for information",
		Parameters: map[string]Parameter{
			"query": {
				Type:        "string",
				Description: "The search query",
				Required:    true,
			},
			"count": {
				Type:        "integer",
				Description: "The number of results to return. Default 5",
				Required:    false,
			},
		},
		Execute: func(args map[string]any) (string, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			numResults := 5 // default
			if n, ok := args["count"].(float64); ok {
				numResults = int(n)
			}
			body := map[string]any{
				"query":      ArgString(args, "query"),
				"numResults": numResults,
				"type":       "auto",
				"contents": map[string]any{
					"highlights": map[string]any{
						"maxCharacters": 2000,
					},
				},
			}
			jsonBody, err := json.Marshal(body)
			if err != nil {
				return "Tool call error: failed to build request: " + err.Error(), nil
			}

			req, err := http.NewRequestWithContext(ctx, "POST", "https://api.exa.ai/search", bytes.NewReader(jsonBody))
			if err != nil {
				return "Tool call error: failed to create request: " + err.Error(), nil
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("x-api-key", os.Getenv("EXA_KEY"))

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return "Tool call error: failed to send request: " + err.Error(), nil
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Sprintf("Tool call error: exa returned status %d", resp.StatusCode), nil
			}

			var result struct {
				Results []struct {
					Title      string   `json:"title"`
					URL        string   `json:"url"`
					Highlights []string `json:"highlights"`
				} `json:"results"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return "Tool call error: failed to decode response: " + err.Error(), nil
			}

			var sb strings.Builder
			for _, r := range result.Results {
				sb.WriteString(fmt.Sprintf("## %s\n(%s)\n\n", r.Title, r.URL))
				for _, h := range r.Highlights {
					sb.WriteString(fmt.Sprintf("- %s\n\n", h))
				}
			}
			return sb.String(), nil
		},
	}
}

func newWebFetch(_ string) Tool {
	return Tool{
		Name:        "web_fetch",
		Description: "Fetch the contents of a web page. Truncates to 50kb, use web_highlights instead for factual lookups",
		Parameters: map[string]Parameter{
			"url": {
				Type:        "string",
				Description: "The URL of the page to fetch",
				Required:    true,
			},
		},
		Execute: func(args map[string]any) (string, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			url := ArgString(args, "url")
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return "Tool call error: failed to create request: " + err.Error(), nil
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return "Tool call error: failed to send request: " + err.Error(), nil
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return "Tool call error: external request returned status " + strconv.Itoa(resp.StatusCode), nil
			}

			limitReader := io.LimitReader(resp.Body, 50*1024)

			markdown, err := htmltomarkdown.ConvertReader(limitReader, converter.WithDomain(url))
			if err != nil {
				return "Tool call error: failed to convert response: " + err.Error(), nil
			}
			return string(markdown), nil
		},
	}
}

func newWebHighlights(workspaceDir string) Tool {
	return Tool{
		Name:        "web_highlights",
		Description: "Get information-dense highlights from a web page for a given URL",
		Parameters: map[string]Parameter{
			"url": {
				Type:        "string",
				Description: "The URL of the web page to fetch",
				Required:    true,
			},
			"maxCharacters": {
				Type:        "integer",
				Description: "The maximum number of characters to fetch. Default 2000",
				Required:    false,
			},
			"subpages": {
				Type:        "integer",
				Description: "The number of subpages to fetch. Default 0",
				Required:    false,
			},
		},
		Execute: func(args map[string]any) (string, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			maxChars := 2000 // default
			if n, ok := args["maxCharacters"].(float64); ok {
				maxChars = int(n)
			}
			subpages := 0 // default
			if n, ok := args["subpages"].(float64); ok {
				subpages = int(n)
			}
			body := map[string]any{
				"urls": []string{ArgString(args, "url")},
				"highlights": map[string]any{
					"maxCharacters": maxChars,
				},
				"subpages": subpages,
			}
			jsonBody, err := json.Marshal(body)
			if err != nil {
				return "Tool call error: failed to build request: " + err.Error(), nil
			}

			req, err := http.NewRequestWithContext(ctx, "POST", "https://api.exa.ai/contents", bytes.NewReader(jsonBody))
			if err != nil {
				return "Tool call error: failed to create request: " + err.Error(), nil
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("x-api-key", os.Getenv("EXA_KEY"))

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return "Tool call error: failed to send request: " + err.Error(), nil
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Sprintf("Tool call error: exa returned status %d", resp.StatusCode), nil
			}

			var result struct {
				Results []struct {
					Title      string   `json:"title"`
					URL        string   `json:"url"`
					Highlights []string `json:"highlights"`
					Subpages   []struct {
						Title      string   `json:"title"`
						URL        string   `json:"url"`
						Highlights []string `json:"highlights"`
					} `json:"subpages"`
				} `json:"results"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return "Tool call error: failed to decode response: " + err.Error(), nil
			}

			var sb strings.Builder
			for _, r := range result.Results {
				sb.WriteString(fmt.Sprintf("## %s\n(%s)\n\n", r.Title, r.URL))
				for _, h := range r.Highlights {
					sb.WriteString(fmt.Sprintf("- %s\n\n", h))
				}
				for _, sp := range r.Subpages {
					sb.WriteString(fmt.Sprintf("### Subpage: %s\n(%s)\n\n", sp.Title, sp.URL))
					for _, h := range sp.Highlights {
						sb.WriteString(fmt.Sprintf("- %s\n\n", h))
					}
				}
			}
			return sb.String(), nil
		},
	}
}
