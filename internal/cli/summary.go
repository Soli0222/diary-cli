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

	fmt.Fprintln(stdout, generator.BuildSummaryText(result.TargetDate, len(result.Notes), result.Title, result.Summary))

	if summaryFlagDiscord {
		if err := discordPoster(cfg, result); err != nil {
			fmt.Fprintf(stderr, "Discord投稿に失敗しました: %v\n", err)
		} else {
			fmt.Fprintln(stderr, "Discordへ投稿しました")
		}
	}

	return nil
}
