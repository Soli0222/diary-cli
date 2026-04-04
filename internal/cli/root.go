package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

const diaryDayStartHour = 5

var (
	flagDate      string
	flagYesterday bool

	Version = "dev"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diary-cli",
		Short: "Misskeyノートを要約して日記ベースを生成するCLIツール",
	}

	cmd.PersistentFlags().StringVarP(&flagDate, "date", "d", "", "対象日 (YYYY-MM-DD, 明示指定時は05:00補正なし)")
	cmd.PersistentFlags().BoolVarP(&flagYesterday, "yesterday", "y", false, "昨日の日記を作成")

	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newSummaryCmd())
	cmd.AddCommand(newPushCmd())
	cmd.AddCommand(newVersionCmd())

	return cmd
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func resolveDate(loc *time.Location) (time.Time, error) {
	return resolveTargetDate(time.Now(), flagDate, flagYesterday, loc)
}

func resolveTargetDate(now time.Time, dateFlag string, yesterdayFlag bool, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = now.Location()
	}

	if dateFlag != "" {
		t, err := time.ParseInLocation("2006-01-02", dateFlag, loc)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid date format (expected YYYY-MM-DD): %w", err)
		}
		return normalizeToLocalMidnight(t, loc), nil
	}

	base := now.In(loc)
	if base.Hour() < diaryDayStartHour {
		base = base.AddDate(0, 0, -1)
	}
	if yesterdayFlag {
		base = base.AddDate(0, 0, -1)
	}

	return normalizeToLocalMidnight(base, loc), nil
}

func resolveDiaryWindow(targetDate time.Time) (time.Time, time.Time) {
	start := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), diaryDayStartHour, 0, 0, 0, targetDate.Location())
	return start, start.Add(24 * time.Hour)
}

func normalizeToLocalMidnight(t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = t.Location()
	}
	inLoc := t.In(loc)
	return time.Date(inLoc.Year(), inLoc.Month(), inLoc.Day(), 0, 0, 0, 0, loc)
}
