package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/soli0222/diary-cli/internal/ai"
	"github.com/soli0222/diary-cli/internal/config"
	"github.com/soli0222/diary-cli/internal/discord"
	"github.com/soli0222/diary-cli/internal/generator"
	"github.com/soli0222/diary-cli/internal/misskey"
	"github.com/soli0222/diary-cli/internal/models"
	"github.com/soli0222/diary-cli/internal/preprocess"
)

const (
	outputMarkdown = "markdown"
	outputSummary  = "summary"
	outputJSON     = "json"
	outputNone     = "none"
)

var (
	flagOutput   string
	flagDiscord  bool
	flagProvider string

	loadConfig          = config.Load
	diaryWorkflowRunner = runDiaryWorkflow
	discordPoster       = postSummaryToDiscord
)

type diaryRunResult struct {
	TargetDate time.Time
	StartTime  time.Time
	EndTime    time.Time
	Title      string
	Summary    string
	Notes      []models.Note
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "ノート取得から日記ベース生成まで実行する",
		RunE:  runRun,
	}

	cmd.Flags().StringVarP(&flagOutput, "output", "o", outputMarkdown, "出力形式 (markdown, summary, json, none)")
	cmd.Flags().BoolVar(&flagDiscord, "discord", false, "Discord Webhookにも投稿する")
	cmd.Flags().StringVarP(&flagProvider, "provider", "p", "", "AIプロバイダ (claude, openai, gemini)")

	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()

	result, err := diaryWorkflowRunner(cmd.Context(), cfg, flagProvider, stderr)
	if err != nil {
		return err
	}

	if err := handleRunOutput(stdout, stderr, cfg, result, flagOutput); err != nil {
		return err
	}

	if flagDiscord {
		if err := discordPoster(cfg, result); err != nil {
			fmt.Fprintf(stderr, "Discord投稿に失敗しました: %v\n", err)
		} else {
			fmt.Fprintln(stderr, "Discordへ投稿しました")
		}
	}

	return nil
}

