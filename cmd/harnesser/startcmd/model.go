package startcmd

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/c00/harnesser/models"
	"github.com/c00/harnesser/runner"
)

// model for BubbleTea
type tuiModel struct {
	ctx context.Context

	// spinner is active when LLM is running
	spinner spinner.Model
	// textarea for input
	textarea textarea.Model
	// viewport for output
	viewport viewport.Model
	// permission asks the user to approve or reject pending tool calls
	permission *permissionModel

	// agent is the LLM Agent
	agent *runner.Runner

	state tuiState
	err   error
	height int
}

type runStepResultMsg struct {
	response runner.Response
	err      error
}

type tuiState string

const (
	ready         tuiState = "ready"
	busy          tuiState = "busy"
	askPermission tuiState = "ask-permission"
)

func initialModel(ctx context.Context, agent *runner.Runner) tuiModel {
	// 1. Initialize Spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// 2. Initialize Textarea
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.Blur()
	ta.SetWidth(30)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.DynamicHeight = true
	ta.MinHeight = 1
	ta.MaxHeight = 10

	return tuiModel{
		ctx:      ctx,
		agent:    agent,
		spinner:  s,
		textarea: ta,
		viewport: viewport.New(),
		state:    busy,
	}
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, textarea.Blink, agentStepCmd(m.ctx, m.agent))
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd, vpCmd, spCmd tea.Cmd
		cmds                []tea.Cmd
	)

	// a) Handle global keys
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if m.state != ready {
				// Ignore if we're not ready for input
				return m, nil
			}

			input := strings.TrimSpace(m.textarea.Value())
			if input != "" {
				m.state = busy
				m.textarea.Blur()
				m.agent.AddMessage(models.NewUserTextMessage(input))

				cmds = append(cmds, agentStepCmd(m.ctx, m.agent), m.spinner.Tick)

				m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width()).Render(m.agent.Messages().String()))
				m.textarea.Reset()
				m.updateViewportHeight()
				m.viewport.GotoBottom()

				return m, tea.Batch(cmds...)
			}
		}

	case runStepResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}

		// Handle the LLM response
		switch msg.response.Type {
		case runner.ResponseTypeNew,
			runner.ResponseTypeInferenceResultNoTools:
			// Just wait for more input
			m.state = ready
			cmds = append(cmds, m.textarea.Focus())
		case runner.ResponseTypeInferenceResultWithTools,
			runner.ResponseTypeToolResults:
			m.state = busy // For clarity

			// Run another step
			cmds = append(cmds, agentStepCmd(m.ctx, m.agent))

		case runner.ResponseTypeDone:
			return m, tea.Quit
		case runner.ResponseTypeAskPermission:
			permission, err := newPermissionModel(msg.response.ToApprove, m.viewport.Width())
			if err != nil {
				m.err = fmt.Errorf("cannot ask permission: %w", err)
				return m, tea.Quit
			}
			m.state = askPermission
			m.textarea.Blur()
			m.permission = permission
		}
		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width()).Render(m.agent.Messages().String()))
		m.viewport.GotoBottom()

	case permissionDecisionMsg:
		if m.permission == nil {
			m.err = fmt.Errorf("received a permission decision without an active permission prompt")
			return m, tea.Quit
		}
		return m, confirmToolCallCmd(m.ctx, m.agent, msg)

	case permissionDecisionResultMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("cannot record tool call decision: %w", msg.err)
			return m, tea.Quit
		}
		if m.permission == nil || !m.permission.advance(msg.toolCallID) {
			m.err = fmt.Errorf("received a decision result for unexpected tool call %q", msg.toolCallID)
			return m, tea.Quit
		}
		if m.permission.done() {
			m.permission = nil
			m.state = busy
			cmds = append(cmds, agentStepCmd(m.ctx, m.agent), m.spinner.Tick)
		}

	case tea.WindowSizeMsg:
		// b) Layout Calculation
		m.height = msg.Height
		m.viewport.SetWidth(msg.Width)
		m.textarea.SetWidth(msg.Width)
		if m.permission != nil {
			m.permission.SetWidth(msg.Width)
		}
	}

	// c) Delegate updates to sub-components
	switch m.state {
	case ready:
		m.textarea, tiCmd = m.textarea.Update(msg)
	case busy:
		m.viewport, vpCmd = m.viewport.Update(msg)
	case askPermission:
		if m.permission != nil {
			_, tiCmd = m.permission.Update(msg)
		}
	}
	m.spinner, spCmd = m.spinner.Update(msg)
	m.updateViewportHeight()

	cmds = append(cmds, tiCmd, vpCmd, spCmd)
	return m, tea.Batch(cmds...)
}

func (m *tuiModel) updateViewportHeight() {
	if m.height <= 0 {
		return
	}

	// Reserve one line between the viewport and textarea, plus the status line.
	verticalMargin := m.textarea.Height() + 2
	m.viewport.SetHeight(max(1, m.height-verticalMargin))
}

func (m tuiModel) View() tea.View {
	spinnerView := ""
	if m.state == busy {
		spinnerView = fmt.Sprintf(" %s working...", m.spinner.View())
	}

	var content string
	switch m.state {
	case ready:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			m.viewport.View(),
			"\n"+m.textarea.View(),
			"",
		)
	case busy:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			m.viewport.View(),
			"\n"+m.textarea.View(),
			fmt.Sprintf(" %s working...", m.spinner.View()),
		)
	case askPermission:
		if m.permission == nil {
			content = m.viewport.View()
			break
		}
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			m.viewport.View(),
			m.permission.View().Content,
		)
	default:
		// d) Assemble the final view
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			m.viewport.View(),
			"\n"+m.textarea.View(),
			spinnerView,
		)
	}

	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

func agentStepCmd(ctx context.Context, agent *runner.Runner) tea.Cmd {
	return func() tea.Msg {
		resp, err := agent.RunStep(ctx)

		return runStepResultMsg{
			response: resp,
			err:      err,
		}
	}
}

func confirmToolCallCmd(ctx context.Context, agent *runner.Runner, msg permissionDecisionMsg) tea.Cmd {
	return func() tea.Msg {
		err := agent.ConfirmToolCall(ctx, msg.toolCallID, msg.decision)
		return permissionDecisionResultMsg{toolCallID: msg.toolCallID, err: err}
	}
}
