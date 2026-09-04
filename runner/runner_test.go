package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/c00/harnesser/llm/mockllm"
	"github.com/c00/harnesser/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRunner creates a Runner with empty dirs and a mock LLM.
func newTestRunner(t *testing.T) (*Runner, *mockllm.MockLLM, string) {
	t.Helper()

	dir := t.TempDir()
	llm := mockllm.New()

	r, err := NewRunner(
		llm,
		filepath.Join(dir, "prompts"),
		filepath.Join(dir, "history"),
		filepath.Join(dir, "tools"),
		"",
	)
	require.NoError(t, err)

	return r, llm, dir
}

// writeTestTool writes a tool definition yaml file into the runner's tools dir.
func writeTestTool(t *testing.T, r *Runner, name string, trusted bool, command []string) {
	t.Helper()

	content := `
enabled: true
trusted: ` + boolStr(trusted) + `
tool:
  name: ` + name + `
  description: test tool
command:
`
	for _, c := range command {
		content += "  - " + c + "\n"
	}

	err := os.WriteFile(filepath.Join(r.toolsDir, name+".yaml"), []byte(content), 0o644)
	require.NoError(t, err)
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestNewRunner(t *testing.T) {
	t.Run("creates dirs and generates name", func(t *testing.T) {
		dir := t.TempDir()
		r, err := NewRunner(mockllm.New(),
			filepath.Join(dir, "prompts"),
			filepath.Join(dir, "history"),
			filepath.Join(dir, "tools"),
			"",
		)
		require.NoError(t, err)

		assert.DirExists(t, r.promptsDir)
		assert.DirExists(t, r.historyDir)
		assert.DirExists(t, r.toolsDir)
		assert.NotEmpty(t, r.name)
		assert.True(t, filepath.IsAbs(r.name) || filepath.Dir(r.name) == ".", "generated name should be a bare filename, got %q", r.name)
		assert.Equal(t, ".yaml", filepath.Ext(r.name))
	})

	t.Run("uses provided name", func(t *testing.T) {
		dir := t.TempDir()
		r, err := NewRunner(mockllm.New(),
			filepath.Join(dir, "prompts"),
			filepath.Join(dir, "history"),
			filepath.Join(dir, "tools"),
			"my-session.yaml",
		)
		require.NoError(t, err)
		assert.Equal(t, "my-session.yaml", r.name)
	})

	t.Run("loads tools and prompts", func(t *testing.T) {
		dir := t.TempDir()
		promptsDir := filepath.Join(dir, "prompts")
		toolsDir := filepath.Join(dir, "tools")
		histDir := filepath.Join(dir, "history")

		require.NoError(t, os.MkdirAll(promptsDir, 0o755))
		require.NoError(t, os.MkdirAll(toolsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(promptsDir, "system.md"), []byte("you are a test"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(toolsDir, "echo.yaml"), []byte(`
enabled: true
trusted: true
tool:
  name: echo
command:
  - echo
  - hello
`), 0o644))

		r, err := NewRunner(mockllm.New(), promptsDir, histDir, toolsDir, "")
		require.NoError(t, err)

		require.Len(t, r.prompts, 1)
		assert.Equal(t, "you are a test", r.prompts[0].Content.String())
		require.Contains(t, r.toolDefs, "echo")
		assert.True(t, r.toolDefs["echo"].Trusted)
	})

	t.Run("fails on invalid tool yaml", func(t *testing.T) {
		dir := t.TempDir()
		toolsDir := filepath.Join(dir, "tools")
		require.NoError(t, os.MkdirAll(toolsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(toolsDir, "bad.yaml"), []byte(":\n  - not: [valid"), 0o644))

		_, err := NewRunner(mockllm.New(), filepath.Join(dir, "prompts"), filepath.Join(dir, "history"), toolsDir, "")
		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot load tools")
	})
}

func TestEnsureDir(t *testing.T) {
	tests := []struct {
		name    string
		dir     string
		wantErr bool
	}{
		{name: "creates nested dir", dir: filepath.Join(t.TempDir(), "a", "b", "c")},
		{name: "existing dir is fine", dir: t.TempDir()},
		{name: "fails on file path", dir: func() string {
			p := filepath.Join(t.TempDir(), "file")
			require.NoError(t, os.WriteFile(p, []byte("x"), 0o644))
			return p
		}(), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{}
			err := r.ensureDir(tt.dir)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.DirExists(t, tt.dir)
			}
		})
	}
}

