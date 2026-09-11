package startcmd

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/c00/harnesser/models"
	"github.com/c00/harnesser/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionModelDecision(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		wantDecision models.ToolCallDecision
	}{
		{name: "approve", key: "y", wantDecision: models.ToolCallDecisionApprove},
		{name: "reject", key: "n", wantDecision: models.ToolCallDecisionReject},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := newPermissionModel([]runner.PendingToolcall{{
				ToolCall: models.ToolCall{ToolCallID: "call_1", Function: "echo", Args: `{}`},
				ToolDefinition: models.ToolDefinition{
					Tool:    models.Tool{Name: "echo"},
					Command: []string{"echo"},
				},
			}}, 80)
			require.NoError(t, err)

			_, cmd := model.Update(tea.KeyPressMsg{Code: rune(tt.key[0]), Text: tt.key})
			require.NotNil(t, cmd)
			msg, ok := cmd().(permissionDecisionMsg)
			require.True(t, ok)
			assert.Equal(t, "call_1", msg.toolCallID)
			assert.Equal(t, tt.wantDecision, msg.decision)
		})
	}
}

func TestPermissionModelAdvancesRequests(t *testing.T) {
	model := &permissionModel{
		requests: []permissionRequest{
			{toolCallID: "call_1"},
			{toolCallID: "call_2"},
		},
		waiting: true,
	}

	assert.True(t, model.advance("call_1"))
	assert.False(t, model.done())
	assert.False(t, model.waiting)
	assert.False(t, model.advance("call_1"))
	assert.True(t, model.advance("call_2"))
	assert.True(t, model.done())
}
