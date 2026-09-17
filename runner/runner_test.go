package runner

import (
	"context"
	"testing"

	"github.com/c00/harnesser/historyprovider"
	"github.com/c00/harnesser/llm/mockllm"
	"github.com/c00/harnesser/models"
	"github.com/c00/harnesser/promptsprovider"
	"github.com/c00/harnesser/toolsprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockPromptProvider() *promptsprovider.MemoryProvider {
	p := &promptsprovider.MemoryProvider{}
	p.AddPrompt(promptsprovider.NewPrompt("foo", "You are a helpful assistant"))
	return p
}

func newHistProvider() *historyprovider.MemoryProvider {
	return historyprovider.NewMemoryProvider()
}

// newTestRunner creates a Runner with empty dirs and a mock LLM.
func newTestRunner(t *testing.T) (*Runner, *mockllm.MockLLM) {
	t.Helper()

	llm := mockllm.New()

	r := NewRunner(
		llm,
		newMockPromptProvider(),
		newHistProvider(),
		toolsprovider.NewMemoryProvider(),
		"",
	)

	return r, llm
}

// addTestTool writes a tool definition yaml file into the runner's tools dir.
func addTestTool(t *testing.T, r *Runner, name string, trusted bool, command []string) {
	t.Helper()

	def := models.ToolDefinition{
		Enabled: true,
		Trusted: trusted,
		Tool: models.Tool{
			Name:        name,
			Description: "test tool",
		},
		Command: command,
	}

	p, ok := r.toolsProvider.(*toolsprovider.MemoryProvider)
	assert.True(t, ok)

	p.AddDefinition(def)
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestAddMessage(t *testing.T) {
	r, _ := newTestRunner(t)

	r.AddMessage(models.NewUserTextMessage("one"))
	r.AddMessage(models.NewAssistantMessage("two"))

	msgs := r.Messages()
	require.Len(t, msgs, 2)
	assert.Equal(t, "one", msgs[0].Content.String())
	assert.Equal(t, "two", msgs[1].Content.String())
}

func TestMessages(t *testing.T) {
	r, _ := newTestRunner(t)
	r.AddMessage(models.NewUserTextMessage("original"))

	got := r.Messages()
	require.Len(t, got, 1)

	// Mutating the returned copy (append/reorder) must not affect the runner's messages.
	// Note: this is a shallow copy - nested Content slices are still shared.
	got = append(got, models.NewAssistantMessage("extra"))

	assert.Len(t, r.Messages(), 1)
	assert.Equal(t, "original", r.Messages()[0].Content.String())
}

func TestConfirmToolCall(t *testing.T) {
	t.Run("empty messages is a no-op", func(t *testing.T) {
		r, _ := newTestRunner(t)
		assert.NoError(t, r.ConfirmToolCall(context.Background(), "call_1", models.ToolCallDecisionApprove))
	})

	t.Run("unknown tool call id is a no-op", func(t *testing.T) {
		r, _ := newTestRunner(t)
		r.AddMessage(models.Message{
			Role:      models.RoleAssistant,
			ToolCalls: []models.ToolCall{{ToolCallID: "call_1", Function: "echo"}},
		})

		assert.NoError(t, r.ConfirmToolCall(context.Background(), "call_unknown", models.ToolCallDecisionApprove))
		assert.Empty(t, models.ToolCallDecision(r.Messages()[0].ToolCalls[0].Decision))
	})

	t.Run("sets decision on matching tool call", func(t *testing.T) {
		r, _ := newTestRunner(t)
		r.histProv.Select("confirm.yaml")

		r.AddMessage(models.Message{
			Role:      models.RoleAssistant,
			ToolCalls: []models.ToolCall{{ToolCallID: "call_1", Function: "echo"}},
		})

		require.NoError(t, r.ConfirmToolCall(context.Background(), "call_1", models.ToolCallDecisionApprove))
		assert.Equal(t, models.ToolCallDecisionApprove, r.Messages()[0].ToolCalls[0].Decision)

		// Decision is persisted to history
		entry := r.histProv.Get()

		require.Len(t, entry.Messages, 1)
		assert.Equal(t, models.ToolCallDecisionApprove, entry.Messages[0].ToolCalls[0].Decision)
	})
}

func TestRunPrompt(t *testing.T) {
	r, llm := newTestRunner(t)
	llm.AddTextResponse("hello", "hi there")

	resp, err := r.RunPrompt(context.Background(), models.NewUserTextMessage("hello"))
	require.NoError(t, err)
	assert.Equal(t, "hi there", resp.Content.String())

	// User message + assistant response are both in history
	msgs := r.Messages()
	require.Len(t, msgs, 2)
	assert.Equal(t, models.RoleUser, msgs[0].Role)
	assert.Equal(t, models.RoleAssistant, msgs[1].Role)
}

func TestRunInference(t *testing.T) {
	t.Run("returns llm response and appends it", func(t *testing.T) {
		r, llm := newTestRunner(t)
		llm.AddTextResponse("question", "the answer")

		r.AddMessage(models.NewUserTextMessage("question"))

		resp, err := r.RunInference(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "the answer", resp.Content.String())
		assert.Equal(t, models.RoleAssistant, resp.Role)

		msgs := r.Messages()
		require.Len(t, msgs, 2)
		assert.Equal(t, "the answer", msgs[1].Content.String())
	})

	t.Run("sends prompts plus messages to the llm", func(t *testing.T) {
		r, _ := newTestRunner(t)
		provider := &promptsprovider.MemoryProvider{}
		provider.AddPrompt(promptsprovider.NewPrompt("system", "system prompt"))
		provider.AddPrompt(promptsprovider.Prompt{
			Active:  false,
			Key:     "inactive",
			Message: models.NewSystemMessage("inactive prompt"),
		})
		r.promptsProvider = provider

		llm := &capturingLLM{}
		r.llm = llm
		addTestTool(t, r, "echo", true, []string{"echo", "hi"})

		r.AddMessage(models.NewUserTextMessage("user msg"))
		_, err := r.RunInference(t.Context())
		require.NoError(t, err)

		require.Len(t, llm.messages, 2)
		assert.Equal(t, models.RoleSystem, llm.messages[0].Role)
		assert.Equal(t, "system prompt", llm.messages[0].Content.String())
		assert.Equal(t, models.RoleUser, llm.messages[1].Role)
		assert.Equal(t, "user msg", llm.messages[1].Content.String())
		require.Len(t, llm.tools, 1)
		assert.Equal(t, "echo", llm.tools[0].Name)
	})

	t.Run("llm error is returned", func(t *testing.T) {
		r, _ := newTestRunner(t)
		// MockLLM errors when there are no messages; simulate by clearing prompts and messages
		// is not possible via RunInference, so use a failing provider instead.
		r.llm = &failingLLM{}
		r.AddMessage(models.NewUserTextMessage("q"))

		_, err := r.RunInference(context.Background())
		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot generate llm response")
	})
}

// failingLLM always errors, for testing error paths.
type failingLLM struct{}

func (f *failingLLM) Config() models.LlmConfig { return models.LlmConfig{Name: "failing"} }
func (f *failingLLM) Generate(ctx context.Context, messages []models.Message, tools []models.Tool) (models.Message, error) {
	return models.Message{}, assert.AnError
}

type capturingLLM struct {
	messages models.Messages
	tools    models.Tools
}

func (c *capturingLLM) Config() models.LlmConfig { return models.LlmConfig{Name: "capturing"} }
func (c *capturingLLM) Generate(_ context.Context, messages []models.Message, tools []models.Tool) (models.Message, error) {
	c.messages = append(models.Messages{}, messages...)
	c.tools = append(models.Tools{}, tools...)
	return models.NewAssistantMessage("ok"), nil
}

func TestRunStep(t *testing.T) {
	type fixture struct {
		runner *Runner
		llm    *mockllm.MockLLM
	}
	setup := func(t *testing.T) *fixture {
		r, llm := newTestRunner(t)
		return &fixture{runner: r, llm: llm}
	}

	tests := []struct {
		name      string
		setupMsgs func(t *testing.T, f *fixture)
		wantType  RunnerResponseType
		wantMsg   string
		wantErr   bool
		errPart   string
	}{
		{
			name:      "no messages returns new",
			setupMsgs: func(t *testing.T, f *fixture) {},
			wantType:  ResponseTypeNew,
		},
		{
			name: "last message user runs inference without tool calls",
			setupMsgs: func(t *testing.T, f *fixture) {
				f.llm.AddTextResponse("hello", "hi")
				f.runner.AddMessage(models.NewUserTextMessage("hello"))
			},
			wantType: ResponseTypeInferenceResultNoTools,
		},
		{
			name: "last message user runs inference with tool calls",
			setupMsgs: func(t *testing.T, f *fixture) {
				f.llm.AddToolcallResponse("do it", "echo", "{}", "")
				f.runner.AddMessage(models.NewUserTextMessage("do it"))
			},
			wantType: ResponseTypeInferenceResultWithTools,
		},
		{
			name: "assistant with unapproved untrusted tool call asks permission",
			setupMsgs: func(t *testing.T, f *fixture) {
				addTestTool(t, f.runner, "echo", false, []string{"echo", "hi"})
				f.runner.AddMessage(models.Message{
					Role:      models.RoleAssistant,
					ToolCalls: []models.ToolCall{{ToolCallID: "call_echo", Function: "echo"}},
				})
			},
			wantType: ResponseTypeAskPermission,
		},
		{
			name: "assistant with approved untrusted tool call runs tools",
			setupMsgs: func(t *testing.T, f *fixture) {
				addTestTool(t, f.runner, "echo", false, []string{"echo", "hi"})
				f.runner.AddMessage(models.Message{
					Role: models.RoleAssistant,
					ToolCalls: []models.ToolCall{
						{ToolCallID: "call_echo", Function: "echo", Args: "{}", Decision: models.ToolCallDecisionApprove},
					},
				})
			},
			wantType: ResponseTypeToolResults,
		},
		{
			name: "assistant with trusted tool call runs tools without approval",
			setupMsgs: func(t *testing.T, f *fixture) {
				addTestTool(t, f.runner, "echo", true, []string{"echo", "hi"})
				f.runner.AddMessage(models.Message{
					Role:      models.RoleAssistant,
					ToolCalls: []models.ToolCall{{ToolCallID: "call_echo", Function: "echo", Args: "{}"}},
				})
			},
			wantType: ResponseTypeToolResults,
		},
		{
			name: "assistant with unknown tool returns error message",
			setupMsgs: func(t *testing.T, f *fixture) {
				f.runner.AddMessage(models.Message{
					Role:      models.RoleAssistant,
					ToolCalls: []models.ToolCall{{ToolCallID: "call_nope", Function: "nope", Args: "{}"}},
				})
			},
			wantType: ResponseTypeToolResults,
			wantMsg:  "tool not found: nope",
		},
		{
			name: "assistant with failing command returns error message",
			setupMsgs: func(t *testing.T, f *fixture) {
				addTestTool(t, f.runner, "fail", true, []string{"false"})
				f.runner.AddMessage(models.Message{
					Role:      models.RoleAssistant,
					ToolCalls: []models.ToolCall{{ToolCallID: "call_fail", Function: "fail", Args: "{}"}},
				})
			},
			wantType: ResponseTypeToolResults,
			wantMsg:  "cannot run tool call 'fail': running tool 'fail' failed",
		},
		{
			name: "assistant text message is done",
			setupMsgs: func(t *testing.T, f *fixture) {
				f.runner.AddMessage(models.NewAssistantMessage("all done"))
			},
			wantType: ResponseTypeDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := setup(t)
			tt.setupMsgs(t, f)

			resp, err := f.runner.RunStep(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.errPart)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantType, resp.Type)
			if tt.wantMsg != "" {
				require.Len(t, resp.Messages, 1)
				assert.Contains(t, resp.Messages[0].Content.String(), tt.wantMsg)
			}
		})
	}

	t.Run("ask permission returns only unapproved calls", func(t *testing.T) {
		f := setup(t)
		addTestTool(t, f.runner, "echo", false, []string{"echo", "hi"})
		f.runner.AddMessage(models.Message{
			Role: models.RoleAssistant,
			ToolCalls: []models.ToolCall{
				{ToolCallID: "call_1", Function: "echo", Decision: models.ToolCallDecisionApprove},
				{ToolCallID: "call_2", Function: "echo"},
			},
		})

		resp, err := f.runner.RunStep(context.Background())
		require.NoError(t, err)
		assert.Equal(t, ResponseTypeAskPermission, resp.Type)
		require.Len(t, resp.ToApprove, 1)
		assert.Equal(t, "call_2", resp.ToApprove[0].ToolCall.ToolCallID)
	})

	t.Run("inference error is returned", func(t *testing.T) {
		f := setup(t)
		f.runner.llm = &failingLLM{}
		f.runner.AddMessage(models.NewUserTextMessage("q"))

		_, err := f.runner.RunStep(context.Background())
		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot run inference")
	})
}

