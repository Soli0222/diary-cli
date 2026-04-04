package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Misskey MisskeyConfig `mapstructure:"misskey"`
	AI      AIConfig      `mapstructure:"ai"`
	Diary   DiaryConfig   `mapstructure:"diary"`
	Summaly SummalyConfig `mapstructure:"summaly"`
	Discord DiscordConfig `mapstructure:"discord"`
}

type MisskeyConfig struct {
	InstanceURL string `mapstructure:"instance_url"`
	Token       string `mapstructure:"token"`
}

type AIConfig struct {
	DefaultProvider string           `mapstructure:"default_provider"`
	Claude          AIProviderConfig `mapstructure:"claude"`
	OpenAI          AIProviderConfig `mapstructure:"openai"`
	Gemini          AIProviderConfig `mapstructure:"gemini"`
}

type AIProviderConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

type DiaryConfig struct {
	OutputDir string `mapstructure:"output_dir"`
	Author    string `mapstructure:"author"`
	Editor    string `mapstructure:"editor"`
	Timezone  string `mapstructure:"timezone"`
}

type SummalyConfig struct {
	Endpoint string `mapstructure:"endpoint"`
}

type DiscordConfig struct {
	WebhookURL string `mapstructure:"webhook_url"`
}

func DefaultConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "diary-cli"), nil
}

func DefaultConfigPath() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func Load() (*Config, error) {
	cfgPath, err := DefaultConfigPath()
	if err != nil {
		return nil, err
	}

	v := viper.New()
	v.SetConfigFile(cfgPath)
	v.SetConfigType("yaml")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)
	bindEnv(v)

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.Diary.Editor == "" {
		cfg.Diary.Editor = EnvOrDefault("EDITOR", "vim")
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("ai.default_provider", "claude")
	v.SetDefault("ai.claude.model", "claude-sonnet-4-6")
	v.SetDefault("ai.openai.model", "gpt-5.4-mini")
	v.SetDefault("ai.gemini.model", "gemini-3.1-flash-preview")
	v.SetDefault("diary.output_dir", "./diary")
	v.SetDefault("diary.author", EnvOrDefault("USER", "Soli"))
	v.SetDefault("diary.editor", EnvOrDefault("EDITOR", "vim"))
	v.SetDefault("diary.timezone", "Asia/Tokyo")
	v.SetDefault("summaly.endpoint", "")
	v.SetDefault("discord.webhook_url", "")
}

func bindEnv(v *viper.Viper) {
	_ = v.BindEnv("misskey.instance_url", "MISSKEY_INSTANCE_URL")
	_ = v.BindEnv("misskey.token", "MISSKEY_TOKEN")
	_ = v.BindEnv("ai.claude.api_key", "ANTHROPIC_API_KEY")
	_ = v.BindEnv("ai.openai.api_key", "OPENAI_API_KEY")
	_ = v.BindEnv("ai.gemini.api_key", "GOOGLE_API_KEY")
	_ = v.BindEnv("discord.webhook_url", "DISCORD_WEBHOOK_URL")
}

func EnvOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func (c *Config) DiaryLocation() (*time.Location, error) {
	name := strings.TrimSpace(c.Diary.Timezone)
	if name == "" {
		name = "Asia/Tokyo"
	}

	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("invalid diary.timezone %q: %w", name, err)
	}
	return loc, nil
}