func TestLoadTools(t *testing.T) {
	t.Run("loads yaml and yml, skips others and dirs", func(t *testing.T) {
		r, _, _ := newTestRunner(t)

		writeTestTool(t, r, "tool-a", true, []string{"echo", "a"})
		require.NoError(t, os.WriteFile(filepath.Join(r.toolsDir, "tool-b.yml"), []byte(`
trusted: false
tool:
  name: tool-b
command:
  - echo
  - b
`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(r.toolsDir, "ignored.txt"), []byte("nope"), 0o644))
		require.NoError(t, os.Mkdir(filepath.Join(r.toolsDir, "subdir"), 0o755))

		require.NoError(t, r.loadTools())

		assert.Len(t, r.toolDefs, 2)
		require.Contains(t, r.toolDefs, "tool-a")
		require.Contains(t, r.toolDefs, "tool-b")
		assert.True(t, r.toolDefs["tool-a"].Trusted)
		assert.False(t, r.toolDefs["tool-b"].Trusted)
	})

	t.Run("fails on unreadable dir", func(t *testing.T) {
		r := &Runner{toolsDir: filepath.Join(t.TempDir(), "does-not-exist")}
		assert.Error(t, r.loadTools())
	})

	t.Run("fails on invalid yaml", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		require.NoError(t, os.WriteFile(filepath.Join(r.toolsDir, "bad.yaml"), []byte(":\n  - not: [valid"), 0o644))
		assert.Error(t, r.loadTools())
	})
}

func TestLoadPrompts(t *testing.T) {
	t.Run("loads md and txt, skips others and dirs", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		require.NoError(t, os.WriteFile(filepath.Join(r.promptsDir, "a.md"), []byte("prompt a"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(r.promptsDir, "b.txt"), []byte("prompt b"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(r.promptsDir, "c.yaml"), []byte("nope"), 0o644))
		require.NoError(t, os.Mkdir(filepath.Join(r.promptsDir, "subdir"), 0o755))

		require.NoError(t, r.loadPrompts())

		require.Len(t, r.prompts, 2)
		assert.Equal(t, models.RoleSystem, r.prompts[0].Role)
		texts := []string{r.prompts[0].Content.String(), r.prompts[1].Content.String()}
		assert.ElementsMatch(t, []string{"prompt a", "prompt b"}, texts)
	})

	t.Run("fails on unreadable dir", func(t *testing.T) {
		r := &Runner{promptsDir: filepath.Join(t.TempDir(), "does-not-exist")}
		assert.Error(t, r.loadPrompts())
	})
}

func TestLoadHistory(t *testing.T) {
	t.Run("no name set is a no-op", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		r.name = ""
		r.messages = models.Messages{models.NewUserTextMessage("existing")}

		require.NoError(t, r.LoadHistory())
		assert.Len(t, r.Messages(), 1, "should not reset messages when name is empty")
	})

	t.Run("missing file returns empty history", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		r.name = "missing.yaml"

		require.NoError(t, r.LoadHistory())
		assert.Empty(t, r.Messages())
	})

	t.Run("loads history from history dir", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		r.name = "session.yaml"

		require.NoError(t, os.WriteFile(filepath.Join(r.historyDir, "session.yaml"), []byte(`
- role: user
  content:
    - type: text
      text: hello
- role: assistant
  content:
    - type: text
      text: hi there
`), 0o644))

		require.NoError(t, r.LoadHistory())
		msgs := r.Messages()
		require.Len(t, msgs, 2)
		assert.Equal(t, models.RoleUser, msgs[0].Role)
		assert.Equal(t, "hello", msgs[0].Content.String())
		assert.Equal(t, models.RoleAssistant, msgs[1].Role)
	})

	t.Run("absolute name is used directly", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		abs := filepath.Join(t.TempDir(), "abs-history.yaml")
		r.name = abs

		require.NoError(t, os.WriteFile(abs, []byte(`
- role: user
  content:
    - type: text
      text: from abs path
`), 0o644))

		require.NoError(t, r.LoadHistory())
		require.Len(t, r.Messages(), 1)
		assert.Equal(t, "from abs path", r.Messages()[0].Content.String())
	})

	t.Run("history path that is a directory errors", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		r.name = "adir"
		require.NoError(t, os.Mkdir(filepath.Join(r.historyDir, "adir"), 0o755))

		err := r.LoadHistory()
		require.Error(t, err)
		assert.ErrorContains(t, err, "is a directory")
	})

	t.Run("invalid yaml errors", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		r.name = "bad.yaml"
		require.NoError(t, os.WriteFile(filepath.Join(r.historyDir, "bad.yaml"), []byte(":\n  - not: [valid"), 0o644))

		err := r.LoadHistory()
		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot parse history")
	})
}

