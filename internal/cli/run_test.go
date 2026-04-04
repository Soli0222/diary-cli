package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/soli0222/diary-cli/internal/config"
	"github.com/soli0222/diary-cli/internal/models"
)

func TestFilterNotes(t *testing.T) {
	makeText := func(v string) *string { return &v }
	makeID := func(v string) *string { return &v }
	base := time.Date(2026, 2, 23, 5, 0, 0, 0, time.UTC)

	notes := []models.Note{
		{
			ID:        "keep-late",
			CreatedAt: base.Add(2 * time.Hour),
			Text:      makeText("kept"),
		},
		{
			ID:        "reply",
			CreatedAt: base.Add(time.Hour),
			Text:      makeText("reply"),
			ReplyID:   makeID("r1"),
		},
		{
			ID:        "channel",
			CreatedAt: base.Add(30 * time.Minute),
			Text:      makeText("channel"),
			ChannelID: makeID("c1"),
		},
		{
			ID:        "renote",
			CreatedAt: base.Add(45 * time.Minute),
			RenoteID:  makeID("rn1"),
		},
		{
			ID:        "keep-early",
			CreatedAt: base,
			Text:      makeText("kept early"),
		},
	}

	got := filterNotes(notes)

	if len(got) != 2 {
		t.Fatalf("len(filterNotes()) = %d, want 2", len(got))
	}
	if got[0].ID != "keep-early" || got[1].ID != "keep-late" {
		t.Fatalf("unexpected order: %#v", got)
	}
}

func TestResolveProviderName(t *testing.T) {
	cfg := &config.Config{}
	cfg.AI.DefaultProvider = "Gemini"

	if got := resolveProviderName(cfg, " OpenAI "); got != "openai" {
		t.Fatalf("resolveProviderName(flag) = %q", got)
	}
	if got := resolveProviderName(cfg, ""); got != "gemini" {
		t.Fatalf("resolveProviderName(config) = %q", got)
	}
}