func TestRunTools(t *testing.T) {
	t.Run("no messages returns empty", func(t *testing.T) {
		r, _ := newTestRunner(t)
		msgs, err := r.RunTools(context.Background())
		require.NoError(t, err)
		assert.Empty(t, msgs)
	})

	t.Run("no tool calls returns empty", func(t *testing.T) {
		r, _ := newTestRunner(t)
		r.AddMessage(models.NewAssistantMessage("no tools here"))

		msgs, err := r.RunTools(context.Background())
		require.NoError(t, err)
		assert.Empty(t, msgs)
	})

	t.Run("runs tool and appends result", func(t *testing.T) {
		r, _ := newTestRunner(t)
		addTestTool(t, r, "echo", true, []string{"echo", "tool output"})

		r.AddMessage(models.Message{
			Role:      models.RoleAssistant,
			ToolCalls: []models.ToolCall{{ToolCallID: "call_echo", Function: "echo", Args: "{}"}},
		})

		msgs, err := r.RunTools(context.Background())
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Equal(t, models.RoleTool, msgs[0].Role)
		assert.Equal(t, "call_echo", msgs[0].ToolID)
		assert.Equal(t, "tool output\n", msgs[0].Content.String())

		// Result is appended to runner messages
		assert.Len(t, r.Messages(), 2)
	})

	t.Run("rejected tool call produces error message", func(t *testing.T) {
		r, _ := newTestRunner(t)
		addTestTool(t, r, "echo", false, []string{"echo", "hi"})

		r.AddMessage(models.Message{
			Role: models.RoleAssistant,
			ToolCalls: []models.ToolCall{
				{ToolCallID: "call_echo", Function: "echo", Args: "{}", Decision: models.ToolCallDecisionReject},
			},
		})

		msgs, err := r.RunTools(context.Background())
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Equal(t, "user rejected the running of this tool call", msgs[0].Content.String())
	})

}

