package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/c00/harnesser/interpolator"
	"github.com/c00/harnesser/llm"
	"github.com/c00/harnesser/models"
	"go.yaml.in/yaml/v4"
)

const (
	ResponseTypeNew                      RunnerResponseType = "new"
	ResponseTypeInferenceResultWithTools RunnerResponseType = "inference-result-with-tools"
	ResponseTypeInferenceResultNoTools   RunnerResponseType = "inference-result-no-tools"
	ResponseTypeAskPermission            RunnerResponseType = "ask-permission"
	ResponseTypeToolResults              RunnerResponseType = "tool-results"
	ResponseTypeDone                     RunnerResponseType = "done"
)

var ErrNoDecision = errors.New("no decision for tool call")

type RunnerResponseType string

type UpdateResponse struct {
	Message models.Message
}

type Response struct {
	Type      RunnerResponseType
	Messages  models.Messages
	ToApprove []models.ToolCall
}

type Runner struct {
	llm        llm.LlmProvider
	promptsDir string
	historyDir string
	toolsDir   string

	// points to the file in the history dir
	// if unset, will generate a name
	// if exists, will overwrite / append
	// if new, will create
	name string

	prompts  models.Messages
	messages models.Messages
	toolDefs map[string]models.ToolDefinition
}

func NewRunner(llm llm.LlmProvider, promptsDir, histDir, toolsDir string, name string) (*Runner, error) {
	if name == "" {
		name = fmt.Sprintf("%v.yaml", time.Now().Format(time.RFC3339))
	}

	runner := Runner{
		llm:        llm,
		promptsDir: promptsDir,
		historyDir: histDir,
		toolsDir:   toolsDir,
		name:       name,
		messages:   models.Messages{},
		toolDefs:   map[string]models.ToolDefinition{},
	}

	err := errors.Join(
		runner.ensureDir(runner.toolsDir),
		runner.ensureDir(runner.historyDir),
		runner.ensureDir(runner.promptsDir),
	)

	if err != nil {
		return nil, fmt.Errorf("cannot create dirs: %w", err)
	}

	err = runner.loadTools()
	if err != nil {
		return nil, fmt.Errorf("cannot load tools: %w", err)
	}

	err = runner.loadPrompts()
	if err != nil {
		return nil, fmt.Errorf("cannot load prompts: %w", err)
	}

	return &runner, nil
}

// ConfirmToolCall lets the user approve or reject a tool call
// Only works for the last message
func (r *Runner) ConfirmToolCall(ctx context.Context, toolCallID string, decision models.ToolCallDecision) error {
	if len(r.messages) == 0 {
		return nil
	}

	lastMsg := r.messages[len(r.messages)-1]

	for i, tc := range lastMsg.ToolCalls {
		if tc.ToolCallID == toolCallID {
			tc.Decision = decision
			lastMsg.ToolCalls[i] = tc
			r.messages[len(r.messages)-1] = lastMsg
			err := r.WriteHistory()
			if err != nil {
				return fmt.Errorf("cannot write history: %w", err)
			}
			return nil
		}
	}

	return nil
}

func (r *Runner) tools() models.Tools {
	tools := models.Tools{}
	for _, t := range r.toolDefs {
		tools = append(tools, t.Tool)
	}

	return tools
}

func (r *Runner) ensureDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create directory '%v': %w", dir, err)
	}
	return nil
}

// Load the tools from the toolDir.
func (r *Runner) loadTools() error {
	r.toolDefs = map[string]models.ToolDefinition{}

	entries, err := os.ReadDir(r.toolsDir)
	if err != nil {
		return fmt.Errorf("cannot read tools dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(r.toolsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("cannot read tool %q: %w", path, err)
		}

		var toolDef models.ToolDefinition
		if err := yaml.Unmarshal(data, &toolDef); err != nil {
			return fmt.Errorf("cannot parse tool %q: %w", path, err)
		}
		r.toolDefs[toolDef.Tool.Name] = toolDef
	}

	return nil
}

func (r *Runner) loadPrompts() error {
	r.prompts = models.Messages{}

	entries, err := os.ReadDir(r.promptsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".md" && ext != ".txt" {
			continue
		}

		path := filepath.Join(r.promptsDir, entry.Name())
		prompt, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("cannot read prompt %q: %w", path, err)
		}
		r.prompts = append(r.prompts, models.NewSystemMessage(string(prompt)))
	}

	return nil
}

