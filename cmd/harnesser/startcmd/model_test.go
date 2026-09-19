package startcmd

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/c00/harnesser/agent"
	"github.com/c00/harnesser/history"
	"github.com/c00/harnesser/systemprompts"
	"github.com/c00/harnesser/toolset"
	"github.com/c00/harnesser/types"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestViewportHeightTracksTextareaHeight(t *testing.T) {
	agent := agent.NewAgent(nil, systemprompts.NewMemoryProvider(), history.NewMemoryProvider(), toolset.NewEmptyRegistry())
	model := initialModel(t.Context(), agent, false, func(m tea.Msg) {})
	model.state = ready
	model.textarea.Focus()

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	model = updated.(tuiModel)
	assert.Equal(t, 1, model.textarea.Height())
	assert.Equal(t, 17, model.viewport.Height())
	assert.Equal(t, 20, lipgloss.Height(model.View().Content))

	updated, _ = model.Update(tea.PasteMsg{Content: "one\ntwo\nthree"})
	model = updated.(tuiModel)
	assert.Equal(t, 3, model.textarea.Height())
	assert.Equal(t, 15, model.viewport.Height())
	assert.Equal(t, 20, lipgloss.Height(model.View().Content))

	model.textarea.SetValue(strings.Repeat("line\n", 11))
	model.updateViewportHeight()
	assert.Equal(t, model.textarea.MaxHeight, model.textarea.Height())
	assert.Equal(t, 8, model.viewport.Height())
	assert.Equal(t, 20, lipgloss.Height(model.View().Content))
}

func TestInitialModelStartupState(t *testing.T) {
	tests := []struct {
		name        string
		runOnStart  bool
		wantState   tuiState
		wantFocused bool
	}{
		{name: "waits for input without a prompt", wantState: ready, wantFocused: true},
		{name: "runs when a prompt was supplied", runOnStart: true, wantState: busy, wantFocused: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := agent.NewAgent(nil, systemprompts.NewMemoryProvider(), history.NewMemoryProvider(), toolset.NewEmptyRegistry())
			model := initialModel(t.Context(), agent, tt.runOnStart, func(m tea.Msg) {})

			assert.Equal(t, tt.wantState, model.state)
			assert.Equal(t, tt.wantFocused, model.textarea.Focused())
			assert.Equal(t, tt.runOnStart, model.runOnStart)
		})
	}
}

func TestInitialModelRendersLoadedHistory(t *testing.T) {
	agent := agent.NewAgent(nil, systemprompts.NewMemoryProvider(), history.NewMemoryProvider(), toolset.NewEmptyRegistry())
	agent.AddMessage(types.NewUserTextMessage("previous question"))
	agent.AddMessage(types.NewAssistantMessage("previous answer"))
	model := initialModel(t.Context(), agent, false, func(m tea.Msg) {})

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	model = updated.(tuiModel)

	rendered := ansi.Strip(model.viewport.View())
	assert.Contains(t, rendered, "[you]: previous question")
	assert.Contains(t, rendered, "[assistant]: previous answer")
}

func TestViewportHeightReservesPermissionPrompt(t *testing.T) {
	agent := agent.NewAgent(nil, systemprompts.NewMemoryProvider(), history.NewMemoryProvider(), toolset.NewEmptyRegistry())
	model := initialModel(t.Context(), agent, false, func(m tea.Msg) {})
	model.state = askPermission
	model.permission = &permissionModel{
		requests: []permissionRequest{{
			toolCallID: "call_1",
			function:   "shell",
			command:    strings.Repeat("long command ", 8),
		}},
		width: 20,
	}

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 20, Height: 20})
	model = updated.(tuiModel)

	promptHeight := lipgloss.Height(model.permission.View().Content)
	assert.Greater(t, promptHeight, 7)
	assert.Equal(t, 20-promptHeight, model.viewport.Height())
	assert.Equal(t, 20, lipgloss.Height(model.View().Content))
}

func TestRenderMessages(t *testing.T) {
	tools := toolset.NewEmptyRegistry()
	tools.SetToolDefs(types.ToolDefinition{
		Tool: types.Tool{
			Name: "print",
		},
		Command: &types.ToolCommand{
			Command: []string{"printf", "{{.Params.path}}"},
		},
	})

	agent := agent.NewAgent(nil, systemprompts.NewMemoryProvider(), history.NewMemoryProvider(), tools)
	agent.AddMessage(types.NewUserTextMessage("first"))
	agent.AddMessage(types.NewUserTextMessage("second"))
	agent.AddMessage(types.Message{
		Role:      types.RoleAssistant,
		Content:   types.MessageParts{{Type: types.PartText, Text: "checking"}},
		Reasoning: types.ReasoningParts{{Type: "reasoning.text", Text: "hidden thought"}},
		ToolCalls: []types.ToolCall{{Function: "print", Args: `{"path":"report.txt"}`}},
	})
	agent.AddMessage(types.NewToolResultMessage("call_1", "ignored result"))
	agent.AddMessage(types.NewAssistantMessage("done"))

	model := initialModel(t.Context(), agent, false, func(m tea.Msg) {})
	model.viewport.SetWidth(80)
	rendered := model.renderMessages()

	assert.Equal(t, strings.Join([]string{
		"[you]: first",
		"[you]: second",
		"",
		"[assistant]: checking",
		"",
		"[tool]: printf report.txt",
		"[result]: ignored result",
		"",
		"[assistant]: done",
	}, "\n"), strings.TrimRight(ansi.Strip(rendered), " "))
	assert.NotContains(t, rendered, "hidden thought")

	model.viewport.SetWidth(16)
	for line := range strings.SplitSeq(model.renderMessages(), "\n") {
		assert.LessOrEqual(t, lipgloss.Width(line), 16)
	}
}

func TestRenderMessagesTruncatesToolResults(t *testing.T) {
	agent := agent.NewAgent(nil, systemprompts.NewMemoryProvider(), history.NewMemoryProvider(), toolset.NewEmptyRegistry())
	agent.AddMessage(types.NewToolResultMessage("call_1", strings.Repeat("界", 201)))

	model := initialModel(t.Context(), agent, false, func(m tea.Msg) {})
	model.viewport.SetWidth(500)
	rendered := ansi.Strip(model.renderMessages())

	assert.Equal(t, "[result]: "+strings.Repeat("界", 200)+" [output truncated]", rendered)
}
