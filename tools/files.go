package tools

import (
	"os"
	"path/filepath"
)

func newReadFile(workspaceDir string) Tool {
	return Tool{
		Name:        "read_file",
		Description: "Read a file from the workspace",
		Parameters: map[string]Parameter{
			"path": {Type: "string", Description: "file path relative to workspace", Required: true},
		},
		Execute: func(args map[string]any) (string, error) {
			data, err := os.ReadFile(filepath.Join(workspaceDir, ArgString(args, "path")))
			return string(data), err
		},
	}
}

func newWriteFile(workspaceDir string) Tool {
	return Tool{
		Name:        "write_file",
		Description: "Write a file to the workspace",
		Parameters: map[string]Parameter{
			"path":    {Type: "string", Description: "file path relative to workspace", Required: true},
			"content": {Type: "string", Description: "file content", Required: true},
		},
		Execute: func(args map[string]any) (string, error) {
			return "", os.WriteFile(filepath.Join(workspaceDir, ArgString(args, "path")), []byte(ArgString(args, "content")), 0644)
		},
	}
}
