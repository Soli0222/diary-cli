package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const SummarySystemPrompt = `あなたはMisskeyノートをもとに、その日の出来事を時系列で整理するアシスタントです。

ルール:
- 日本語で書く
- 時間帯ごとに見出しをつけて整理する
- 事実ベースで簡潔にまとめる
- ユーザーの気分や関心の流れが分かるようにする
- 絵文字は使わない
- Markdown本文のみを返す`

const titleSystemPrompt = `あなたは日記タイトルを付けるアシスタントです。

ルール:
- 日本語
- 10〜30文字程度
- その日の象徴的な話題や感触が伝わる短いタイトル
- タイトル本文のみを返す`

func BuildSummaryPrompt(date time.Time, formattedNotes string) string {
	return fmt.Sprintf(
		"対象日: %s\n\n以下はMisskeyノートを時間帯ごとに整理したものです。1日のサマリーを作成してください。\n\n%s",
		date.Format("2006-01-02"),
		formattedNotes,
	)
}

func BuildTitlePrompt(date time.Time, summary string) string {
	return fmt.Sprintf(
		"対象日: %s\n\n以下のサマリーにタイトルを付けてください。\n\n%s",
		date.Format("2006-01-02"),
		summary,
	)
}

func GenerateSummary(ctx context.Context, provider AIProvider, formattedNotes string, date time.Time) (string, error) {
	text, err := provider.Summarize(ctx, BuildSummaryPrompt(date, formattedNotes), SummarySystemPrompt)
	if err != nil {
		return "", fmt.Errorf("%s summary failed: %w", provider.Name(), err)
	}
	return strings.TrimSpace(text), nil
}

func GenerateTitle(ctx context.Context, provider AIProvider, summary string, date time.Time) (string, error) {
	text, err := provider.GenerateTitle(ctx, BuildTitlePrompt(date, summary))
	if err != nil {
		return "", fmt.Errorf("%s title generation failed: %w", provider.Name(), err)
	}
	return strings.TrimSpace(text), nil
}
