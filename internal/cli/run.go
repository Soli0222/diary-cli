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
	"github.com/soli0222/diary-cli/internal/metrics"
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

	executionNow := time.Now()
	targetDate, err := resolveTargetDate(executionNow, flagDate, flagYesterday)
	if err != nil {
		return err
	}

	dateStr := targetDate.Format("2006-01-02")
	fmt.Printf("📝 %s の日記を作成します\n", dateStr)

	prof, profilePath := loadUserProfile(cfg)
	beforeStable := len(prof.StableFacts)
	beforePending := len(prof.PendingConfirmations)
	beforeConflicts := len(prof.Conflicts)

	// 1. Fetch notes
	notes, err := fetchNotesForRun(cfg, targetDate, executionNow, flagDate != "", prof)
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
	sessionMetrics := session.GetMetrics()

	updatedProf := prof
	if cfg.Chat.ProfileEnabled {
		updatedProf, err = updateUserProfile(claudeClient, prof, profilePath, conversation, session.GetConfirmationOutcomes(), targetDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  プロファイル更新をスキップしました: %v\n", err)
			updatedProf = prof
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
	summary, err := gen.GenerateSummary(preprocess.FormatAllNotes(notes), targetDate)
	if err != nil {
		return fmt.Errorf("summary generation failed: %w", err)
	}

	// 6. Generate title
	title, err := gen.GenerateTitle(diaryBody)
	if err != nil {
		return fmt.Errorf("title generation failed: %w", err)
	}

	// 7. Build and save markdown
	diaryTime := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), executionNow.Hour(), executionNow.Minute(), 0, 0, executionNow.Location())
	markdown := generator.BuildMarkdown(diaryTime, cfg.Diary.Author, title, diaryBody, summary)

	outputPath, err := saveDiary(cfg.Diary.OutputDir, targetDate, markdown)
	if err != nil {
		return err
	}
	fmt.Printf("✅ 日記を保存しました: %s\n", outputPath)
	recordAndPrintRunMetrics(
		targetDate,
		sessionMetrics,
		beforeStable,
		len(updatedProf.StableFacts),
		beforePending,
		len(updatedProf.PendingConfirmations),
		beforeConflicts,
		len(updatedProf.Conflicts),
	)

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

func resolveCollectionWindowForRun(now, targetDate time.Time, isExplicitDate bool, prof *profile.UserProfile) (start, end time.Time, ok bool, reason string) {
	dayStart := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	if isExplicitDate {
		return dayStart, dayEnd, false, "explicit_date"
	}
	if prof == nil {
		return dayStart, dayEnd, false, "nil_profile"
	}
	if prof.UpdatedAt == "" {
		return dayStart, dayEnd, false, "empty_updated_at"
	}

	parsed, err := time.Parse(time.RFC3339, prof.UpdatedAt)
	if err != nil {
		return dayStart, dayEnd, false, "invalid_updated_at"
	}
	if !parsed.Before(now) {
		return dayStart, dayEnd, false, "future_or_equal_updated_at"
	}

	return parsed, now, true, ""
}

func fetchNotesForRun(cfg *config.Config, targetDate, executionNow time.Time, isExplicitDate bool, prof *profile.UserProfile) ([]models.Note, error) {
	client := misskey.NewClient(cfg.Misskey.InstanceURL, cfg.Misskey.Token)

	me, err := client.GetMe()
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	start, end, useProfileWindow, reason := resolveCollectionWindowForRun(executionNow, targetDate, isExplicitDate, prof)
	if useProfileWindow {
		fmt.Printf("🕒 ノート収集範囲: %s 〜 %s (profile.updated_at)\n", start.Format(time.RFC3339), end.Format(time.RFC3339))
		notes, err := client.GetNotesForTimeRange(me.ID, start, end, false)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch notes: %w", err)
		}
		return notes, nil
	}

	switch reason {
	case "explicit_date":
		fmt.Println("🕒 ノート収集範囲: 対象日1日分 (--date 指定)")
	case "nil_profile", "empty_updated_at":
		// 初回実行や profile 無効時の通常系フォールバック。警告は出さない。
	case "invalid_updated_at", "future_or_equal_updated_at":
		fmt.Fprintf(os.Stderr, "⚠️  profile.updated_at を使えないため日単位取得にフォールバックします (%s)\n", reason)
	}

	notes, err := client.GetNotesForDay(me.ID, targetDate, false)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch notes: %w", err)
	}
	return notes, nil
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

func updateUserProfile(client *claude.Client, current *profile.UserProfile, path string, conversation []claude.Message, outcomes []chat.ConfirmationOutcome, date time.Time) (*profile.UserProfile, error) {
	updates, err := profile.ExtractUpdates(client, conversation, date, current)
	if err != nil {
		return current, fmt.Errorf("learning extraction failed: %w", err)
	}

	merged := profile.Merge(current, updates, date)
	merged = profile.ApplyConfirmations(merged, toProfileOutcomes(outcomes), date)
	if err := profile.Save(path, merged); err != nil {
		return current, fmt.Errorf("profile save failed: %w", err)
	}
	return merged, nil
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
			Uncertain:   item.Uncertain,
			Method:      item.Method,
			Reason:      item.Reason,
		})
	}
	return out
}

func recordAndPrintRunMetrics(date time.Time, m chat.Metrics, beforeStable, afterStable, beforePending, afterPending, beforeConflicts, afterConflicts int) {
	item := metrics.RunMetrics{
		Date:                  date.Format("2006-01-02"),
		QuestionsTotal:        m.QuestionsTotal,
		SummaryCheckTurns:     m.SummaryCheckTurns,
		StructuredTurns:       m.StructuredTurns,
		FallbackTurns:         m.FallbackTurns,
		ConfirmationAttempts:  m.ConfirmationAttempts,
		ConfirmationConfirmed: m.ConfirmationConfirmed,
		ConfirmationDenied:    m.ConfirmationDenied,
		ConfirmationUncertain: m.ConfirmationUncertain,
		AvgAnswerLength:       m.AvgAnswerLength,
		DuplicateQuestionRate: m.DuplicateQuestionRate,
		StableFactsBefore:     beforeStable,
		StableFactsAfter:      afterStable,
		PendingBefore:         beforePending,
		PendingAfter:          afterPending,
		ConflictsBefore:       beforeConflicts,
		ConflictsAfter:        afterConflicts,
	}

	if err := metrics.Append("", item); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  実行メトリクスの保存に失敗しました: %v\n", err)
	}

	fmt.Println("\n--- Interview Metrics ---")
	fmt.Printf("質問数: %d（要約確認 %d）\n", m.QuestionsTotal, m.SummaryCheckTurns)
	fmt.Printf("構造化ターン: %d / フォールバック: %d\n", m.StructuredTurns, m.FallbackTurns)
	fmt.Printf("確認結果: 試行 %d / 確定 %d / 否定 %d / 不確実 %d\n", m.ConfirmationAttempts, m.ConfirmationConfirmed, m.ConfirmationDenied, m.ConfirmationUncertain)
	fmt.Printf("平均回答文字数: %.1f\n", m.AvgAnswerLength)
	fmt.Printf("重複質問率: %.1f%%\n", m.DuplicateQuestionRate*100)
	fmt.Printf("プロファイル変化: stable %d→%d, pending %d→%d, conflicts %d→%d\n", beforeStable, afterStable, beforePending, afterPending, beforeConflicts, afterConflicts)
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
