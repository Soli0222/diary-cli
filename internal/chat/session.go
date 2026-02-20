package chat

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/soli0222/diary-cli/internal/claude"
)

type Options struct {
	ProfileSummary           string
	SummaryEvery             int
	MaxUnknownsBeforeConfirm int
	EmpathyStyle             string
}

func buildSystemPromptNormal(p1Count, p2Count, p3Count int, notes, profileSummary string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, `あなたはユーザーの日記作成を手伝うインタビュアーです。
ユーザーのMisskeyノート（SNS投稿）を元に、その日の出来事について質問し、言語化を促してください。

## ルール
- 質問は1回につき1つだけ
- 日本語で質問してください
- ユーザーの回答に寄り添いながら、次の質問を考えてください
- 質問だけを返してください。余計な前置きは不要です

## フェーズ
あなたは以下のフェーズに沿って質問を進めてください。

### フェーズ1: 事実確認（%d問程度）
ノートの時系列を見て、主要なトピックについて経緯・背景を質問する。
例: 「午前中に○○について投稿していましたが、これはどういう経緯でしたか？」

### フェーズ2: 深掘り（%d問程度）
フェーズ1の回答を受けて、感情や理由、内省を促す質問をする。
例: 「それに対してどう感じましたか？」「なぜそう思ったのですか？」

### フェーズ3: 締め（%d問程度）
一日の総括を促す。
例: 「今日一日を振り返って、一番印象に残ったことは？」

## ユーザーのノート
%s`, p1Count, p2Count, p3Count, notes)

	if profileSummary != "" {
		sb.WriteString("\n\n## ユーザープロファイル（過去セッションからの学習）\n")
		sb.WriteString(profileSummary)
	}

	return sb.String()
}

func buildSystemPromptFewNotes(p1Count, p2Count, p3Count int, notes, profileSummary string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, `あなたはユーザーの日記作成を手伝うインタビュアーです。
今日はSNS投稿が少ない日です。ノートに書かれていない活動も積極的に引き出してください。

## ルール
- 質問は1回につき1つだけ
- 日本語で質問してください
- ユーザーの回答に寄り添いながら、次の質問を考えてください
- 質問だけを返してください。余計な前置きは不要です

## 重要な方針
- ノートが少ないため、ノートの内容だけに頼らず、一日全体の過ごし方を広く聞いてください
- ノートが投稿されていない時間帯について、何をしていたか積極的に質問してください
- 仕事・作業の内容、食事、移動、休憩など日常的な活動も日記の材料になります
- 少ない情報から深い対話を引き出すことを意識してください

## フェーズ
あなたは以下のフェーズに沿って質問を進めてください。

### フェーズ1: 概要把握（%d問程度）
ノートの内容に軽く触れつつ、一日全体の流れを聞く。
例: 「今日は投稿が少なめですが、忙しい一日でしたか？どんな一日でしたか？」
例: 「夜に○○について投稿していましたが、日中はどのように過ごしていましたか？」

### フェーズ2: 深掘り（%d問程度）
フェーズ1の回答を掘り下げて、具体的なエピソードや感情を引き出す。
ノートに表れていない活動についても聞く。
例: 「仕事では具体的にどんなことに取り組んでいましたか？」
例: 「それについてどう感じましたか？」

### フェーズ3: 締め（%d問程度）
一日の総括を促す。
例: 「今日一日を振り返って、一番印象に残ったことは？」

## ユーザーのノート
%s`, p1Count, p2Count, p3Count, notes)

	if profileSummary != "" {
		sb.WriteString("\n\n## ユーザープロファイル（過去セッションからの学習）\n")
		sb.WriteString(profileSummary)
	}

	return sb.String()
}

const fewNotesThreshold = 10

// Session manages an interactive chat session.
type Session struct {
	client                   *claude.Client
	messages                 []claude.Message
	systemPrompt             string
	maxQuestions             int
	minQuestions             int
	questionNum              int
	fewNotes                 bool
	phase1End                int // questions [0, phase1End) are phase 1
	phase2End                int // questions [phase1End, phase2End) are phase 2
	summaryEvery             int
	maxUnknownsBeforeConfirm int
	empathyStyle             string
	state                    TurnState
}

// phaseBoundaries computes phase transition points based on maxQuestions.
// Normal mode uses a 3:3:2 ratio, few notes mode uses a 2:4:2 ratio.
// Returns (phase1End, phase2End).
func phaseBoundaries(maxQ int, fewNotes bool) (int, int) {
	var p1Weight, p12Weight int
	if fewNotes {
		// 2:4:2
		p1Weight = 2
		p12Weight = 6
	} else {
		// 3:3:2
		p1Weight = 3
		p12Weight = 6
	}
	const totalWeight = 8

	phase1End := maxQ * p1Weight / totalWeight
	phase2End := maxQ * p12Weight / totalWeight

	// Ensure each phase gets at least 1 question
	if phase1End < 1 {
		phase1End = 1
	}
	if phase2End <= phase1End {
		phase2End = phase1End + 1
	}
	if phase2End >= maxQ {
		phase2End = maxQ - 1
	}

	return phase1End, phase2End
}

