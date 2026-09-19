package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/c00/harnesser/history"
	"github.com/c00/harnesser/llm"
	"github.com/c00/harnesser/systemprompts"
	"github.com/c00/harnesser/toolcallbuilder"
	"github.com/c00/harnesser/toolset"
	"github.com/c00/harnesser/types"
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
	Message types.Message
}

type Response struct {
	Type      RunnerResponseType
	Messages  types.Messages
	ToApprove []PendingToolcall
}

type PendingToolcall struct {
	ToolCall       types.ToolCall
	ToolDefinition types.ToolDefinition
}

type Agent struct {
	llm             llm.LlmProvider
	promptsProvider systemprompts.PromptsReader
	histProv        history.HistoryProvider
	toolsRegistry   *toolset.ToolRegistry

	messages types.Messages
}

func NewAgent(llm llm.LlmProvider, promptsProvider systemprompts.PromptsReader, histProv history.HistoryProvider, tools *toolset.ToolRegistry) *Agent {
	runner := Agent{
		llm:             llm,
		promptsProvider: promptsProvider,
		histProv:        histProv,
		toolsRegistry:   tools,
		messages:        types.Messages{},
	}

	runner.LoadHistory()

	return &runner
}

// ConfirmToolCall lets the user approve or reject a tool call
// Only works for the last message
func (r *Agent) ConfirmToolCall(ctx context.Context, toolCallID string, decision types.ToolCallDecision) error {
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
func (r *Agent) LoadHistory() {
	r.messages = r.histProv.Get().Messages
}

func (r *Agent) WriteHistory() error {
	entry := r.histProv.Get()
	entry.Messages = r.messages

	err := r.histProv.Save(entry)
	if err != nil {
		return fmt.Errorf("cannot write history: %w", err)
	}

	return nil
}

// Add User message and RunStep (one step only)
func (r *Agent) RunPrompt(ctx context.Context, msg types.Message) (types.Message, error) {
	r.AddMessage(msg)
	return r.RunInference(ctx)
}

// runInference runs inference to the LLM Provider. If callback is set, runs streaming.
func (r *Agent) runInference(ctx context.Context, cb llm.StreamDeltaFunc) (types.Message, error) {
	allMsgs := append(types.Messages{}, r.promptsProvider.Prompts().ActiveMessages()...)
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
	var resp types.Message

	if shouldStream {
		resp, err = streamingProvider.GenerateStream(ctx, allMsgs, r.toolsRegistry.GetTools(), cb)
		if err != nil {
			return types.Message{}, fmt.Errorf("cannot generate llm response: %w", err)
		}
	} else {
		resp, err = r.llm.Generate(ctx, allMsgs, r.toolsRegistry.GetTools())
		if err != nil {
			return types.Message{}, fmt.Errorf("cannot generate llm response: %w", err)
		}
	}

	r.AddMessage(resp)
	err = r.WriteHistory()
	if err != nil {
		return types.Message{}, fmt.Errorf("cannot write history: %w", err)
	}

	return resp, nil
}

// RunInferenceStream runs inference to the LLM Provider with streaming output.
func (r *Agent) RunInferenceStream(ctx context.Context, cb llm.StreamDeltaFunc) (types.Message, error) {
	if cb == nil {
		return types.Message{}, errors.New("stream callback cannot be nil")
	}

	return r.runInference(ctx, cb)
}

// RunInference runs inference to the LLM Provider.
func (r *Agent) RunInference(ctx context.Context) (types.Message, error) {
	return r.runInference(ctx, nil)
}

// RunStep runs the next step. This is either inference or tool calls
func (r *Agent) runStep(ctx context.Context, cb llm.StreamDeltaFunc) (Response, error) {
	// Nothing to do
	if len(r.messages) == 0 {
		return Response{Type: ResponseTypeNew, Messages: types.Messages{}}, nil
	}

	lastMsg := r.messages[len(r.messages)-1]

	if lastMsg.Role == types.RoleUser || lastMsg.Role == types.RoleTool {
		msg, err := r.runInference(ctx, cb)
		if err != nil {
			return Response{}, fmt.Errorf("cannot run inference: %w", err)
		}

		respType := ResponseTypeInferenceResultNoTools
		if len(msg.ToolCalls) > 0 {
			respType = ResponseTypeInferenceResultWithTools
		}

		return Response{
			Messages: types.Messages{msg},
			Type:     respType,
		}, nil
	} else if lastMsg.Role == types.RoleAssistant && len(lastMsg.ToolCalls) > 0 {

		// I need to check if ALL tool calls are approved.
		// If any are not approved, then we just wait with executing anything.
		// Not the most efficient but fine for now.
		// If any are actively rejected, the RunTools will gracefully deal with it.

		notApprovedYet := []PendingToolcall{}

		for _, tc := range lastMsg.ToolCalls {
			td, err := r.toolsRegistry.GetToolDef(tc.Function)
			if err != nil {
				slog.Warn("cannot get tool definition, skipping", "error", err.Error(), "function", tc.Function)
				continue
			}

			if !td.Trusted && (tc.Decision == types.ToolCallDecisionNoDecision || tc.Decision == "") {
				notApprovedYet = append(notApprovedYet, PendingToolcall{ToolCall: tc, ToolDefinition: td})
			}
		}

		if len(notApprovedYet) > 0 {
			return Response{
				Type:      ResponseTypeAskPermission,
				Messages:  types.Messages{lastMsg},
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
		Messages: types.Messages{},
		Type:     ResponseTypeDone,
	}, nil
}

// RunStep runs the next step. This is either inference or tool calls
func (r *Agent) RunStep(ctx context.Context) (Response, error) {
	return r.runStep(ctx, nil)
}

// RunStep runs the next step. This is either inference or tool calls
func (r *Agent) RunStepStream(ctx context.Context, cb llm.StreamDeltaFunc) (Response, error) {
	return r.runStep(ctx, cb)
}

// Looks at the current latest message, and runs tool calls if any
// We assume that approval has already been given by the user
func (r *Agent) RunTools(ctx context.Context) (types.Messages, error) {
	if len(r.messages) == 0 {
		// No messages
		return types.Messages{}, nil
	}

	lastMessage := r.messages[len(r.messages)-1]
	if len(lastMessage.ToolCalls) == 0 {
		// No tool calls to process
		return types.Messages{}, nil
	}

	messages := types.Messages{}
	for _, tc := range lastMessage.ToolCalls {
		msg, err := r.runTool(ctx, tc)
		if err != nil {
			slog.Debug("Cannot run toolcall", "id", tc.ToolCallID, "name", tc.Function, "error", err.Error())
			msg = types.NewToolErrorMessage(tc.ToolCallID, fmt.Sprintf("cannot run tool call '%v': %v", tc.Function, err.Error()))
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

func (r *Agent) runTool(ctx context.Context, tc types.ToolCall) (types.Message, error) {
	result, err := r.toolsRegistry.Run(ctx, tc)
	if err != nil {
		return types.Message{}, fmt.Errorf("cannot run tool %v: %w", tc.Function, err)
	}

	return types.NewToolResultMessage(tc.ToolCallID, result), nil
}

// AddMessage Adds a message to the loaded history. Does not trigger inference.
func (r *Agent) AddMessage(msgs types.Message) {
	r.messages = append(r.messages, msgs)
}

// Messages gets a copy of the current message cache
func (r *Agent) Messages() types.Messages {
	messagesCopy := make([]types.Message, len(r.messages))
	copy(messagesCopy, r.messages)

	return messagesCopy
}

// ToolCallCommand returns the command that will be executed for a tool call.
func (r *Agent) ToolCallCommand(tc types.ToolCall) (string, error) {
	td, err := r.toolsRegistry.GetToolDef(tc.Function)
	if err != nil {
		return "", fmt.Errorf("cannot get tool def for %q: %w", tc.Function, err)
	}

	builder, err := toolcallbuilder.NewToolCallCmdBuilder(tc, td)
	if err != nil {
		return "", fmt.Errorf("cannot create tool call builder: %w", err)
	}

	return strings.TrimSpace(builder.CommandString()), nil
}
