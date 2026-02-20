package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/soli0222/diary-cli/internal/chat"
	"github.com/soli0222/diary-cli/internal/claude"
	"github.com/soli0222/diary-cli/internal/config"
	"github.com/soli0222/diary-cli/internal/generator"
	"github.com/soli0222/diary-cli/internal/misskey"
	"github.com/soli0222/diary-cli/internal/models"
	"github.com/soli0222/diary-cli/internal/preprocess"
	"github.com/soli0222/diary-cli/internal/profile"
)

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "対話を通じて日記を作成する",
		RunE:  runRun,
	}
}

func runRun(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	date, err := resolveDate()
	if err != nil {
		return err
	}

	dateStr := date.Format("2006-01-02")
	fmt.Printf("📝 %s の日記を作成します\n", dateStr)

	// 1. Fetch notes
	notes, err := fetchNotes(cfg, date)
	if err != nil {
		return err
	}
	notes = preprocess.EnrichNotesWithSummaly(notes, preprocess.NewSummalyClientWithEndpoint(cfg.Summaly.Endpoint))
	fmt.Printf("📥 Misskeyから%d件のノートを取得しました\n", len(notes))

	if len(notes) == 0 {
		fmt.Println("⚠️  ノートが見つかりませんでした")
		return nil
	}

	// 2. Preprocess
	groups := preprocess.GroupNotes(notes)
	formattedNotes := preprocess.FormatGroupedNotes(groups)

	// 3. Interactive chat session
	claudeClient := claude.NewClient(cfg.Claude.APIKey, cfg.Claude.Model)
	prof, profilePath := loadUserProfile(cfg)

	session := chat.NewSessionWithOptions(
		claudeClient,
		formattedNotes,
		len(notes),
		cfg.Chat.MaxQuestions,
		cfg.Chat.MinQuestions,
		chat.Options{
			ProfileSummary:           profile.SummaryForPrompt(prof, 8),
			SummaryEvery:             cfg.Chat.SummaryEvery,
			MaxUnknownsBeforeConfirm: cfg.Chat.MaxUnknownsBeforeConfirm,
			EmpathyStyle:             cfg.Chat.EmpathyStyle,
			PendingHypotheses:        toPendingHypotheses(prof.PendingConfirmations, 5),
		},
	)

	conversation, err := session.Run()
	if err != nil {
		return fmt.Errorf("chat session failed: %w", err)
	}

	if cfg.Chat.ProfileEnabled {
		if err := updateUserProfile(claudeClient, prof, profilePath, conversation, session.GetConfirmationOutcomes(), date); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  プロファイル更新をスキップしました: %v\n", err)
		}
	}

	// 4. Generate diary
	fmt.Println("📄 日記を生成中...")
	gen := generator.NewGenerator(claudeClient)

	diaryBody, err := gen.GenerateDiary(conversation)
	if err != nil {
		return fmt.Errorf("diary generation failed: %w", err)
	}

	// 5. Generate summary
	fmt.Println("📄 サマリーを生成中...")
	summary, err := gen.GenerateSummary(preprocess.FormatAllNotes(notes), date)
	if err != nil {
		return fmt.Errorf("summary generation failed: %w", err)
	}

	// 6. Generate title
	title, err := gen.GenerateTitle(diaryBody)
	if err != nil {
		return fmt.Errorf("title generation failed: %w", err)
	}

	// 7. Build and save markdown
	now := time.Now()
	diaryTime := time.Date(date.Year(), date.Month(), date.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
	markdown := generator.BuildMarkdown(diaryTime, cfg.Diary.Author, title, diaryBody, summary)

	outputPath, err := saveDiary(cfg.Diary.OutputDir, date, markdown)
	if err != nil {
		return err
	}
	fmt.Printf("✅ 日記を保存しました: %s\n", outputPath)

	// 8. Open in editor
	fmt.Print("エディタで開きますか？ (y/N) ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer == "y" || answer == "yes" {
			editorCmd := exec.Command(cfg.Diary.Editor, outputPath)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr
			if err := editorCmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  エディタの起動に失敗しました: %v\n", err)
			}
		}
	}

	return nil
}

func loadUserProfile(cfg *config.Config) (*profile.UserProfile, string) {
	path := cfg.Chat.ProfilePath
	if !cfg.Chat.ProfileEnabled {
		return profile.NewEmpty(), path
	}

	prof, err := profile.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  プロファイル読み込みに失敗しました。空プロファイルで続行します: %v\n", err)
	}
	if prof == nil {
		prof = profile.NewEmpty()
	}
	return prof, path
}

func updateUserProfile(client *claude.Client, current *profile.UserProfile, path string, conversation []claude.Message, outcomes []chat.ConfirmationOutcome, date time.Time) error {
	updates, err := profile.ExtractUpdates(client, conversation, date, current)
	if err != nil {
		return fmt.Errorf("learning extraction failed: %w", err)
	}

	merged := profile.Merge(current, updates, date)
	merged = profile.ApplyConfirmations(merged, toProfileOutcomes(outcomes), date)
	if err := profile.Save(path, merged); err != nil {
		return fmt.Errorf("profile save failed: %w", err)
	}
	return nil
}

func toPendingHypotheses(items []profile.PendingConfirmation, limit int) []chat.PendingHypothesis {
	if len(items) == 0 || limit <= 0 {
		return nil
	}
	if len(items) < limit {
		limit = len(items)
	}
	out := make([]chat.PendingHypothesis, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, chat.PendingHypothesis{
			Category: items[i].Category,
			Value:    items[i].Value,
		})
	}
	return out
}

func toProfileOutcomes(items []chat.ConfirmationOutcome) []profile.ConfirmationOutcome {
	if len(items) == 0 {
		return nil
	}
	out := make([]profile.ConfirmationOutcome, 0, len(items))
	for _, item := range items {
		out = append(out, profile.ConfirmationOutcome{
			QuestionNum: item.QuestionNum,
			Category:    item.Category,
			Value:       item.Value,
			Question:    item.Question,
			Answer:      item.Answer,
			Confirmed:   item.Confirmed,
			Denied:      item.Denied,
		})
	}
	return out
}

func fetchNotes(cfg *config.Config, date time.Time) ([]models.Note, error) {
	client := misskey.NewClient(cfg.Misskey.InstanceURL, cfg.Misskey.Token)

	me, err := client.GetMe()
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	notes, err := client.GetNotesForDay(me.ID, date, false)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch notes: %w", err)
	}

	return notes, nil
}

func saveDiary(outputDir string, date time.Time, content string) (string, error) {
	yearDir := filepath.Join(outputDir, date.Format("2006"))
	if err := os.MkdirAll(yearDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	filename := date.Format("0102") + ".md"
	outputPath := filepath.Join(yearDir, filename)

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return outputPath, nil
}