func runDiaryWorkflow(ctx context.Context, cfg *config.Config, providerName string, progress io.Writer) (*diaryRunResult, error) {
	loc, err := cfg.DiaryLocation()
	if err != nil {
		return nil, err
	}

	targetDate, err := resolveDate(loc)
	if err != nil {
		return nil, err
	}

	startTime, endTime := resolveDiaryWindow(targetDate)
	if progress != nil {
		fmt.Fprintf(progress, "%s の対象期間: %s 〜 %s\n", targetDate.Format("2006-01-02"), startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	}

	notes, err := fetchNotesForWindow(cfg, startTime, endTime)
	if err != nil {
		return nil, err
	}
	notes = preprocess.EnrichNotesWithSummaly(filterNotes(notes), preprocess.NewSummalyClientWithEndpoint(cfg.Summaly.Endpoint))

	if progress != nil {
		fmt.Fprintf(progress, "Misskeyから%d件のノートを取得しました\n", len(notes))
	}
	if len(notes) == 0 {
		return &diaryRunResult{
			TargetDate: targetDate,
			StartTime:  startTime,
			EndTime:    endTime,
			Title:      targetDate.Format("2006-01-02"),
			Summary:    "この日は対象期間内のノートがありませんでした。",
		}, nil
	}

	grouped := preprocess.GroupNotes(notes, loc)
	formattedNotes := preprocess.FormatGroupedNotes(grouped, loc)

	providerName = resolveProviderName(cfg, providerName)
	if providerName == "" {
		return nil, fmt.Errorf("ai.default_provider か --provider を指定してください")
	}

	var summary string
	provider, err := buildProviderFromConfig(ctx, providerName, cfg)
	if err != nil {
		return nil, err
	}
	summary, err = ai.GenerateSummary(ctx, provider, formattedNotes, targetDate)
	if err != nil {
		return nil, err
	}

	titleProvider, err := buildProviderFromConfig(ctx, providerName, cfg)
	if err != nil {
		return nil, err
	}
	title, err := ai.GenerateTitle(ctx, titleProvider, summary, targetDate)
	if err != nil {
		return nil, err
	}

	return &diaryRunResult{
		TargetDate: targetDate,
		StartTime:  startTime,
		EndTime:    endTime,
		Title:      title,
		Summary:    summary,
		Notes:      notes,
	}, nil
}

func handleRunOutput(stdout, status io.Writer, cfg *config.Config, result *diaryRunResult, output string) error {
	switch strings.ToLower(strings.TrimSpace(output)) {
	case "", outputMarkdown:
		if strings.TrimSpace(cfg.Diary.OutputDir) == "" {
			return fmt.Errorf("diary.output_dir is required for markdown output")
		}
		fileTime := time.Date(
			result.TargetDate.Year(),
			result.TargetDate.Month(),
			result.TargetDate.Day(),
			result.StartTime.In(result.TargetDate.Location()).Hour(),
			result.StartTime.In(result.TargetDate.Location()).Minute(),
			0,
			0,
			result.TargetDate.Location(),
		)
		markdown := generator.BuildMarkdown(fileTime, cfg.Diary.Author, result.Title, result.Summary)
		outputPath, err := saveDiary(cfg.Diary.OutputDir, result.TargetDate, markdown)
		if err != nil {
			return err
		}
		if status != nil {
			fmt.Fprintf(status, "保存しました: %s\n", outputPath)
		}
		return nil
	case outputSummary:
		fmt.Fprintln(stdout, generator.BuildSummaryText(result.TargetDate, len(result.Notes), result.Title, result.Summary))
		return nil
	case outputJSON:
		payload := generator.BuildJSONOutput(
			result.TargetDate,
			result.StartTime,
			result.EndTime,
			result.Title,
			result.Summary,
			result.Notes,
		)
		encoded, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode json output: %w", err)
		}
		fmt.Fprintln(stdout, string(encoded))
		return nil
	case outputNone:
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func resolveProviderName(cfg *config.Config, flagValue string) string {
	if v := strings.ToLower(strings.TrimSpace(flagValue)); v != "" {
		return v
	}
	return strings.ToLower(strings.TrimSpace(cfg.AI.DefaultProvider))
}

func buildProviderFromConfig(ctx context.Context, name string, cfg *config.Config) (ai.AIProvider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "claude":
		if strings.TrimSpace(cfg.AI.Claude.APIKey) == "" {
			return nil, fmt.Errorf("ai.claude.api_key is required")
		}
		return ai.NewClaudeProvider(cfg.AI.Claude.APIKey, cfg.AI.Claude.Model), nil
	case "openai":
		if strings.TrimSpace(cfg.AI.OpenAI.APIKey) == "" {
			return nil, fmt.Errorf("ai.openai.api_key is required")
		}
		return ai.NewOpenAIProvider(cfg.AI.OpenAI.APIKey, cfg.AI.OpenAI.Model), nil
	case "gemini":
		if strings.TrimSpace(cfg.AI.Gemini.APIKey) == "" {
			return nil, fmt.Errorf("ai.gemini.api_key is required")
		}
		return ai.NewGeminiProvider(ctx, cfg.AI.Gemini.APIKey, cfg.AI.Gemini.Model)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", name)
	}
}

func fetchNotesForWindow(cfg *config.Config, startTime, endTime time.Time) ([]models.Note, error) {
	if strings.TrimSpace(cfg.Misskey.InstanceURL) == "" {
		return nil, fmt.Errorf("misskey.instance_url is required")
	}
	if strings.TrimSpace(cfg.Misskey.Token) == "" {
		return nil, fmt.Errorf("misskey.token is required")
	}

	client := misskey.NewClient(cfg.Misskey.InstanceURL, cfg.Misskey.Token)
	me, err := client.GetMe()
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	notes, err := client.GetNotesForTimeRange(me.ID, startTime, endTime, true)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch notes: %w", err)
	}

	return notes, nil
}

func filterNotes(notes []models.Note) []models.Note {
	filtered := make([]models.Note, 0, len(notes))
	for _, note := range notes {
		if note.ReplyID != nil {
			continue
		}
		if note.ChannelID != nil {
			continue
		}
		if !note.IsOriginalNote() {
			continue
		}
		filtered = append(filtered, note)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})
	return filtered
}

func postSummaryToDiscord(cfg *config.Config, result *diaryRunResult) error {
	if strings.TrimSpace(cfg.Discord.WebhookURL) == "" {
		return fmt.Errorf("discord.webhook_url is required when --discord is set")
	}

	client := discord.NewClient(cfg.Discord.WebhookURL)
	return client.PostSummary(result.TargetDate.Format("2006-01-02"), len(result.Notes), result.Title, result.Summary)
}

func saveDiary(outputDir string, date time.Time, content string) (string, error) {
	yearDir := filepath.Join(outputDir, date.Format("2006"))
	if err := os.MkdirAll(yearDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	filename := date.Format("0102") + ".md"
	outputPath := filepath.Join(yearDir, filename)

	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return outputPath, nil
}
