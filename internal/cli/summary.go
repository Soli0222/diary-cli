package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/soli0222/diary-cli/internal/generator"
)

var (
	summaryFlagDiscord  bool
	summaryFlagProvider string
)

func newSummaryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "ノート取得とAI要約のみ実行する",
		RunE:  runSummary,
	}

	cmd.Flags().BoolVar(&summaryFlagDiscord, "discord", false, "Discord Webhookにも投稿する")
	cmd.Flags().StringVarP(&summaryFlagProvider, "provider", "p", "", "AIプロバイダ (claude, openai, gemini)")

	return cmd
}

func runSummary(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()

	result, err := diaryWorkflowRunner(cmd.Context(), cfg, summaryFlagProvider, stderr)
	if err != nil {
		return err
	}

	if err := writeLine(stdout, generator.BuildSummaryText(result.TargetDate, len(result.Notes), result.Title, result.Summary)); err != nil {
		return err
	}

	if summaryFlagDiscord {
		if err := discordPoster(cfg, result); err != nil {
			if writeErr := writeLine(stderr, fmt.Sprintf("Discord投稿に失敗しました: %v", err)); writeErr != nil {
				return writeErr
			}
		} else {
			if err := writeLine(stderr, "Discordへ投稿しました"); err != nil {
				return err
			}
		}
	}

	return nil
}
