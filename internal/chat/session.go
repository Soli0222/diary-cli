package chat

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/soli0222/diary-cli/internal/claude"
)

const systemPrompt = `あなたはユーザーの日記作成を手伝うインタビュアーです。
ユーザーのMisskeyノート（SNS投稿）を元に、その日の出来事について質問し、言語化を促してください。

## ルール
- 質問は1回につき1つだけ
- 日本語で質問してください
- ユーザーの回答に寄り添いながら、次の質問を考えてください
- 質問だけを返してください。余計な前置きは不要です

## フェーズ
あなたは以下のフェーズに沿って質問を進めてください。

### フェーズ1: 事実確認（最初の1〜3問）
ノートの時系列を見て、主要なトピックについて経緯・背景を質問する。
例: 「午前中に○○について投稿していましたが、これはどういう経緯でしたか？」

### フェーズ2: 深掘り（次の1〜3問）
フェーズ1の回答を受けて、感情や理由、内省を促す質問をする。
例: 「それに対してどう感じましたか？」「なぜそう思ったのですか？」

### フェーズ3: 締め（最後の1〜2問）
一日の総括を促す。
例: 「今日一日を振り返って、一番印象に残ったことは？」

## ユーザーのノート
%s`

// Session manages an interactive chat session.
type Session struct {
	client       *claude.Client
	messages     []claude.Message
	systemPrompt string
	maxQuestions int
	minQuestions int
	questionNum  int
}

// NewSession creates a new chat session with the given notes context.
func NewSession(client *claude.Client, formattedNotes string, maxQ, minQ int) *Session {
	return &Session{
		client:       client,
		systemPrompt: fmt.Sprintf(systemPrompt, formattedNotes),
		maxQuestions: maxQ,
		minQuestions: minQ,
	}
}

// Run executes the interactive chat session and returns the full conversation history.
func (s *Session) Run() ([]claude.Message, error) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("\n--- 対話セッション開始 ---")
	fmt.Printf("（/done で終了、最大%d問）\n", s.maxQuestions)

	for s.questionNum < s.maxQuestions {
		// Get next question from Claude
		question, err := s.nextQuestion()
		if err != nil {
			return nil, fmt.Errorf("failed to get question: %w", err)
		}

		s.questionNum++
		fmt.Printf("\n🤖 %s\n\n> ", question)

		// Read user input
		if !scanner.Scan() {
			break
		}
		answer := strings.TrimSpace(scanner.Text())

		if answer == "/done" {
			if s.questionNum <= s.minQuestions {
				fmt.Printf("（最低%d問まで回答してください。あと%d問です）\n> ", s.minQuestions, s.minQuestions-s.questionNum+1)
				if !scanner.Scan() {
					break
				}
				answer = strings.TrimSpace(scanner.Text())
				if answer == "/done" {
					break
				}
			} else {
				break
			}
		}

		// Add assistant question and user answer to history
		s.messages = append(s.messages,
			claude.Message{Role: "assistant", Content: question},
			claude.Message{Role: "user", Content: answer},
		)
	}

	fmt.Println("\n--- 対話セッション終了 ---")

	return s.messages, nil
}

func (s *Session) nextQuestion() (string, error) {
	// Build prompt for getting next question
	msgs := make([]claude.Message, len(s.messages))
	copy(msgs, s.messages)

	if len(msgs) == 0 {
		// First question
		msgs = append(msgs, claude.Message{
			Role:    "user",
			Content: "上記のノートを元に、最初の質問をしてください。フェーズ1（事実確認）から始めてください。",
		})
	} else {
		// Add instruction for next question
		phaseHint := ""
		switch {
		case s.questionNum < 3:
			phaseHint = "引き続きフェーズ1（事実確認）の質問をしてください。必要であればフェーズ2に移っても構いません。"
		case s.questionNum < 6:
			phaseHint = "フェーズ2（深掘り）の質問をしてください。必要であればフェーズ3に移っても構いません。"
		default:
			phaseHint = "フェーズ3（締め）の質問をしてください。"
		}
		msgs = append(msgs, claude.Message{
			Role:    "user",
			Content: fmt.Sprintf("次の質問をしてください。%s（%d問目/%d問中）", phaseHint, s.questionNum+1, s.maxQuestions),
		})
	}

	return s.client.Chat(s.systemPrompt, msgs)
}

// GetMessages returns the conversation messages.
func (s *Session) GetMessages() []claude.Message {
	return s.messages
}
