package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/c00/harnesser/historyprovider"
	"github.com/c00/harnesser/llm"
	"github.com/c00/harnesser/models"
	"github.com/c00/harnesser/promptsprovider"
	"github.com/c00/harnesser/tools"
	"github.com/c00/harnesser/toolsprovider"
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
	ToApprove []PendingToolcall
}

type PendingToolcall struct {
	ToolCall       models.ToolCall
	ToolDefinition models.ToolDefinition
}

type Runner struct {
	llm             llm.LlmProvider
	promptsProvider promptsprovider.PromptsReader
	histProv        historyprovider.HistoryProvider
	toolsProvider   toolsprovider.ToolsProvider

	messages models.Messages
}

func NewRunner(llm llm.LlmProvider, promptsProvider promptsprovider.PromptsReader, histProv historyprovider.HistoryProvider, toolsProvider toolsprovider.ToolsProvider) *Runner {
	runner := Runner{
		llm:             llm,
		promptsProvider: promptsProvider,
		histProv:        histProv,
		toolsProvider:   toolsProvider,
		messages:        models.Messages{},
	}

	return &runner
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

// Load history from file (if any)
func (r *Runner) LoadHistory() {
	r.messages = r.histProv.Get().Messages
}

func (r *Runner) WriteHistory() error {
	entry := r.histProv.Get()
	entry.Messages = r.messages

	err := r.histProv.Save(entry)
	if err != nil {
		return fmt.Errorf("cannot write history: %w", err)
	}

	return nil
}

// Add User message and RunStep (one step only)
func (r *Runner) RunPrompt(ctx context.Context, msg models.Message) (models.Message, error) {
	r.AddMessage(msg)
	return r.RunInference(ctx)
}

// runInference runs inference to the LLM Provider. If callback is set, runs streaming.
func (r *Runner) runInference(ctx context.Context, cb llm.StreamDeltaFunc) (models.Message, error) {
	allMsgs := append(models.Messages{}, r.promptsProvider.Prompts().ActiveMessages()...)
	allMsgs = append(allMsgs, r.messages...)

	shouldStream := true

	// If no callback, don't stream
	if cb == nil {
		shouldStream = false
	}

	// If streaming is not supported, don't stream
	streamingProvider, ok := r.llm.(llm.LlmStreamingProvider)
	if !ok {
		shouldStream = false
	}

	var err error
	var resp models.Message

	if shouldStream {
		resp, err = streamingProvider.GenerateStream(ctx, allMsgs, r.toolsProvider.GetTools(), cb)
		if err != nil {
			return models.Message{}, fmt.Errorf("cannot generate llm response: %w", err)
		}
	} else {
		resp, err = r.llm.Generate(ctx, allMsgs, r.toolsProvider.GetTools())
		if err != nil {
			return models.Message{}, fmt.Errorf("cannot generate llm response: %w", err)
		}
	}

	r.AddMessage(resp)
	err = r.WriteHistory()
	if err != nil {
		return models.Message{}, fmt.Errorf("cannot write history: %w", err)
	}

	return resp, nil
}

// RunInferenceStream runs inference to the LLM Provider with streaming output.
func (r *Runner) RunInferenceStream(ctx context.Context, cb llm.StreamDeltaFunc) (models.Message, error) {
	if cb == nil {
		return models.Message{}, errors.New("stream callback cannot be nil")
	}

	return r.runInference(ctx, cb)
}

// RunInference runs inference to the LLM Provider.
func (r *Runner) RunInference(ctx context.Context) (models.Message, error) {
	return r.runInference(ctx, nil)
}

// RunStep runs the next step. This is either inference or tool calls
func (r *Runner) runStep(ctx context.Context, cb llm.StreamDeltaFunc) (Response, error) {
	// Nothing to do
	if len(r.messages) == 0 {
		return Response{Type: ResponseTypeNew, Messages: models.Messages{}}, nil
	}

	lastMsg := r.messages[len(r.messages)-1]

	if lastMsg.Role == models.RoleUser || lastMsg.Role == models.RoleTool {
		msg, err := r.runInference(ctx, cb)
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

		notApprovedYet := []PendingToolcall{}

		for _, tc := range lastMsg.ToolCalls {
			td, err := r.toolsProvider.GetDefinition(tc.Function)
			if err != nil {
				slog.Warn("cannot get tool definition, skipping", "error", err.Error(), "function", tc.Function)
				continue
			}

			if !td.Trusted && (tc.Decision == models.ToolCallDecisionNoDecision || tc.Decision == "") {
				notApprovedYet = append(notApprovedYet, PendingToolcall{ToolCall: tc, ToolDefinition: td})
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

// RunStep runs the next step. This is either inference or tool calls
func (r *Runner) RunStep(ctx context.Context) (Response, error) {
	return r.runStep(ctx, nil)
}

// RunStep runs the next step. This is either inference or tool calls
func (r *Runner) RunStepStream(ctx context.Context, cb llm.StreamDeltaFunc) (Response, error) {
	return r.runStep(ctx, cb)
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
			slog.Debug("Cannot run toolcall", "id", tc.ToolCallID, "name", tc.Function, "error", err.Error())
			msg = models.NewToolErrorMessage(tc.ToolCallID, fmt.Sprintf("cannot run tool call '%v': %v", tc.Function, err.Error()))
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
	td, err := r.toolsProvider.GetDefinition(tc.Function)
	if err != nil {
		return models.Message{}, fmt.Errorf("cannot get tool def for %q: %w", tc.Function, err)
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

	builder, err := tools.NewToolCallBuilder(tc, td)
	if err != nil {
		return models.Message{}, fmt.Errorf("cannot create tool call builder: %w", err)
	}

	// TODO limit env vars
	cmd := exec.CommandContext(execCtx, builder.CommandName(), builder.CommandArgs()...)

	slog.Debug("Executing command", "command", fmt.Sprintf("%v", builder.CommandString()))

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

// ToolCallCommand returns the command that will be executed for a tool call.
func (r *Runner) ToolCallCommand(tc models.ToolCall) (string, error) {
	td, err := r.toolsProvider.GetDefinition(tc.Function)
	if err != nil {
		return "", fmt.Errorf("cannot get tool def for %q: %w", tc.Function, err)
	}

	builder, err := tools.NewToolCallBuilder(tc, td)
	if err != nil {
		return "", fmt.Errorf("cannot create tool call builder: %w", err)
	}

	return strings.TrimSpace(builder.CommandString()), nil
}
