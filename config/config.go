package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/c00/harnesser/models"
	"go.yaml.in/yaml/v4"
)

const (
	DirName    = ".harnesser"
	PromptsDir = "prompts"
	ToolsDir   = "tools"
	HistoryDir = "history"
	ConfigFile = "config.yaml"
)

type Config struct {
	DataDir   string           `yaml:"-"`
	Landlock  LandlockSettings `yaml:"landlock"`
	LlmConfig models.LlmConfig `yaml:"llmConfig"`
}

type LandlockSettings struct {
	// Landlock the application to the current working dir and main configuration
	Active bool `yaml:"active"`
	// Include folders in PATH as read only (for executing tools)
	IncludePath bool `yaml:"includePath"`
	// Extra directories to grant readonly access
	ExtraRODirs []string `yaml:"extraRODirs"`
	// Extra directories to grant write access
	ExtraRWDirs []string `yaml:"extraRWDirs"`
}

// Load confg from disk or return defaults
func Load() (Config, error) {
	cfg := Defaults()

	// Try load the users config
	userHome, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("no user home dir: %w", err)
	}
	userConfigFilename := filepath.Join(userHome, DirName, ConfigFile)

	data, err := os.ReadFile(userConfigFilename)
	if err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("cannot read user config file: %w", err)
	} else {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("cannot parse user config file: %w", err)
		}
	}

	localConfigFilename := filepath.Join("./", DirName, ConfigFile)

	data, err = os.ReadFile(localConfigFilename)
	if err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("cannot read local config file: %w", err)
	} else {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("cannot parse local config file: %w", err)
		}
	}

	// Set datadir
	// If local .harnesser folder exists, use that, otherwise use home folder.
	if _, err := os.Stat(filepath.Join("./", DirName)); err == nil {
		cfg.DataDir, _ = filepath.Abs(filepath.Join("./", DirName))
	}

	return cfg, nil
}

func Write(cfg Config, userHome bool) error {
	filename := filepath.Join("./", DirName, ConfigFile)
	path := DirName

	if userHome {
		dir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("no user home dir: %w", err)
		}
		filename = filepath.Join(dir, DirName, ConfigFile)
		path = filepath.Join(dir, DirName)
	}

	err := os.MkdirAll(path, 0o755)
	if err != nil {
		return fmt.Errorf("cannot create .harnesser dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("cannot encode config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0o644); err != nil {
		return fmt.Errorf("cannot write config file %q: %w", filename, err)
	}

	return nil
}

func Defaults() Config {
	return Config{
		Landlock: LandlockSettings{
			Active:      true,
			IncludePath: true,
		},
		LlmConfig: models.LlmConfig{
			Name:     "openrouter",
			Provider: "openrouter",
			Models: []string{
				"deepseek/deepseek-v4-flash-vision-exp",
				"deepseek/deepseek-v4-flash-0731",
			},
			MaxOutputTokens: 2048,
			Reasoning:       models.ReasoningMedium,
		},
	}
}