func TestRunTool(t *testing.T) {
	t.Run("unknown tool errors", func(t *testing.T) {
		r, _ := newTestRunner(t)
		_, err := r.runTool(context.Background(), models.ToolCall{Function: "nope"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "tool not found")
	})

	t.Run("untrusted without decision errors", func(t *testing.T) {
		r, _ := newTestRunner(t)
		addTestTool(t, r, "echo", false, []string{"echo", "hi"})

		_, err := r.runTool(context.Background(), models.ToolCall{ToolCallID: "call_1", Function: "echo", Args: "{}", Decision: models.ToolCallDecisionNoDecision})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoDecision)
	})

	t.Run("untrusted rejected returns error message", func(t *testing.T) {
		r, _ := newTestRunner(t)
		addTestTool(t, r, "echo", false, []string{"echo", "hi"})

		msg, err := r.runTool(context.Background(), models.ToolCall{
			ToolCallID: "call_1",
			Function:   "echo",
			Decision:   models.ToolCallDecisionReject,
		})
		require.NoError(t, err)
		assert.Equal(t, "user rejected the running of this tool call", msg.Content.String())
	})

	t.Run("runs command and returns output", func(t *testing.T) {
		r, _ := newTestRunner(t)
		addTestTool(t, r, "echo", true, []string{"echo", "hello world"})

		msg, err := r.runTool(context.Background(), models.ToolCall{
			ToolCallID: "call_1",
			Function:   "echo",
			Args:       "{}",
		})
		require.NoError(t, err)
		assert.Equal(t, models.RoleTool, msg.Role)
		assert.Equal(t, "call_1", msg.ToolID)
		assert.Equal(t, "hello world\n", msg.Content.String())
	})

	t.Run("interpolates args into command", func(t *testing.T) {
		r, _ := newTestRunner(t)
		def := models.ToolDefinition{
			Enabled: true,
			Trusted: true,
			Tool: models.Tool{
				Name: "greet",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type": "string",
						},
					},
					"required": []string{"name"},
				},
			},
			Command: []string{"echo", "hello {{.Params.name}}"},
		}

		p, ok := r.toolsProvider.(*toolsprovider.MemoryProvider)
		assert.True(t, ok)
		p.AddDefinition(def)

		msg, err := r.runTool(context.Background(), models.ToolCall{
			ToolCallID: "call_1",
			Function:   "greet",
			Args:       `{"name": "world"}`,
		})
		require.NoError(t, err)
		assert.Equal(t, "hello world\n", msg.Content.String())
	})

	t.Run("invalid args against schema errors", func(t *testing.T) {
		r, _ := newTestRunner(t)
		def := models.ToolDefinition{
			Enabled: true,
			Trusted: true,
			Tool: models.Tool{
				Name: "greet",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type": "string",
						},
					},
					"required": []string{"name"},
				},
			},
			Command: []string{"echo", "hello {{.Params.name}}"},
		}

		p, ok := r.toolsProvider.(*toolsprovider.MemoryProvider)
		assert.True(t, ok)
		p.AddDefinition(def)

		_, err := r.runTool(context.Background(), models.ToolCall{
			ToolCallID: "call_1",
			Function:   "strict",
			Args:       `{}`,
		})
		require.Error(t, err)
	})
}