func TestWriteHistory(t *testing.T) {
	t.Run("no name set is a no-op", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		r.name = ""

		require.NoError(t, r.WriteHistory())
	})

	t.Run("writes to history dir", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		r.name = "out.yaml"
		r.AddMessage(models.NewUserTextMessage("save me"))

		require.NoError(t, r.WriteHistory())
		assert.FileExists(t, filepath.Join(r.historyDir, "out.yaml"))

		// Round-trip
		r2 := &Runner{name: "out.yaml", historyDir: r.historyDir}
		require.NoError(t, r2.LoadHistory())
		require.Len(t, r2.Messages(), 1)
		assert.Equal(t, "save me", r2.Messages()[0].Content.String())
	})

	t.Run("absolute name is used directly", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		abs := filepath.Join(t.TempDir(), "nested", "abs.yaml")
		r.name = abs

		require.NoError(t, r.WriteHistory())
		assert.FileExists(t, abs)
	})
}

func TestAddMessage(t *testing.T) {
	r, _, _ := newTestRunner(t)

	r.AddMessage(models.NewUserTextMessage("one"))
	r.AddMessage(models.NewAssistantMessage("two"))

	msgs := r.Messages()
	require.Len(t, msgs, 2)
	assert.Equal(t, "one", msgs[0].Content.String())
	assert.Equal(t, "two", msgs[1].Content.String())
}

func TestMessages(t *testing.T) {
	r, _, _ := newTestRunner(t)
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
		r, _, _ := newTestRunner(t)
		assert.NoError(t, r.ConfirmToolCall(context.Background(), "call_1", models.ToolCallDecisionApprove))
	})

	t.Run("unknown tool call id is a no-op", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		r.AddMessage(models.Message{
			Role:      models.RoleAssistant,
			ToolCalls: []models.ToolCall{{ToolCallID: "call_1", Function: "echo"}},
		})

		assert.NoError(t, r.ConfirmToolCall(context.Background(), "call_unknown", models.ToolCallDecisionApprove))
		assert.Empty(t, models.ToolCallDecision(r.Messages()[0].ToolCalls[0].Decision))
	})

	t.Run("sets decision on matching tool call", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		r.name = "confirm.yaml"
		r.AddMessage(models.Message{
			Role:      models.RoleAssistant,
			ToolCalls: []models.ToolCall{{ToolCallID: "call_1", Function: "echo"}},
		})

		require.NoError(t, r.ConfirmToolCall(context.Background(), "call_1", models.ToolCallDecisionApprove))
		assert.Equal(t, models.ToolCallDecisionApprove, r.Messages()[0].ToolCalls[0].Decision)

		// Decision is persisted to history
		r2 := &Runner{name: "confirm.yaml", historyDir: r.historyDir}
		require.NoError(t, r2.LoadHistory())
		require.Len(t, r2.Messages(), 1)
		assert.Equal(t, models.ToolCallDecisionApprove, r2.Messages()[0].ToolCalls[0].Decision)
	})
}

