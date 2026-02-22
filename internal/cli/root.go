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

	// Version is set at build time via ldflags.
	Version = "dev"
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
	cmd.AddCommand(newStatsCmd())
	cmd.AddCommand(newVersionCmd())

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
	return resolveTargetDate(time.Now(), flagDate, flagYesterday)
}

func resolveTargetDate(now time.Time, dateFlag string, yesterdayFlag bool) (time.Time, error) {
	loc := now.Location()

	if dateFlag != "" {
		t, err := time.ParseInLocation("2006-01-02", dateFlag, loc)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid date format (expected YYYY-MM-DD): %w", err)
		}
		return normalizeToLocalMidnight(t, loc), nil
	}

	if yesterdayFlag {
		return normalizeToLocalMidnight(now.AddDate(0, 0, -1), loc), nil
	}

	return normalizeToLocalMidnight(now, loc), nil
}

func normalizeToLocalMidnight(t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = t.Location()
	}
	return time.Date(t.In(loc).Year(), t.In(loc).Month(), t.In(loc).Day(), 0, 0, 0, 0, loc)
}