// Load history from file (if any)
func (r *Runner) LoadHistory() error {
	// If no name is set, silently return
	if r.name == "" {
		return nil
	}

	r.messages = models.Messages{}

	paths := []string{r.name}
	if !filepath.IsAbs(r.name) {
		paths = append(paths, filepath.Join(r.historyDir, r.name))
	}

	var historyPath string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err == nil {
			if info.IsDir() {
				return fmt.Errorf("history path %q is a directory", path)
			}
			historyPath = path
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("cannot stat history %q: %w", path, err)
		}
	}

	if historyPath == "" {
		return nil
	}

	data, err := os.ReadFile(historyPath)
	if err != nil {
		return fmt.Errorf("cannot read history %q: %w", historyPath, err)
	}
	if len(data) == 0 {
		return nil
	}
	if err := yaml.Unmarshal(data, &r.messages); err != nil {
		return fmt.Errorf("cannot parse history %q: %w", historyPath, err)
	}

	return nil
}

func (r *Runner) WriteHistory() error {
	if r.name == "" {
		return nil
	}

	historyPath := r.name
	if !filepath.IsAbs(historyPath) && filepath.Dir(historyPath) == "." {
		historyPath = filepath.Join(r.historyDir, historyPath)
	}

	if err := os.MkdirAll(filepath.Dir(historyPath), 0o755); err != nil {
		return fmt.Errorf("cannot create history directory: %w", err)
	}

	data, err := yaml.Marshal(r.messages)
	if err != nil {
		return fmt.Errorf("cannot encode history: %w", err)
	}
	if err := os.WriteFile(historyPath, data, 0o644); err != nil {
		return fmt.Errorf("cannot write history %q: %w", historyPath, err)
	}

	return nil
}

// Add User message and RunStep (one step only)
func (r *Runner) RunPrompt(ctx context.Context, msg models.Message) (models.Message, error) {
	r.AddMessage(msg)
	return r.RunInference(ctx)
}

// RunInference runs inference to the LLM Provider.
func (r *Runner) RunInference(ctx context.Context) (models.Message, error) {
	allMsgs := make([]models.Message, len(r.prompts))
	copy(allMsgs, r.prompts)
	allMsgs = append(allMsgs, r.messages...)

	resp, err := r.llm.Generate(ctx, allMsgs, r.tools())
	if err != nil {
		return models.Message{}, fmt.Errorf("cannot generate llm response: %w", err)
	}

	r.AddMessage(resp)
	err = r.WriteHistory()
	if err != nil {
		return models.Message{}, fmt.Errorf("cannot write history: %w", err)
	}

	return resp, nil
}

// RunStep runs the next step. This is either inference or tool calls
func (r *Runner) RunStep(ctx context.Context) (Response, error) {
	// Nothing to do
	if len(r.messages) == 0 {
		return Response{Type: ResponseTypeNew, Messages: models.Messages{}}, nil
	}

	lastMsg := r.messages[len(r.messages)-1]

	if lastMsg.Role == models.RoleUser || lastMsg.Role == models.RoleTool {
		msg, err := r.RunInference(ctx)
		if err != nil {
			return Response{}, fmt.Errorf("cannot run inference: %w", err)
		}

		respType := ResponseTypeInferenceResultNoTools
		if len(msg.ToolCalls) > 0 {
			respType = ResponseTypeInferenceResultWithTools
		}

		return Response{
			Messages: models.Messages{msg},
			Type:     respType,
		}, nil
	} else if lastMsg.Role == models.RoleAssistant && len(lastMsg.ToolCalls) > 0 {

		// I need to check if ALL tool calls are approved.
		// If any are not approved, then we just wait with executing anything.
		// Not the most efficient but fine for now.
		// If any are actively rejected, the RunTools will gracefully deal with it.

		notApprovedYet := []models.ToolCall{}

		for _, tc := range lastMsg.ToolCalls {
			td, ok := r.toolDefs[tc.Function]
			if !ok {
				return Response{}, fmt.Errorf("tool not defined: %v", tc.Function)
			}

			if !td.Trusted && (tc.Decision == models.ToolCallDecisionNoDecision || tc.Decision == "") {
				notApprovedYet = append(notApprovedYet, tc)
			}
		}

		if len(notApprovedYet) > 0 {
			return Response{
				Type:      ResponseTypeAskPermission,
				Messages:  models.Messages{lastMsg},
				ToApprove: notApprovedYet,
			}, nil
		} else {
			msgs, err := r.RunTools(ctx)
			if err != nil {
				return Response{}, fmt.Errorf("cannot run tools: %w", err)
			}
			return Response{Messages: msgs, Type: ResponseTypeToolResults}, nil
		}
	}

	return Response{
		Messages: models.Messages{},
		Type:     ResponseTypeDone,
	}, nil
}

