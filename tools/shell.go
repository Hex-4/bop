package tools

import (
	"context"
	"os/exec"
	"time"
)

func newShell(workspaceDir string) Tool {
	return Tool{
		Name:        "shell",
		Description: "Execute shell commands (using standard POSIX `sh`)",
		Parameters: map[string]Parameter{
			"command": {
				Type:        "string",
				Description: "The shell command to execute",
				Required:    true,
			},
			"directory": {
				Type:        "string",
				Description: "The directory to execute the command in. Defaults to your workspace directory",
				Required:    false,
			},
		},
		Execute: func(args map[string]any) (string, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			dir := workspaceDir
			if d := ArgString(args, "directory"); d != "" {
				dir = d
			}

			cmd := exec.CommandContext(ctx, "sh", "-c", ArgString(args, "command"))
			cmd.Dir = dir
			output, err := cmd.CombinedOutput()
			if err != nil {
				return string(output) + "\n" + err.Error(), nil
			}
			return string(output), nil
		},
	}
}
