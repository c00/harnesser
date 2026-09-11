package startcmd

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
)

func TestViewportHeightTracksTextareaHeight(t *testing.T) {
	model := initialModel(t.Context(), nil)
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