// Looks at the current latest message, and runs tool calls if any
// We assume that approval has already been given by the user
func (r *Runner) RunTools(ctx context.Context) (models.Messages, error) {
	if len(r.messages) == 0 {
		// No messages
		return models.Messages{}, nil
	}

	lastMessage := r.messages[len(r.messages)-1]
	if len(lastMessage.ToolCalls) == 0 {
		// No tool calls to process
		return models.Messages{}, nil
	}

	messages := models.Messages{}
	for _, tc := range lastMessage.ToolCalls {
		msg, err := r.runTool(ctx, tc)
		if err != nil {
			return nil, fmt.Errorf("cannot run tool call '%v': %w", tc.Function, err)
		}
		messages = append(messages, msg)
	}

	r.messages = append(r.messages, messages...)

	err := r.WriteHistory()
	if err != nil {
		return nil, fmt.Errorf("cannot write history: %w", err)
	}

	return messages, nil
}

func (r *Runner) runTool(ctx context.Context, tc models.ToolCall) (models.Message, error) {
	// Find the tool
	td, ok := r.toolDefs[tc.Function]
	if !ok {
		return models.Message{}, fmt.Errorf("tool not defined: %v", tc.Function)
	}

	// Can we run this tool?
	if !td.Trusted {
		switch tc.Decision {
		case "":
			return models.Message{}, fmt.Errorf("tool %v, function %v: %w", tc.ToolCallID, tc.Function, ErrNoDecision)
		case models.ToolCallDecisionNoDecision:
			return models.Message{}, fmt.Errorf("tool %v, function %v: %w", tc.ToolCallID, tc.Function, ErrNoDecision)
		case models.ToolCallDecisionReject:
			return models.NewToolErrorMessage(tc.ToolCallID, "user rejected the running of this tool call"), nil
		}
	}

	// build context
	// Give it a cancelable context, with a timeout of 5 seconds
	execCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	cmdSlice, err := interpolator.InterpolatedCommand(tc, td)
	if err != nil {
		return models.Message{}, fmt.Errorf("cannot interpolate command for tool '%v': %w", td.Tool.Name, err)
	}
	args := cmdSlice[1:]
	// Remove empties
	args = slices.DeleteFunc(args, func(s string) bool {
		return s == ""
	})
	// Build the command with only the environment variables tools are allowed to inherit.
	// TODO limit env vars
	cmd := exec.CommandContext(execCtx, cmdSlice[0], args...)

	slog.Debug("Executing command", "command", fmt.Sprintf("%v %v", cmdSlice[0], strings.Join(args, " ")))

	var stderr strings.Builder
	cmd.Stderr = &stderr

	data, err := cmd.Output()
	if err != nil {
		// If the process finishes with a non-zero exit code, return an error
		stderrText := strings.TrimSpace(stderr.String())
		if stderrText != "" {
			return models.Message{}, fmt.Errorf("running tool '%v' failed: %w: %s", td.Tool.Name, err, stderrText)
		}

		return models.Message{}, fmt.Errorf("running tool '%v' failed: %w", td.Tool.Name, err)
	}

	return models.NewToolResultMessage(tc.ToolCallID, string(data)), nil
}

// AddMessage Adds a message to the loaded history. Does not trigger inference.
func (r *Runner) AddMessage(msgs models.Message) {
	r.messages = append(r.messages, msgs)
}

// Messages gets a copy of the current message cache
func (r *Runner) Messages() models.Messages {
	messagesCopy := make([]models.Message, len(r.messages))
	copy(messagesCopy, r.messages)

	return messagesCopy
}