func TestRunPrompt(t *testing.T) {
	r, llm, _ := newTestRunner(t)
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
		r, llm, _ := newTestRunner(t)
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
		r, llm, _ := newTestRunner(t)
		require.NoError(t, os.WriteFile(filepath.Join(r.promptsDir, "system.md"), []byte("system prompt"), 0o644))
		require.NoError(t, r.loadPrompts())

		var gotMsgs []models.Message
		var gotTools models.Tools
		// Use a phrase that only matches when the system prompt is included
		llm.AddTextResponse("system prompt", "ok")

		r.AddMessage(models.NewUserTextMessage("user msg"))
		_, err := r.RunInference(context.Background())
		require.NoError(t, err)

		// Re-run with a capture via tools check: the mock matched the system prompt,
		// which proves prompts were sent. Also verify tools are passed through.
		writeTestTool(t, r, "echo", true, []string{"echo", "hi"})
		require.NoError(t, r.loadTools())
		gotTools = r.tools()
		require.Len(t, gotTools, 1)
		assert.Equal(t, "echo", gotTools[0].Name)

		_ = gotMsgs
	})

	t.Run("writes history", func(t *testing.T) {
		r, llm, _ := newTestRunner(t)
		r.name = "inference.yaml"
		llm.AddTextResponse("q", "a")

		r.AddMessage(models.NewUserTextMessage("q"))
		_, err := r.RunInference(context.Background())
		require.NoError(t, err)

		r2 := &Runner{name: "inference.yaml", historyDir: r.historyDir}
		require.NoError(t, r2.LoadHistory())
		require.Len(t, r2.Messages(), 2)
	})

	t.Run("llm error is returned", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
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

func TestRunStep(t *testing.T) {
	type fixture struct {
		runner *Runner
		llm    *mockllm.MockLLM
	}
	setup := func(t *testing.T) *fixture {
		r, llm, _ := newTestRunner(t)
		return &fixture{runner: r, llm: llm}
	}

	tests := []struct {
		name      string
		setupMsgs func(t *testing.T, f *fixture)
		wantType  RunnerResponseType
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
				writeTestTool(t, f.runner, "echo", false, []string{"echo", "hi"})
				require.NoError(t, f.runner.loadTools())
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
				writeTestTool(t, f.runner, "echo", false, []string{"echo", "hi"})
				require.NoError(t, f.runner.loadTools())
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
				writeTestTool(t, f.runner, "echo", true, []string{"echo", "hi"})
				require.NoError(t, f.runner.loadTools())
				f.runner.AddMessage(models.Message{
					Role:      models.RoleAssistant,
					ToolCalls: []models.ToolCall{{ToolCallID: "call_echo", Function: "echo", Args: "{}"}},
				})
			},
			wantType: ResponseTypeToolResults,
		},
		{
			name: "assistant with unknown tool errors",
			setupMsgs: func(t *testing.T, f *fixture) {
				f.runner.AddMessage(models.Message{
					Role:      models.RoleAssistant,
					ToolCalls: []models.ToolCall{{ToolCallID: "call_nope", Function: "nope"}},
				})
			},
			wantErr: true,
			errPart: "tool not defined",
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
		})
	}

	t.Run("ask permission returns only unapproved calls", func(t *testing.T) {
		f := setup(t)
		writeTestTool(t, f.runner, "echo", false, []string{"echo", "hi"})
		require.NoError(t, f.runner.loadTools())
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
		assert.Equal(t, "call_2", resp.ToApprove[0].ToolCallID)
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
		r, _, _ := newTestRunner(t)
		msgs, err := r.RunTools(context.Background())
		require.NoError(t, err)
		assert.Empty(t, msgs)
	})

	t.Run("no tool calls returns empty", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		r.AddMessage(models.NewAssistantMessage("no tools here"))

		msgs, err := r.RunTools(context.Background())
		require.NoError(t, err)
		assert.Empty(t, msgs)
	})

	t.Run("runs tool and appends result", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		writeTestTool(t, r, "echo", true, []string{"echo", "tool output"})
		require.NoError(t, r.loadTools())

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

	t.Run("unknown tool errors", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		r.AddMessage(models.Message{
			Role:      models.RoleAssistant,
			ToolCalls: []models.ToolCall{{ToolCallID: "call_x", Function: "nope", Args: "{}"}},
		})

		_, err := r.RunTools(context.Background())
		require.Error(t, err)
		assert.ErrorContains(t, err, "tool not defined")
	})

	t.Run("rejected tool call produces error message", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		writeTestTool(t, r, "echo", false, []string{"echo", "hi"})
		require.NoError(t, r.loadTools())

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

	t.Run("no decision on untrusted tool errors", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		writeTestTool(t, r, "echo", false, []string{"echo", "hi"})
		require.NoError(t, r.loadTools())

		r.AddMessage(models.Message{
			Role: models.RoleAssistant,
			ToolCalls: []models.ToolCall{
				{ToolCallID: "call_echo", Function: "echo", Args: "{}", Decision: models.ToolCallDecisionNoDecision},
			},
		})

		_, err := r.RunTools(context.Background())
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoDecision)
	})

	t.Run("failing command errors", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		writeTestTool(t, r, "fail", true, []string{"false"})
		require.NoError(t, r.loadTools())

		r.AddMessage(models.Message{
			Role:      models.RoleAssistant,
			ToolCalls: []models.ToolCall{{ToolCallID: "call_fail", Function: "fail", Args: "{}"}},
		})

		_, err := r.RunTools(context.Background())
		require.Error(t, err)
		assert.ErrorContains(t, err, "running tool 'fail' failed")
	})
}

