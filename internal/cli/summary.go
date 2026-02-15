package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/soli0222/diary-cli/internal/claude"
	"github.com/soli0222/diary-cli/internal/config"
	"github.com/soli0222/diary-cli/internal/generator"
	"github.com/soli0222/diary-cli/internal/preprocess"
)

func newSummaryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "summary",
		Short: "サマリーのみ生成する（対話スキップ）",
		RunE:  runSummary,
	}
}

func runSummary(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	date, err := resolveDate()
	if err != nil {
		return err
	}

	dateStr := date.Format("2006-01-02")
	fmt.Printf("📝 %s のサマリーを生成します\n", dateStr)

	// 1. Fetch notes
	notes, err := fetchNotes(cfg, date)
	if err != nil {
		return err
	}
	fmt.Printf("📥 Misskeyから%d件のノートを取得しました\n", len(notes))

	if len(notes) == 0 {
		fmt.Println("⚠️  ノートが見つかりませんでした")
		return nil
	}

	// 2. Generate summary
	fmt.Println("📄 サマリーを生成中...")
	claudeClient := claude.NewClient(cfg.Claude.APIKey, cfg.Claude.Model)
	gen := generator.NewGenerator(claudeClient)

	formattedNotes := preprocess.FormatAllNotes(notes)
	summary, err := gen.GenerateSummary(formattedNotes, date)
	if err != nil {
		return fmt.Errorf("summary generation failed: %w", err)
	}

	// 3. Build markdown (summary only, no diary body)
	now := time.Now()
	diaryTime := time.Date(date.Year(), date.Month(), date.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
	markdown := generator.BuildMarkdown(diaryTime, cfg.Diary.Author, dateStr+"のサマリー", "", summary)

	outputPath, err := saveDiary(cfg.Diary.OutputDir, date, markdown)
	if err != nil {
		return err
	}
	fmt.Printf("✅ サマリーを保存しました: %s\n", outputPath)

	return nil
}
