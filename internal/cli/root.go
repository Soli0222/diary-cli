package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	flagDate      string
	flagYesterday bool
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diary-cli",
		Short: "Misskeyノートを元にAIと対話しながら日記を生成するCLIツール",
	}

	cmd.PersistentFlags().StringVar(&flagDate, "date", "", "対象日 (YYYY-MM-DD)")
	cmd.PersistentFlags().BoolVar(&flagYesterday, "yesterday", false, "昨日の日記を作成")

	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newSummaryCmd())
	cmd.AddCommand(newPushCmd())

	return cmd
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// resolveDate determines the target date from flags.
func resolveDate() (time.Time, error) {
	now := time.Now()

	if flagDate != "" {
		t, err := time.ParseInLocation("2006-01-02", flagDate, now.Location())
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid date format (expected YYYY-MM-DD): %w", err)
		}
		return t, nil
	}

	if flagYesterday {
		return now.AddDate(0, 0, -1), nil
	}

	return now, nil
}
