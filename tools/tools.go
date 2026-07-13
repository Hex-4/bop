package tools

type Tool struct {
	Name        string
	Description string
	Parameters  map[string]Parameter // for the JSON schema sent to the model
	RawSchema   map[string]any       // (for complex schemas)
	Execute     func(args map[string]any) (string, error)
}

type Parameter struct {
	Type        string
	Description string
	Required    bool
}

// ArgString pulls a string value out of the args map.
func ArgString(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

func NewRegistry(workspaceDir string, externalToolSlice []Tool) map[string]Tool {
	tools := map[string]Tool{
		"read_file":      newReadFile(workspaceDir),
		"write_file":     newWriteFile(workspaceDir),
		"shell":          newShell(workspaceDir),
		"web_search":     newWebSearch(workspaceDir),
		"web_fetch":      newWebFetch(workspaceDir),
		"web_highlights": newWebHighlights(workspaceDir),
	}
	for _, t := range externalToolSlice {
		tools[t.Name] = t
	}
	return tools
}

func NewSchemaList(tools map[string]Tool) []map[string]any {
	schemas := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		schemas = append(schemas, t.ToSchema())
	}
	return schemas
}

func (t *Tool) ToSchema() map[string]any {
	if t.RawSchema != nil {
		return t.RawSchema
	}

	properties := make(map[string]any)
	for name, param := range t.Parameters {
		properties[name] = map[string]any{
			"type":        param.Type,
			"description": param.Description,
		}
	}

	required := []string{}
	for name, param := range t.Parameters {
		if param.Required {
			required = append(required, name)
		}
	}

	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"parameters": map[string]any{
				"type":       "object",
				"properties": properties,
				"required":   required,
			},
		},
	}
}
