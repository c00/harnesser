package startcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/c00/harnesser/models"
	"github.com/c00/harnesser/runner"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestViewportHeightTracksTextareaHeight(t *testing.T) {
	agent, err := runner.NewRunner(nil, filepath.Join(t.TempDir(), "prompts"), filepath.Join(t.TempDir(), "history"), filepath.Join(t.TempDir(), "tools"), "")
	require.NoError(t, err)
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
			agent, err := runner.NewRunner(nil, filepath.Join(t.TempDir(), "prompts"), filepath.Join(t.TempDir(), "history"), filepath.Join(t.TempDir(), "tools"), "")
			require.NoError(t, err)

			model := initialModel(t.Context(), agent, tt.runOnStart, func(m tea.Msg) {})

			assert.Equal(t, tt.wantState, model.state)
			assert.Equal(t, tt.wantFocused, model.textarea.Focused())
			assert.Equal(t, tt.runOnStart, model.runOnStart)
		})
	}
}

func TestInitialModelRendersLoadedHistory(t *testing.T) {
	agent, err := runner.NewRunner(nil, filepath.Join(t.TempDir(), "prompts"), filepath.Join(t.TempDir(), "history"), filepath.Join(t.TempDir(), "tools"), "")
	require.NoError(t, err)
	agent.AddMessage(models.NewUserTextMessage("previous question"))
	agent.AddMessage(models.NewAssistantMessage("previous answer"))
	model := initialModel(t.Context(), agent, false, func(m tea.Msg) {})

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	model = updated.(tuiModel)

	rendered := ansi.Strip(model.viewport.View())
	assert.Contains(t, rendered, "[you]: previous question")
	assert.Contains(t, rendered, "[assistant]: previous answer")
}

func TestViewportHeightReservesPermissionPrompt(t *testing.T) {
	agent, err := runner.NewRunner(nil, filepath.Join(t.TempDir(), "prompts"), filepath.Join(t.TempDir(), "history"), filepath.Join(t.TempDir(), "tools"), "")
	require.NoError(t, err)
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
	dir := t.TempDir()
	toolsDir := filepath.Join(dir, "tools")
	require.NoError(t, os.MkdirAll(toolsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(toolsDir, "print.yaml"), []byte(`
tool:
  name: print
command:
  - printf
  - '{{.Params.path}}'
`), 0o644))

	agent, err := runner.NewRunner(nil, filepath.Join(dir, "prompts"), filepath.Join(dir, "history"), toolsDir, "")
	require.NoError(t, err)
	agent.AddMessage(models.NewUserTextMessage("first"))
	agent.AddMessage(models.NewUserTextMessage("second"))
	agent.AddMessage(models.Message{
		Role:      models.RoleAssistant,
		Content:   models.MessageParts{{Type: models.PartText, Text: "checking"}},
		Reasoning: models.ReasoningParts{{Type: "reasoning.text", Text: "hidden thought"}},
		ToolCalls: []models.ToolCall{{Function: "print", Args: `{"path":"report.txt"}`}},
	})
	agent.AddMessage(models.NewToolResultMessage("call_1", "ignored result"))
	agent.AddMessage(models.NewAssistantMessage("done"))

	model := initialModel(t.Context(), agent, false, func(m tea.Msg) {})
	model.viewport.SetWidth(80)
	rendered := model.renderMessages()

	assert.Equal(t, strings.Join([]string{
		"[you]: first",
		"[you]: second",
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
	agent, err := runner.NewRunner(nil, filepath.Join(t.TempDir(), "prompts"), filepath.Join(t.TempDir(), "history"), filepath.Join(t.TempDir(), "tools"), "")
	require.NoError(t, err)
	agent.AddMessage(models.NewToolResultMessage("call_1", strings.Repeat("界", 201)))

	model := initialModel(t.Context(), agent, false, func(m tea.Msg) {})
	model.viewport.SetWidth(500)
	rendered := ansi.Strip(model.renderMessages())

	assert.Equal(t, "[result]: "+strings.Repeat("界", 200)+" [output truncated]", rendered)
}
