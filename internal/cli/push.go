package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/soli0222/diary-cli/internal/config"
	"github.com/soli0222/diary-cli/internal/git"
)

func newPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push",
		Short: "生成済みファイルをgit commit & push",
		RunE:  runPush,
	}
}

func runPush(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	date, err := resolveDate()
	if err != nil {
		return err
	}

	dateStr := date.Format("2006-01-02")
	filename := date.Format("0102") + ".md"
	filePath := filepath.Join(date.Format("2006"), filename)

	fmt.Printf("📤 %s の日記をpushします\n", dateStr)

	if err := git.CommitAndPush(cfg.Diary.OutputDir, filePath, dateStr); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}

	fmt.Println("✅ pushしました")
	return nil
}