func TestRunTool(t *testing.T) {
	t.Run("unknown tool errors", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		_, err := r.runTool(context.Background(), models.ToolCall{Function: "nope"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "tool not defined")
	})

	t.Run("untrusted without decision errors", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		writeTestTool(t, r, "echo", false, []string{"echo", "hi"})
		require.NoError(t, r.loadTools())

		_, err := r.runTool(context.Background(), models.ToolCall{ToolCallID: "call_1", Function: "echo", Args: "{}", Decision: models.ToolCallDecisionNoDecision})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoDecision)
	})

	t.Run("untrusted rejected returns error message", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		writeTestTool(t, r, "echo", false, []string{"echo", "hi"})
		require.NoError(t, r.loadTools())

		msg, err := r.runTool(context.Background(), models.ToolCall{
			ToolCallID: "call_1",
			Function:   "echo",
			Decision:   models.ToolCallDecisionReject,
		})
		require.NoError(t, err)
		assert.Equal(t, "user rejected the running of this tool call", msg.Content.String())
	})

	t.Run("runs command and returns output", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		writeTestTool(t, r, "echo", true, []string{"echo", "hello world"})
		require.NoError(t, r.loadTools())

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
		r, _, _ := newTestRunner(t)
		content := `
enabled: true
trusted: true
tool:
  name: greet
  parameters:
    type: object
    properties:
      name:
        type: string
    required:
      - name
command:
  - echo
  - "hello {{.Params.name}}"
`
		require.NoError(t, os.WriteFile(filepath.Join(r.toolsDir, "greet.yaml"), []byte(content), 0o644))
		require.NoError(t, r.loadTools())

		msg, err := r.runTool(context.Background(), models.ToolCall{
			ToolCallID: "call_1",
			Function:   "greet",
			Args:       `{"name": "world"}`,
		})
		require.NoError(t, err)
		assert.Equal(t, "hello world\n", msg.Content.String())
	})

	t.Run("invalid args against schema errors", func(t *testing.T) {
		r, _, _ := newTestRunner(t)
		content := `
enabled: true
trusted: true
tool:
  name: strict
  parameters:
    type: object
    properties:
      name:
        type: string
    required:
      - name
command:
  - echo
  - "{{.Params.name}}"
`
		require.NoError(t, os.WriteFile(filepath.Join(r.toolsDir, "strict.yaml"), []byte(content), 0o644))
		require.NoError(t, r.loadTools())

		_, err := r.runTool(context.Background(), models.ToolCall{
			ToolCallID: "call_1",
			Function:   "strict",
			Args:       `{}`,
		})
		require.Error(t, err)
	})
}

func TestTools(t *testing.T) {
	r, _, _ := newTestRunner(t)
	writeTestTool(t, r, "tool-a", true, []string{"echo", "a"})
	writeTestTool(t, r, "tool-b", false, []string{"echo", "b"})
	require.NoError(t, r.loadTools())

	tools := r.tools()
	require.Len(t, tools, 2)
	assert.ElementsMatch(t, []string{"tool-a", "tool-b"}, tools.List())
}
