package generator

import (
	"fmt"
	"strings"
	"time"

	"github.com/soli0222/diary-cli/internal/claude"
)

const diarySystemPrompt = `あなたは日記のゴーストライターです。
ユーザーとの対話履歴を元に、ユーザー本人が書いたかのような日記を生成してください。

## ルール
- カジュアルな「だ・である」調、もしくは話し言葉寄りの文体
- ユーザー本人が書いたように見える一人称視点
- 対話で引き出された内容を自然な文章にまとめる
- 見出しは使わず、段落で区切る
- 絵文字の使用は禁止
- Markdown形式で出力する（本文のみ、frontmatterは不要）`

const summarySystemPrompt = `あなたはMisskeyのノートを時系列順にサマリーするアシスタントです。

## ルール
- 日本語で回答してください
- 時間帯ごとにグルーピングして整理してください
- 重要なトピックを抽出してまとめてください
- 絵文字の使用は禁止
- Markdown形式で出力する（本文のみ、frontmatterは不要）`

const titleSystemPrompt = `日記の内容を元に、その日を表す短いタイトルを生成してください。
タイトルのみを返してください。余計な説明は不要です。
10〜30文字程度で、その日の象徴的な出来事や気分を表すタイトルにしてください。`

// Generator handles diary and summary generation via Claude API.
type Generator struct {
	client *claude.Client
}

// NewGenerator creates a new Generator.
func NewGenerator(client *claude.Client) *Generator {
	return &Generator{client: client}
}

// GenerateDiary generates diary body text from conversation history.
func (g *Generator) GenerateDiary(conversation []claude.Message) (string, error) {
	// Build conversation summary for the prompt
	var sb strings.Builder
	sb.WriteString("以下はユーザーとの対話履歴です。この内容を元に日記を生成してください。\n\n")
	for _, msg := range conversation {
		role := "ユーザー"
		if msg.Role == "assistant" {
			role = "インタビュアー"
		}
		fmt.Fprintf(&sb, "%s: %s\n\n", role, msg.Content)
	}

	msgs := []claude.Message{
		{Role: "user", Content: sb.String()},
	}

	return g.client.Chat(diarySystemPrompt, msgs)
}

// GenerateSummary generates a time-series summary from notes.
func (g *Generator) GenerateSummary(formattedNotes string, date time.Time) (string, error) {
	dateStr := date.Format("2006-01-02")
	prompt := fmt.Sprintf("以下は%sのMisskeyへの投稿一覧です。この日の活動を時系列順にサマリーしてください。\n\n%s", dateStr, formattedNotes)

	msgs := []claude.Message{
		{Role: "user", Content: prompt},
	}

	return g.client.Chat(summarySystemPrompt, msgs)
}

// GenerateTitle generates a diary title from the diary body.
func (g *Generator) GenerateTitle(diaryBody string) (string, error) {
	msgs := []claude.Message{
		{Role: "user", Content: "以下の日記のタイトルを考えてください。\n\n" + diaryBody},
	}

	title, err := g.client.Chat(titleSystemPrompt, msgs)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(title), nil
}

// BuildMarkdown assembles the final markdown output.
func BuildMarkdown(date time.Time, author, title, diaryBody, summary string) string {
	dateStr := date.Format("2006-01-02")
	timeStr := date.Format("2006-01-02T15:04")

	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "title: %s\n", dateStr)
	fmt.Fprintf(&sb, "author: %s\n", author)
	sb.WriteString("layout: post\n")
	sb.WriteString(fmt.Sprintf("date: %s\n", timeStr))
	sb.WriteString("category: 日記\n")
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	sb.WriteString(diaryBody)
	sb.WriteString("\n\n# Misskeyサマリー\n\n")
	sb.WriteString(summary)
	sb.WriteString("\n")

	return sb.String()
}
