package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/c00/harnesser/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTempHome creates separate temp dirs for the user home and the working
// directory, so tests can control what Load() sees for "~/.harnesser" and
// "./.harnesser" independently. Returns (home, workdir).
func setupTempHome(t *testing.T) (string, string) {
	t.Helper()
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(work)
	return home, work
}

func userConfigPath(home string) string {
	return filepath.Join(home, DirName, ConfigFile)
}

func localConfigPath(work string) string {
	return filepath.Join(work, DirName, ConfigFile)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestLoad_Defaults(t *testing.T) {
	setupTempHome(t)

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, Defaults(), cfg)
	// No local .harnesser dir, so DataDir must stay empty.
	assert.Empty(t, cfg.DataDir)
}

func TestLoad_UserConfigOnly(t *testing.T) {
	home, _ := setupTempHome(t)

	writeFile(t, userConfigPath(home), `
llmConfig:
  name: user-llm
  provider: anthropic
  models:
    - claude-x
  maxoutputtokens: 4096
  reasoning: high
`)

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "user-llm", cfg.LlmConfig.Name)
	assert.Equal(t, "anthropic", cfg.LlmConfig.Provider)
	assert.Equal(t, []string{"claude-x"}, cfg.LlmConfig.Models)
	assert.Equal(t, 4096, cfg.LlmConfig.MaxOutputTokens)
	assert.Equal(t, models.ReasoningHigh, cfg.LlmConfig.Reasoning)

	// No local .harnesser dir -> DataDir must not be set.
	assert.Empty(t, cfg.DataDir)
}

func TestLoad_LocalConfigOnly(t *testing.T) {
	_, work := setupTempHome(t)

	writeFile(t, localConfigPath(work), `
llmConfig:
  name: local-llm
  provider: openai
  models:
    - gpt-x
  maxoutputtokens: 1024
  reasoning: low
`)

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "local-llm", cfg.LlmConfig.Name)
	assert.Equal(t, "openai", cfg.LlmConfig.Provider)
	assert.Equal(t, []string{"gpt-x"}, cfg.LlmConfig.Models)
	assert.Equal(t, 1024, cfg.LlmConfig.MaxOutputTokens)
	assert.Equal(t, models.ReasoningLow, cfg.LlmConfig.Reasoning)

	// Local .harnesser dir exists -> DataDir must point at it (absolute).
	wantDataDir, err := filepath.Abs(filepath.Join(".", DirName))
	require.NoError(t, err)
	assert.Equal(t, wantDataDir, cfg.DataDir)
}

func TestLoad_LocalOverridesUser(t *testing.T) {
	home, work := setupTempHome(t)

	writeFile(t, userConfigPath(home), `
llmConfig:
  name: user-llm
  provider: anthropic
  models:
    - claude-user
    - claude-user-2
  maxoutputtokens: 4096
  reasoning: high
`)

	writeFile(t, localConfigPath(work), `
llmConfig:
  name: local-llm
  maxoutputtokens: 512
`)

	cfg, err := Load()
	require.NoError(t, err)

	// Fields set in local config win.
	assert.Equal(t, "local-llm", cfg.LlmConfig.Name)
	assert.Equal(t, 512, cfg.LlmConfig.MaxOutputTokens)

	// Fields not set in local config keep the user config values.
	assert.Equal(t, "anthropic", cfg.LlmConfig.Provider)
	assert.Equal(t, []string{"claude-user", "claude-user-2"}, cfg.LlmConfig.Models)
	assert.Equal(t, models.ReasoningHigh, cfg.LlmConfig.Reasoning)

	// Local .harnesser dir exists -> DataDir points at the local dir.
	wantDataDir, err := filepath.Abs(filepath.Join(".", DirName))
	require.NoError(t, err)
	assert.Equal(t, wantDataDir, cfg.DataDir)
}

func TestLoad_DataDirNotSetWithoutLocalDir(t *testing.T) {
	home, _ := setupTempHome(t)

	// User config exists, but no local .harnesser dir.
	writeFile(t, userConfigPath(home), `
llmConfig:
  name: user-llm
`)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.DataDir)
}

func TestLoad_InvalidUserConfig(t *testing.T) {
	home, _ := setupTempHome(t)

	writeFile(t, userConfigPath(home), "llmConfig: [not: valid")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user config")
}

func TestLoad_InvalidLocalConfig(t *testing.T) {
	_, work := setupTempHome(t)

	writeFile(t, localConfigPath(work), "llmConfig: [not: valid")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "local config")
}

func TestWrite_Local(t *testing.T) {
	setupTempHome(t)

	cfg := Defaults()
	cfg.LlmConfig.Name = "written"
	cfg.DataDir = "/some/datadir" // must not be serialized (yaml:"-")

	require.NoError(t, Write(cfg, false))

	path := filepath.Join(".", DirName, ConfigFile)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "DataDir")
	assert.NotContains(t, string(data), "/some/datadir")

	// Round-trip.
	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "written", loaded.LlmConfig.Name)
	assert.Equal(t, cfg.LlmConfig.Provider, loaded.LlmConfig.Provider)
	assert.Equal(t, cfg.LlmConfig.Models, loaded.LlmConfig.Models)
	assert.Equal(t, cfg.LlmConfig.MaxOutputTokens, loaded.LlmConfig.MaxOutputTokens)
	assert.Equal(t, cfg.LlmConfig.Reasoning, loaded.LlmConfig.Reasoning)

	// Writing locally created the .harnesser dir, so a subsequent Load
	// should now set DataDir to it.
	wantDataDir, err := filepath.Abs(filepath.Join(".", DirName))
	require.NoError(t, err)
	assert.Equal(t, wantDataDir, loaded.DataDir)
}

func TestWrite_UserHome(t *testing.T) {
	home, _ := setupTempHome(t)

	cfg := Defaults()
	cfg.LlmConfig.Name = "user-written"

	require.NoError(t, Write(cfg, true))

	data, err := os.ReadFile(userConfigPath(home))
	require.NoError(t, err)
	assert.Contains(t, string(data), "user-written")

	// No local .harnesser dir was created by writing to the home dir.
	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "user-written", loaded.LlmConfig.Name)
	assert.Empty(t, loaded.DataDir)
}

func TestWrite_ThenLoad_LocalOverridesUser(t *testing.T) {
	setupTempHome(t)

	// Write user config with distinctive values.
	userCfg := Defaults()
	userCfg.LlmConfig.Name = "user-llm"
	userCfg.LlmConfig.Provider = "anthropic"
	userCfg.LlmConfig.MaxOutputTokens = 8192
	require.NoError(t, Write(userCfg, true))

	// Write local config overriding only some fields.
	localCfg := Defaults()
	localCfg.LlmConfig.Name = "local-llm"
	localCfg.LlmConfig.MaxOutputTokens = 256
	require.NoError(t, Write(localCfg, false))

	loaded, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "local-llm", loaded.LlmConfig.Name)
	assert.Equal(t, 256, loaded.LlmConfig.MaxOutputTokens)
	// Write serializes the whole config, so the local file's values (from
	// Defaults here) win for every field, not just the ones we changed.
	assert.Equal(t, localCfg.LlmConfig.Provider, loaded.LlmConfig.Provider)
	assert.Equal(t, localCfg.LlmConfig.Models, loaded.LlmConfig.Models)

	// DataDir points at local dir since it now exists.
	wantDataDir, err := filepath.Abs(filepath.Join(".", DirName))
	require.NoError(t, err)
	assert.Equal(t, wantDataDir, loaded.DataDir)
}
