package standardtools

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/c00/harnesser/types"
	"go.yaml.in/yaml/v4"
)

var standardTools = []types.ToolDefinition{
	{
		Enabled: true,
		Trusted: true,
		Tool: types.Tool{
			Name:        "firecrawl_scrape",
			Description: "Scrape a url and return it as markdown. The url is required.",
			Parameters: objectParameters(map[string]any{
				"url": stringParameter("The url to scrape."),
			}, "url"),
		},
		Command: &types.ToolCommand{
			Command: []string{
				"curl",
				"-X",
				"POST",
				"https://api.firecrawl.dev/v1/scrape",
				"-H",
				"Content-Type: application/json",
				"-H",
				"Authorization: Bearer {{.Env.FIRECRAWL_API_KEY}}",
				"-d",
				`{ "url": {{ json .Params.url}}, "formats": ["markdown"] }`,
			},
			AllowedEnv: []string{"FIRECRAWL_API_KEY"},
		},
	},
	{
		Enabled: true,
		Trusted: true,
		Tool: types.Tool{
			Name:        "firecrawl_search",
			Description: "Search the web. Query is required.",
			Parameters: objectParameters(map[string]any{
				"query": stringParameter("The search query to send to Firecrawl."),
			}, "query"),
		},
		Command: &types.ToolCommand{
			Command: []string{
				"curl",
				"-X",
				"POST",
				"https://api.firecrawl.dev/v1/search",
				"-H",
				"Content-Type: application/json",
				"-H",
				"Authorization: Bearer {{.Env.FIRECRAWL_API_KEY}}",
				"-d",
				`{ "query": {{ json .Params.query}} }`,
			},
			AllowedEnv: []string{"FIRECRAWL_API_KEY"},
		},
	},
	{
		Enabled: true,
		Trusted: true,
		Tool: types.Tool{
			Name:        "current_time",
			Description: "Gets the current date and time. Useful for when the response needs to be time-aware.",
			Parameters:  map[string]any{},
		},
		Command: &types.ToolCommand{
			Command: []string{"date"},
		},
	},
	commandTool(
		"git_diff",
		"Run a 'git diff' command. Prefer this command over the generic git command when possible.",
		"The arguments for the git diff command. This is everything after `git diff`",
		true,
		"git", "show",
	),
	commandTool(
		"git_show",
		"Run a 'git show' command. Prefer this command over the generic git command when possible.",
		"The arguments for the git show command. This is everything after `git show`",
		true,
		"git", "show",
	),
	commandTool(
		"git_status",
		"Run a 'git status' command. Prefer this command over the generic git command when possible.",
		"The arguments for the git status command. This is everything after `git status`",
		true,
		"git", "status",
	),
	commandTool(
		"git",
		"Run any git command.",
		"The arguments for the git command.",
		false,
		"git",
	),
	commandTool(
		"ls",
		"Standard ls for listing files and directories.",
		"The arguments for the ls command",
		true,
		"ls",
	),
	commandTool(
		"mkdir",
		"Create a directory. This uses the systems mkdir command.",
		"The arguments for the mkdir",
		true,
		"mkdir",
	),
	commandTool(
		"mv",
		"move files or folders. This is the systems default mv command.",
		"The arguments for the mv command.",
		false,
		"mv",
	),
	{
		Enabled: true,
		Trusted: true,
		Tool: types.Tool{
			Name:        "read_file",
			Description: "Read a file from disk. You can limit the range to a number of lines and bytes.",
			Parameters: objectParameters(map[string]any{
				"filename":           stringParameter("The file to read."),
				"startLine":          defaultParameter("number", "The line to start reading at.", 1),
				"maxLines":           defaultParameter("number", "The maximum lines returned.", 100),
				"maxBytes":           defaultParameter("number", "The maximum bytes returned.", 4096),
				"includeLineNumbers": defaultParameter("boolean", "If true, line numbers are included in the output.", false),
			}, "filename"),
		},
		Command: &types.ToolCommand{
			Command: []string{
				`{{ with index .Env "HARNESSER_BIN"}}{{.}}{{else}}harnesser{{end}}`,
				"tool",
				"read-file",
				"{{ .Params.filename }}",
				`{{ with index .Params "startLine" }}--start={{ . }}{{ end }}`,
				`{{ with index .Params "maxLines" }}--max-lines={{ . }}{{ end }}`,
				`{{ with index .Params "maxBytes" }}--max-bytes={{ . }}{{ end }}`,
				`{{ with index .Params "includeLineNumbers" }}--line-numbers{{ end }}`,
			},
		},
	},
	{
		Enabled: true,
		Trusted: false,
		Tool: types.Tool{
			Name:        "replace_text",
			Description: "Replace literal text in a file if the number of matches is as expected.",
			Parameters: objectParameters(map[string]any{
				"filename":        stringParameter("The file to edit."),
				"search":          stringParameter("The literal text to search for."),
				"replacement":     stringParameter("The text that replaces every match."),
				"expectedMatches": defaultParameter("number", "The number of matches required before the file is edited.", 1),
			}, "filename", "search", "replacement"),
		},
		Command: &types.ToolCommand{
			Command: []string{
				`{{ with index .Env "HARNESSER_BIN"}}{{.}}{{else}}harnesser{{end}}`,
				"tool",
				"replace-text",
				"{{ .Params.filename }}",
				"{{ .Params.search }}",
				"{{ .Params.replacement }}",
				`{{ if ne (index .Params "expectedMatches") nil }}--expected-matches={{ index .Params "expectedMatches" }}{{ end }}`,
			},
		},
	},
	commandTool(
		"ripgrep",
		"Execute a ripgrep command. Uses the systems intalled rg. If it does not exist it will return an error.",
		"The arguments for the rg command",
		true,
		"rg",
	),
	commandTool(
		"rm",
		"remove files and folders. This is the systems default rm command.",
		"The arguments for the rm command",
		false,
		"rm",
	),
	{
		Enabled: true,
		Trusted: false,
		Tool: types.Tool{
			Name:        "shell_command",
			Description: "Run arbitrary shell command. Use only when other tools cannot achieve the same result.",
			Parameters: objectParameters(map[string]any{
				"command": stringParameter("The shell command to run."),
			}, "command"),
		},
		Command: &types.ToolCommand{
			Command:  []string{"sh", "-c", "{{ .Params.command }}"},
			ArgsFrom: "args",
		},
	},
	{
		Enabled: true,
		Trusted: true,
		Tool: types.Tool{
			Name:        "write_file",
			Description: "Write to a file If the file does not exist, it will be created. If it does exist, it will be truncated and overwritten. If the append flag is used, new content will be appended to the file.",
			Parameters: objectParameters(map[string]any{
				"filename": stringParameter("The file to write to."),
				"contents": stringParameter("The file contents to write"),
				"append":   defaultParameter("boolean", "If true, contents will be appended to the file.", false),
			}, "filename", "contents"),
		},
		Command: &types.ToolCommand{
			Command: []string{
				`{{ with index .Env "HARNESSER_BIN"}}{{.}}{{else}}harnesser{{end}}`,
				"tool",
				"write-file",
				"{{ .Params.filename }}",
				"{{ .Params.contents }}",
				`{{ with index .Params "append" }}--append{{ end }}`,
			},
		},
	},
}