// NewSession creates a new chat session with the given notes context.
func NewSession(client *claude.Client, formattedNotes string, noteCount, maxQ, minQ int) *Session {
	return NewSessionWithOptions(client, formattedNotes, noteCount, maxQ, minQ, Options{})
}

// NewSessionWithOptions creates a new session with profile-aware behavior.
func NewSessionWithOptions(client *claude.Client, formattedNotes string, noteCount, maxQ, minQ int, opts Options) *Session {
	fewNotes := noteCount < fewNotesThreshold
	p1End, p2End := phaseBoundaries(maxQ, fewNotes)
	p1Count := p1End
	p2Count := p2End - p1End
	p3Count := maxQ - p2End

	var prompt string
	if fewNotes {
		prompt = buildSystemPromptFewNotes(p1Count, p2Count, p3Count, formattedNotes, opts.ProfileSummary)
	} else {
		prompt = buildSystemPromptNormal(p1Count, p2Count, p3Count, formattedNotes, opts.ProfileSummary)
	}

	if opts.SummaryEvery <= 0 {
		opts.SummaryEvery = 2
	}
	if opts.MaxUnknownsBeforeConfirm <= 0 {
		opts.MaxUnknownsBeforeConfirm = 3
	}
	if opts.EmpathyStyle == "" {
		opts.EmpathyStyle = "balanced"
	}

	return &Session{
		client:                   client,
		systemPrompt:             prompt,
		maxQuestions:             maxQ,
		minQuestions:             minQ,
		fewNotes:                 fewNotes,
		phase1End:                p1End,
		phase2End:                p2End,
		summaryEvery:             opts.SummaryEvery,
		maxUnknownsBeforeConfirm: opts.MaxUnknownsBeforeConfirm,
		empathyStyle:             opts.EmpathyStyle,
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

		s.state.UpdateFromAnswer(answer)

		// Add assistant question and user answer to history
		s.messages = append(s.messages,
			claude.Message{Role: "assistant", Content: question},
			claude.Message{Role: "user", Content: answer},
		)
	}

	fmt.Println("\n--- 対話セッション終了 ---")

	return s.messages, nil
}

func (s *Session) getPhaseHint() string {
	if s.fewNotes {
		// ノートが少ない日: 概要把握を短く、深掘りに比重
		switch {
		case s.questionNum < s.phase1End:
			return "引き続きフェーズ1（概要把握）の質問をしてください。必要であればフェーズ2に移っても構いません。"
		case s.questionNum < s.phase2End:
			return "フェーズ2（深掘り）の質問をしてください。ノートに表れていない活動についても積極的に聞いてください。必要であればフェーズ3に移っても構いません。"
		default:
			return "フェーズ3（締め）の質問をしてください。"
		}
	}
	// 通常モード
	switch {
	case s.questionNum < s.phase1End:
		return "引き続きフェーズ1（事実確認）の質問をしてください。必要であればフェーズ2に移っても構いません。"
	case s.questionNum < s.phase2End:
		return "フェーズ2（深掘り）の質問をしてください。必要であればフェーズ3に移っても構いません。"
	default:
		return "フェーズ3（締め）の質問をしてください。"
	}
}

func (s *Session) shouldSummaryCheck() bool {
	return s.summaryEvery > 0 && s.questionNum > 0 && s.questionNum%s.summaryEvery == 0
}

func (s *Session) shouldConfirmUnknowns() bool {
	return s.maxUnknownsBeforeConfirm > 0 && s.state.Unknowns >= s.maxUnknownsBeforeConfirm
}

func (s *Session) getInteractionHint() string {
	var hints []string
	if s.shouldSummaryCheck() {
		hints = append(hints, summaryCheckHint())
	}
	if s.shouldConfirmUnknowns() {
		hints = append(hints, unknownsHint(s.state.Unknowns))
	}
	hints = append(hints, empathyHint(s.empathyStyle))
	return strings.Join(hints, " ")
}

func (s *Session) nextQuestion() (string, error) {
	// Build prompt for getting next question
	msgs := make([]claude.Message, len(s.messages))
	copy(msgs, s.messages)

	if len(msgs) == 0 {
		// First question
		firstPrompt := "上記のノートを元に、最初の質問をしてください。フェーズ1（事実確認）から始めてください。"
		if s.fewNotes {
			firstPrompt = "上記のノートを元に、最初の質問をしてください。フェーズ1（概要把握）から始めてください。"
		}
		msgs = append(msgs, claude.Message{
			Role:    "user",
			Content: firstPrompt,
		})
	} else {
		// Add instruction for next question
		phaseHint := s.getPhaseHint()
		interactionHint := s.getInteractionHint()
		msgs = append(msgs, claude.Message{
			Role:    "user",
			Content: fmt.Sprintf("次の質問をしてください。%s %s（%d問目/%d問中）", phaseHint, interactionHint, s.questionNum+1, s.maxQuestions),
		})
	}

	return s.client.Chat(s.systemPrompt, msgs)
}

// GetMessages returns the conversation messages.
func (s *Session) GetMessages() []claude.Message {
	return s.messages
}
