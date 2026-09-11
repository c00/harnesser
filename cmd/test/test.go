package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type model struct {
	spinner  spinner.Model
	textarea textarea.Model
	viewport viewport.Model
	history  []string
	ready    bool
}

func initialModel() model {
	// 1. Initialize Spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// 2. Initialize Textarea
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.Focus()
	ta.SetWidth(30)
	ta.SetHeight(3)

	return model{
		spinner:  s,
		textarea: ta,
		history:  []string{"Welcome! Type something below."},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, textarea.Blink)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			// Logic: Move text from textarea to viewport
			input := m.textarea.Value()
			if strings.TrimSpace(input) != "" {
				m.history = append(m.history, "> "+input)
				m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width()).Render(strings.Join(m.history, "\n")))
				m.textarea.Reset()
				m.viewport.GotoBottom()
			} else {
				m.history = append(m.history, fmt.Sprintf("** GOT EMPTY LINES: %v", len(input)))
				m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width()).Render(strings.Join(m.history, "\n")))
				m.textarea.Reset()
				m.viewport.GotoBottom()
			}
		}

	case tea.WindowSizeMsg:
		// b) Layout Calculation
		// Reserve space for spinner (1) and textarea (5 including borders)
		verticalMargin := 1 + 5

		if !m.ready {
			m.viewport = viewport.New(viewport.WithWidth(msg.Width), viewport.WithHeight(msg.Height-verticalMargin))
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width()).Render(strings.Join(m.history, "\n")))
			m.ready = true
		} else {
			m.viewport.SetWidth(msg.Width)
			m.viewport.SetHeight(msg.Height - verticalMargin)
		}
		m.textarea.SetWidth(msg.Width)
	}

	// c) Delegate updates to sub-components
	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.spinner, spCmd = m.spinner.Update(msg)

	cmds = append(cmds, tiCmd, vpCmd, spCmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	if !m.ready {
		view := tea.NewView("\n  Initializing...")
		view.AltScreen = true
		return view
	}

	// d) Assemble the final view
	view := tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		fmt.Sprintf(" %s Processing Status...", m.spinner.View()),
		m.viewport.View(),
		"\n"+m.textarea.View(),
	))
	view.AltScreen = true
	return view
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
	}
}