func GetStandardTools() []types.ToolDefinition {
	return standardTools
}

func WriteStandardTools(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create standard tools directory: %w", err)
	}

	for _, def := range standardTools {
		data, err := yaml.Marshal(def)
		if err != nil {
			return fmt.Errorf("marshal standard tool %q: %w", def.Tool.Name, err)
		}

		filename := filepath.Join(path, def.Tool.Name+".yaml")
		if err := os.WriteFile(filename, data, 0o644); err != nil {
			return fmt.Errorf("write standard tool %q: %w", def.Tool.Name, err)
		}
	}

	return nil
}

func commandTool(name, description, argsDescription string, trusted bool, command ...string) types.ToolDefinition {
	return types.ToolDefinition{
		Enabled: true,
		Trusted: trusted,
		Tool: types.Tool{
			Name:        name,
			Description: description,
			Parameters: objectParameters(map[string]any{
				"args": map[string]any{
					"type":        "array",
					"description": argsDescription,
					"items": map[string]any{
						"type": "string",
					},
				},
			}),
		},
		Command: &types.ToolCommand{
			Command:  command,
			ArgsFrom: "args",
		},
	}
}

func objectParameters(properties map[string]any, required ...string) map[string]any {
	parameters := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		parameters["required"] = required
	}
	return parameters
}

func stringParameter(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
	}
}

func defaultParameter(parameterType, description string, defaultValue any) map[string]any {
	return map[string]any{
		"type":        parameterType,
		"description": description,
		"default":     defaultValue,
	}
}
