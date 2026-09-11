package startcmd

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/c00/harnesser/models"
	"github.com/c00/harnesser/runner"
	"github.com/c00/harnesser/tools"
)

type permissionRequest struct {
	toolCallID string
	function   string
	command    string
}

type permissionDecisionMsg struct {
	toolCallID string
	decision   models.ToolCallDecision
}

type permissionDecisionResultMsg struct {
	toolCallID string
	err        error
}

type permissionModel struct {
	requests []permissionRequest
	current  int
	waiting  bool
	width    int
}

func newPermissionModel(pending []runner.PendingToolcall, width int) (*permissionModel, error) {
	if len(pending) == 0 {
		return nil, fmt.Errorf("permission response contains no pending tool calls")
	}

	requests := make([]permissionRequest, 0, len(pending))
	for _, item := range pending {
		builder, err := tools.NewToolCallBuilder(item.ToolCall, item.ToolDefinition)
		if err != nil {
			return nil, fmt.Errorf("cannot render tool call %q: %w", item.ToolCall.ToolCallID, err)
		}
		requests = append(requests, permissionRequest{
			toolCallID: item.ToolCall.ToolCallID,
			function:   item.ToolCall.Function,
			command:    builder.CommandString(),
		})
	}

	return &permissionModel{requests: requests, width: max(1, width)}, nil
}

func (m *permissionModel) Init() tea.Cmd {
	return nil
}

func (m *permissionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.done() || m.waiting {
		return m, nil
	}

	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	var decision models.ToolCallDecision
	switch strings.ToLower(keyMsg.String()) {
	case "y":
		decision = models.ToolCallDecisionApprove
	case "n":
		decision = models.ToolCallDecisionReject
	default:
		return m, nil
	}

	m.waiting = true
	request := m.requests[m.current]
	return m, func() tea.Msg {
		return permissionDecisionMsg{toolCallID: request.toolCallID, decision: decision}
	}
}

func (m *permissionModel) View() tea.View {
	if m.done() {
		return tea.NewView("")
	}

	request := m.requests[m.current]
	status := "[y] Approve  [n] Reject"
	if m.waiting {
		status = "Recording decision..."
	}

	content := fmt.Sprintf(
		"Permission required (%d/%d)\n\nTool: %s\nCommand:\n%s\n\n%s",
		m.current+1,
		len(m.requests),
		request.function,
		request.command,
		status,
	)
	return tea.NewView(lipgloss.NewStyle().Width(max(1, m.width)).Render(content))
}

func (m *permissionModel) SetWidth(width int) {
	m.width = max(1, width)
}

func (m *permissionModel) advance(toolCallID string) bool {
	if m.done() || m.requests[m.current].toolCallID != toolCallID {
		return false
	}
	m.current++
	m.waiting = false
	return true
}

func (m *permissionModel) done() bool {
	return m.current >= len(m.requests)
}
