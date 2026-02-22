package cli

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/soli0222/diary-cli/internal/metrics"
)

func newStatsCmd() *cobra.Command {
	var days int
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "対話品質メトリクスを表示する",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStats(days)
		},
	}
	cmd.Flags().IntVar(&days, "days", 7, "表示する日数")
	return cmd
}

func runStats(days int) error {
	if days <= 0 {
		days = 7
	}

	now := time.Now()
	since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -days+1)
	items, err := metrics.LoadSince("", since)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Printf("📊 直近%d日間のメトリクスはありません\n", days)
		return nil
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Date == items[j].Date {
			return items[i].RecordedAt < items[j].RecordedAt
		}
		return items[i].Date < items[j].Date
	})

	var (
		runs           int
		totalQuestions int
		totalSummary   int
		totalConfTry   int
		totalConfOK    int
		totalAns       float64
		totalDupRate   float64
	)

	for _, item := range items {
		runs++
		totalQuestions += item.QuestionsTotal
		totalSummary += item.SummaryCheckTurns
		totalConfTry += item.ConfirmationAttempts
		totalConfOK += item.ConfirmationConfirmed
		totalAns += item.AvgAnswerLength
		totalDupRate += item.DuplicateQuestionRate
	}

	fmt.Printf("📊 直近%d日（%s〜%s）\n", days, since.Format("2006-01-02"), now.Format("2006-01-02"))
	fmt.Printf("実行回数: %d\n", runs)
	fmt.Printf("平均質問数: %.2f\n", float64(totalQuestions)/float64(runs))
	fmt.Printf("要約確認率: %.1f%%\n", safeRate(totalSummary, totalQuestions)*100)
	fmt.Printf("確認成功率: %.1f%%\n", safeRate(totalConfOK, totalConfTry)*100)
	fmt.Printf("平均回答文字数: %.1f\n", totalAns/float64(runs))
	fmt.Printf("平均重複質問率: %.1f%%\n", (totalDupRate/float64(runs))*100)

	fmt.Println("\n日別:")
	daily := groupByDate(items)
	dates := make([]string, 0, len(daily))
	for d := range daily {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	for _, d := range dates {
		agg := daily[d]
		fmt.Printf("- %s: run=%d, q=%.1f, summary=%.1f%%, conf=%.1f%%, ans=%.1f, dup=%.1f%%\n",
			d,
			agg.runs,
			float64(agg.totalQuestions)/float64(agg.runs),
			safeRate(agg.totalSummary, agg.totalQuestions)*100,
			safeRate(agg.totalConfOK, agg.totalConfTry)*100,
			agg.totalAns/float64(agg.runs),
			(agg.totalDupRate/float64(agg.runs))*100,
		)
	}

	return nil
}

type dailyAgg struct {
	runs           int
	totalQuestions int
	totalSummary   int
	totalConfTry   int
	totalConfOK    int
	totalAns       float64
	totalDupRate   float64
}

func groupByDate(items []metrics.RunMetrics) map[string]dailyAgg {
	m := make(map[string]dailyAgg)
	for _, item := range items {
		agg := m[item.Date]
		agg.runs++
		agg.totalQuestions += item.QuestionsTotal
		agg.totalSummary += item.SummaryCheckTurns
		agg.totalConfTry += item.ConfirmationAttempts
		agg.totalConfOK += item.ConfirmationConfirmed
		agg.totalAns += item.AvgAnswerLength
		agg.totalDupRate += item.DuplicateQuestionRate
		m[item.Date] = agg
	}
	return m
}

func safeRate(num, den int) float64 {
	if den <= 0 {
		return 0
	}
	return float64(num) / float64(den)
}