func TestBuildProviderFromConfigRequiresAPIKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Config{}

	tests := []struct {
		name     string
		provider string
		wantErr  string
	}{
		{name: "claude", provider: "claude", wantErr: "ai.claude.api_key is required"},
		{name: "openai", provider: "openai", wantErr: "ai.openai.api_key is required"},
		{name: "gemini", provider: "gemini", wantErr: "ai.gemini.api_key is required"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := buildProviderFromConfig(ctx, tt.provider, cfg)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("err = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestHandleRunOutputRejectsUnsupportedOutput(t *testing.T) {
	cfg := &config.Config{}
	result := &diaryRunResult{
		TargetDate: time.Date(2026, 2, 23, 0, 0, 0, 0, time.UTC),
		StartTime:  time.Date(2026, 2, 23, 5, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2026, 2, 24, 5, 0, 0, 0, time.UTC),
		Title:      "title",
		Summary:    "summary",
	}

	err := handleRunOutput(&bytes.Buffer{}, &bytes.Buffer{}, cfg, result, "yaml")
	if err == nil || !strings.Contains(err.Error(), "unsupported output format: yaml") {
		t.Fatalf("err = %v", err)
	}
}

func TestHandleRunOutputMarkdownWritesStatusOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &config.Config{}
	cfg.Diary.OutputDir = dir
	cfg.Diary.Author = "soli"

	result := &diaryRunResult{
		TargetDate: time.Date(2026, 4, 4, 0, 0, 0, 0, time.FixedZone("Asia/Tokyo", 9*60*60)),
		StartTime:  time.Date(2026, 4, 4, 5, 0, 0, 0, time.FixedZone("Asia/Tokyo", 9*60*60)),
		EndTime:    time.Date(2026, 4, 5, 5, 0, 0, 0, time.FixedZone("Asia/Tokyo", 9*60*60)),
		Title:      "title",
		Summary:    "summary",
	}

	var stdout, status bytes.Buffer
	if err := handleRunOutput(&stdout, &status, cfg, result, outputMarkdown); err != nil {
		t.Fatalf("handleRunOutput() error = %v", err)
	}

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(status.String(), "保存しました:") {
		t.Fatalf("status = %q, want save message", status.String())
	}
}

func TestRunRunKeepsSummaryOnDiscordFailure(t *testing.T) {
	originalLoadConfig := loadConfig
	originalWorkflowRunner := diaryWorkflowRunner
	originalDiscordPoster := discordPoster
	originalFlagOutput := flagOutput
	originalFlagDiscord := flagDiscord
	originalFlagProvider := flagProvider
	defer func() {
		loadConfig = originalLoadConfig
		diaryWorkflowRunner = originalWorkflowRunner
		discordPoster = originalDiscordPoster
		flagOutput = originalFlagOutput
		flagDiscord = originalFlagDiscord
		flagProvider = originalFlagProvider
	}()

	loadConfig = func() (*config.Config, error) {
		return &config.Config{}, nil
	}
	diaryWorkflowRunner = func(ctx context.Context, cfg *config.Config, providerName string, progress io.Writer) (*diaryRunResult, error) {
		fmt.Fprintln(progress, "progress")
		return &diaryRunResult{
			TargetDate: time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC),
			Title:      "タイトル",
			Summary:    "本文",
			Notes:      []models.Note{{ID: "1"}},
		}, nil
	}
	discordPoster = func(cfg *config.Config, result *diaryRunResult) error {
		return errors.New("boom")
	}
	flagOutput = outputSummary
	flagDiscord = true
	flagProvider = "claude"

	var stdout, stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := runRun(cmd, nil); err != nil {
		t.Fatalf("runRun() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "2026-04-04 のサマリー") {
		t.Fatalf("stdout = %q, want summary output", stdout.String())
	}
	if !strings.Contains(stderr.String(), "progress") {
		t.Fatalf("stderr = %q, want progress output", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Discord投稿に失敗しました: boom") {
		t.Fatalf("stderr = %q, want discord warning", stderr.String())
	}
}

func TestRunSummaryKeepsOutputOnDiscordFailure(t *testing.T) {
	originalLoadConfig := loadConfig
	originalWorkflowRunner := diaryWorkflowRunner
	originalDiscordPoster := discordPoster
	originalSummaryFlagDiscord := summaryFlagDiscord
	originalSummaryFlagProvider := summaryFlagProvider
	defer func() {
		loadConfig = originalLoadConfig
		diaryWorkflowRunner = originalWorkflowRunner
		discordPoster = originalDiscordPoster
		summaryFlagDiscord = originalSummaryFlagDiscord
		summaryFlagProvider = originalSummaryFlagProvider
	}()

	loadConfig = func() (*config.Config, error) {
		return &config.Config{}, nil
	}
	diaryWorkflowRunner = func(ctx context.Context, cfg *config.Config, providerName string, progress io.Writer) (*diaryRunResult, error) {
		fmt.Fprintln(progress, "summary progress")
		return &diaryRunResult{
			TargetDate: time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC),
			Title:      "タイトル",
			Summary:    "本文",
			Notes:      []models.Note{{ID: "1"}, {ID: "2"}},
		}, nil
	}
	discordPoster = func(cfg *config.Config, result *diaryRunResult) error {
		return errors.New("boom")
	}
	summaryFlagDiscord = true
	summaryFlagProvider = "claude"

	var stdout, stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := runSummary(cmd, nil); err != nil {
		t.Fatalf("runSummary() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "ノート数: 2") {
		t.Fatalf("stdout = %q, want summary output", stdout.String())
	}
	if !strings.Contains(stderr.String(), "summary progress") {
		t.Fatalf("stderr = %q, want progress output", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Discord投稿に失敗しました: boom") {
		t.Fatalf("stderr = %q, want discord warning", stderr.String())
	}
}
