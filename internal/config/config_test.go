package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestSetDefaults(t *testing.T) {
	t.Setenv("USER", "TestUser")
	t.Setenv("EDITOR", "helix")

	v := viper.New()
	setDefaults(v)

	checks := map[string]string{
		"ai.default_provider": "claude",
		"ai.claude.model":     "claude-sonnet-4-6",
		"ai.openai.model":     "gpt-5.4-mini",
		"ai.gemini.model":     "gemini-3.1-flash-preview",
		"diary.output_dir":    "./diary",
		"diary.author":        "TestUser",
		"diary.editor":        "helix",
		"diary.timezone":      "Asia/Tokyo",
		"summaly.endpoint":    "",
		"discord.webhook_url": "",
	}

	for key, want := range checks {
		if got := v.GetString(key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MISSKEY_INSTANCE_URL", "https://env.example")
	t.Setenv("MISSKEY_TOKEN", "env-token")
	t.Setenv("ANTHROPIC_API_KEY", "env-anthropic")
	t.Setenv("OPENAI_API_KEY", "env-openai")
	t.Setenv("GOOGLE_API_KEY", "env-google")
	t.Setenv("DISCORD_WEBHOOK_URL", "https://discord.example/env")

	configDir := filepath.Join(home, ".config", "diary-cli")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	content := []byte(`misskey:
  instance_url: https://file.example
  token: file-token
ai:
  default_provider: openai
  claude:
    api_key: file-anthropic
  openai:
    api_key: file-openai
  gemini:
    api_key: file-google
discord:
  webhook_url: https://discord.example/file
`)
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Misskey.InstanceURL != "https://env.example" {
		t.Fatalf("Misskey.InstanceURL = %q", cfg.Misskey.InstanceURL)
	}
	if cfg.Misskey.Token != "env-token" {
		t.Fatalf("Misskey.Token = %q", cfg.Misskey.Token)
	}
	if cfg.AI.Claude.APIKey != "env-anthropic" {
		t.Fatalf("AI.Claude.APIKey = %q", cfg.AI.Claude.APIKey)
	}
	if cfg.AI.OpenAI.APIKey != "env-openai" {
		t.Fatalf("AI.OpenAI.APIKey = %q", cfg.AI.OpenAI.APIKey)
	}
	if cfg.AI.Gemini.APIKey != "env-google" {
		t.Fatalf("AI.Gemini.APIKey = %q", cfg.AI.Gemini.APIKey)
	}
	if cfg.Discord.WebhookURL != "https://discord.example/env" {
		t.Fatalf("Discord.WebhookURL = %q", cfg.Discord.WebhookURL)
	}
	if cfg.AI.DefaultProvider != "openai" {
		t.Fatalf("AI.DefaultProvider = %q", cfg.AI.DefaultProvider)
	}
}

func TestLoadFallsBackWhenConfigFileMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USER", "FallbackUser")
	t.Setenv("EDITOR", "nvim")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AI.DefaultProvider != "claude" {
		t.Fatalf("AI.DefaultProvider = %q", cfg.AI.DefaultProvider)
	}
	if cfg.AI.Claude.Model != "claude-sonnet-4-6" {
		t.Fatalf("AI.Claude.Model = %q", cfg.AI.Claude.Model)
	}
	if cfg.AI.OpenAI.Model != "gpt-5.4-mini" {
		t.Fatalf("AI.OpenAI.Model = %q", cfg.AI.OpenAI.Model)
	}
	if cfg.AI.Gemini.Model != "gemini-3.1-flash-preview" {
		t.Fatalf("AI.Gemini.Model = %q", cfg.AI.Gemini.Model)
	}
	if cfg.Diary.OutputDir != "./diary" {
		t.Fatalf("Diary.OutputDir = %q", cfg.Diary.OutputDir)
	}
	if cfg.Diary.Author != "FallbackUser" {
		t.Fatalf("Diary.Author = %q", cfg.Diary.Author)
	}
	if cfg.Diary.Editor != "nvim" {
		t.Fatalf("Diary.Editor = %q", cfg.Diary.Editor)
	}
	if cfg.Diary.Timezone != "Asia/Tokyo" {
		t.Fatalf("Diary.Timezone = %q", cfg.Diary.Timezone)
	}
}

func TestDiaryLocation(t *testing.T) {
	cfg := &Config{}

	loc, err := cfg.DiaryLocation()
	if err != nil {
		t.Fatalf("DiaryLocation() error = %v", err)
	}
	if loc.String() != "Asia/Tokyo" {
		t.Fatalf("DiaryLocation() = %q", loc)
	}
}

func TestDiaryLocationInvalid(t *testing.T) {
	cfg := &Config{}
	cfg.Diary.Timezone = "Mars/Base"

	if _, err := cfg.DiaryLocation(); err == nil {
		t.Fatal("DiaryLocation() error = nil, want invalid timezone error")
	}
}
