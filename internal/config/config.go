package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Misskey MisskeyConfig `mapstructure:"misskey"`
	Claude  ClaudeConfig  `mapstructure:"claude"`
	Diary   DiaryConfig   `mapstructure:"diary"`
	Chat    ChatConfig    `mapstructure:"chat"`
	Summaly SummalyConfig `mapstructure:"summaly"`
}

type MisskeyConfig struct {
	InstanceURL string `mapstructure:"instance_url"`
	Token       string `mapstructure:"token"`
}

type ClaudeConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

type DiaryConfig struct {
	OutputDir string `mapstructure:"output_dir"`
	Author    string `mapstructure:"author"`
	Editor    string `mapstructure:"editor"`
}

type ChatConfig struct {
	MaxQuestions             int    `mapstructure:"max_questions"`
	MinQuestions             int    `mapstructure:"min_questions"`
	SummaryEvery             int    `mapstructure:"summary_every"`
	MaxUnknownsBeforeConfirm int    `mapstructure:"max_unknowns_before_confirm"`
	EmpathyStyle             string `mapstructure:"empathy_style"`
	ProfileEnabled           bool   `mapstructure:"profile_enabled"`
	ProfilePath              string `mapstructure:"profile_path"`
}

type SummalyConfig struct {
	Endpoint string `mapstructure:"endpoint"`
}

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".config", "diary-cli")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	// Defaults
	viper.SetDefault("claude.model", "claude-sonnet-4-6")
	viper.SetDefault("diary.author", "Soli")
	viper.SetDefault("diary.editor", "")
	viper.SetDefault("chat.max_questions", 8)
	viper.SetDefault("chat.min_questions", 3)
	viper.SetDefault("chat.summary_every", 2)
	viper.SetDefault("chat.max_unknowns_before_confirm", 3)
	viper.SetDefault("chat.empathy_style", "balanced")
	viper.SetDefault("chat.profile_enabled", true)
	viper.SetDefault("chat.profile_path", "")
	viper.SetDefault("summaly.endpoint", "")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.Misskey.InstanceURL == "" {
		return nil, fmt.Errorf("misskey.instance_url is required")
	}
	if cfg.Misskey.Token == "" {
		return nil, fmt.Errorf("misskey.token is required")
	}
	if cfg.Claude.APIKey == "" {
		return nil, fmt.Errorf("claude.api_key is required")
	}
	if cfg.Diary.OutputDir == "" {
		return nil, fmt.Errorf("diary.output_dir is required")
	}

	// Resolve editor: config > $EDITOR > vi
	if cfg.Diary.Editor == "" {
		cfg.Diary.Editor = os.Getenv("EDITOR")
		if cfg.Diary.Editor == "" {
			cfg.Diary.Editor = "vi"
		}
	}

	return &cfg, nil
}
