package chat

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/soli0222/diary-cli/internal/claude"
)

type Options struct {
	ProfileSummary           string
	SummaryEvery             int
	MaxUnknownsBeforeConfirm int
	EmpathyStyle             string
	PendingHypotheses        []PendingHypothesis
}

type PendingHypothesis struct {
	Category string
	Value    string
}

type ConfirmationOutcome struct {
	QuestionNum int
	Category    string
	Value       string
	Question    string
	Answer      string
	Confirmed   bool
	Denied      bool
	Uncertain   bool
	Method      string
	Reason      string
}

type Metrics struct {
	QuestionsTotal        int
	SummaryCheckTurns     int
	StructuredTurns       int
	FallbackTurns         int
	ConfirmationAttempts  int
	ConfirmationConfirmed int
	ConfirmationDenied    int
	ConfirmationUncertain int
	AvgAnswerLength       float64
	DuplicateQuestionRate float64
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
	pendingHypotheses        []PendingHypothesis
	activeConfirmation       *PendingHypothesis
	confirmationOutcomes     []ConfirmationOutcome
	structuredTurns          int
	fallbackTurns            int
	summaryCheckTurns        int
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
		pendingHypotheses:        append([]PendingHypothesis(nil), opts.PendingHypotheses...),
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
		s.recordConfirmationOutcome(question, answer)

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
	return s.summaryEvery > 0 &&
		s.questionNum > 0 &&
		s.questionNum%s.summaryEvery == 0 &&
		!s.recentSummaryCheck()
}

func (s *Session) shouldConfirmUnknowns() bool {
	return s.maxUnknownsBeforeConfirm > 0 && s.state.Unknowns >= s.maxUnknownsBeforeConfirm
}

func (s *Session) getInteractionHint() string {
	var hints []string
	target := s.pickPendingHypothesis()
	if target != nil {
		s.activeConfirmation = target
		hints = append(hints, pendingConfirmationHint(*target))
	} else {
		s.activeConfirmation = nil
	}
	if s.shouldSummaryCheck() {
		hints = append(hints, summaryCheckHint())
	}
	if s.shouldConfirmUnknowns() {
		hints = append(hints, unknownsHint(s.state.Unknowns))
	}
	hints = append(hints, questionQualityHint())
	hints = append(hints, empathyHint(s.empathyStyle))
	return strings.Join(hints, " ")
}

func (s *Session) recentSummaryCheck() bool {
	for i := len(s.messages) - 1; i >= 0; i-- {
		if s.messages[i].Role != "assistant" {
			continue
		}
		q := s.messages[i].Content
		return strings.Contains(q, "理解で合っていますか") ||
			strings.Contains(q, "という理解で") ||
			strings.Contains(q, "合っていますか")
	}
	return false
}

func (s *Session) pickPendingHypothesis() *PendingHypothesis {
	if len(s.pendingHypotheses) == 0 {
		return nil
	}
	if !s.shouldSummaryCheck() && !s.shouldConfirmUnknowns() {
		return nil
	}

	target := s.pendingHypotheses[0]
	return &target
}

func (s *Session) recordConfirmationOutcome(question, answer string) {
	if s.activeConfirmation == nil {
		return
	}

	verdict := s.classifyConfirmation(question, answer, *s.activeConfirmation)
	s.confirmationOutcomes = append(s.confirmationOutcomes, ConfirmationOutcome{
		QuestionNum: s.questionNum,
		Category:    s.activeConfirmation.Category,
		Value:       s.activeConfirmation.Value,
		Question:    question,
		Answer:      answer,
		Confirmed:   verdict.Confirmed,
		Denied:      verdict.Denied,
		Uncertain:   verdict.Uncertain,
		Method:      verdict.Method,
		Reason:      verdict.Reason,
	})

	if verdict.Confirmed || verdict.Denied {
		s.removePendingHypothesis(*s.activeConfirmation)
	}
	s.activeConfirmation = nil
}

type confirmationVerdict struct {
	Confirmed bool
	Denied    bool
	Uncertain bool
	Method    string
	Reason    string
}

func classifyConfirmationAnswer(answer string) confirmationVerdict {
	a := strings.TrimSpace(strings.ToLower(answer))
	if a == "" {
		return confirmationVerdict{Uncertain: true, Method: "rule", Reason: "empty answer"}
	}

	negativeTokens := []string{"いいえ", "違う", "ちがう", "違います", "not", "no", "そんなことない"}
	for _, token := range negativeTokens {
		if strings.Contains(a, token) {
			return confirmationVerdict{Denied: true, Method: "rule", Reason: "negative token matched: " + token}
		}
	}

	positiveTokens := []string{"はい", "そうです", "その通り", "あってます", "合ってます", "yes", "yep"}
	for _, token := range positiveTokens {
		if strings.Contains(a, token) {
			return confirmationVerdict{Confirmed: true, Method: "rule", Reason: "positive token matched: " + token}
		}
	}

	return confirmationVerdict{Uncertain: true, Method: "rule", Reason: "no explicit signal"}
}

func (s *Session) classifyConfirmation(question, answer string, target PendingHypothesis) confirmationVerdict {
	rule := classifyConfirmationAnswer(answer)
	if rule.Confirmed || rule.Denied {
		return rule
	}
	if s.client == nil {
		return rule
	}

	judge, err := s.judgeConfirmationWithLLM(question, answer, target)
	if err != nil {
		return rule
	}
	return judge
}

func (s *Session) judgeConfirmationWithLLM(question, answer string, target PendingHypothesis) (confirmationVerdict, error) {
	system := `あなたは確認回答の判定器です。JSONのみ返してください。
出力:
{"result":"confirmed|denied|uncertain","reason":"短い理由"}
`
	userPrompt := fmt.Sprintf("確認対象: [%s] %s\n質問: %s\n回答: %s", target.Category, target.Value, question, answer)
	raw, err := s.client.Chat(system, []claude.Message{{Role: "user", Content: userPrompt}})
	if err != nil {
		return confirmationVerdict{}, err
	}
	resp, err := parseConfirmationJudgeResponse(raw)
	if err != nil {
		return confirmationVerdict{}, err
	}
	return resp, nil
}

func (s *Session) removePendingHypothesis(target PendingHypothesis) {
	if len(s.pendingHypotheses) == 0 {
		return
	}
	dst := s.pendingHypotheses[:0]
	for _, h := range s.pendingHypotheses {
		if strings.EqualFold(strings.TrimSpace(h.Category), strings.TrimSpace(target.Category)) &&
			strings.EqualFold(strings.TrimSpace(h.Value), strings.TrimSpace(target.Value)) {
			continue
		}
		dst = append(dst, h)
	}
	s.pendingHypotheses = dst
}

func (s *Session) nextQuestion() (string, error) {
	// Build prompt for getting next question
	msgs := make([]claude.Message, len(s.messages))
	copy(msgs, s.messages)

	if len(msgs) == 0 {
		// First question
		firstPrompt := "上記のノートを元に、最初の質問をしてください。フェーズ1（事実確認）から始めてください。\n" + turnSchemaInstruction()
		if s.fewNotes {
			firstPrompt = "上記のノートを元に、最初の質問をしてください。フェーズ1（概要把握）から始めてください。\n" + turnSchemaInstruction()
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
			Content: fmt.Sprintf("次の質問をしてください。%s %s（%d問目/%d問中）\n%s", phaseHint, interactionHint, s.questionNum+1, s.maxQuestions, turnSchemaInstruction()),
		})
	}

	raw, err := s.client.Chat(s.systemPrompt, msgs)
	if err != nil {
		return "", err
	}
	turn, err := parseTurnResponse(raw)
	if err == nil {
		s.structuredTurns++
		if turn.SummaryCheck {
			s.summaryCheckTurns++
		}
		return turn.Question, nil
	}
	s.fallbackTurns++
	q := fallbackQuestion(raw)
	if strings.Contains(q, "つまり") {
		s.summaryCheckTurns++
	}
	return q, nil
}

// GetMessages returns the conversation messages.
func (s *Session) GetMessages() []claude.Message {
	return s.messages
}

func (s *Session) GetConfirmationOutcomes() []ConfirmationOutcome {
	out := make([]ConfirmationOutcome, len(s.confirmationOutcomes))
	copy(out, s.confirmationOutcomes)
	return out
}

func (s *Session) GetMetrics() Metrics {
	m := Metrics{
		QuestionsTotal:        s.questionNum,
		SummaryCheckTurns:     s.summaryCheckTurns,
		StructuredTurns:       s.structuredTurns,
		FallbackTurns:         s.fallbackTurns,
		ConfirmationAttempts:  len(s.confirmationOutcomes),
		ConfirmationConfirmed: 0,
		ConfirmationDenied:    0,
		ConfirmationUncertain: 0,
	}

	var answerChars int
	questionSet := map[string]int{}
	questionCount := 0
	for i := 0; i < len(s.messages); i++ {
		msg := s.messages[i]
		switch msg.Role {
		case "assistant":
			key := normalizeQuestionForMetric(msg.Content)
			if key != "" {
				questionSet[key]++
				questionCount++
			}
		case "user":
			answerChars += utf8.RuneCountInString(msg.Content)
		}
	}

	if m.QuestionsTotal > 0 {
		m.AvgAnswerLength = float64(answerChars) / float64(m.QuestionsTotal)
	}
	if questionCount > 0 {
		duplicates := questionCount - len(questionSet)
		m.DuplicateQuestionRate = float64(duplicates) / float64(questionCount)
		m.DuplicateQuestionRate = math.Max(0, math.Min(1, m.DuplicateQuestionRate))
	}

	for _, outcome := range s.confirmationOutcomes {
		switch {
		case outcome.Confirmed:
			m.ConfirmationConfirmed++
		case outcome.Denied:
			m.ConfirmationDenied++
		default:
			m.ConfirmationUncertain++
		}
	}

	return m
}

func normalizeQuestionForMetric(q string) string {
	q = strings.TrimSpace(strings.ToLower(q))
	if q == "" {
		return ""
	}
	replacer := strings.NewReplacer("？", "?", "。", "", "、", "", " ", "", "\t", "")
	return replacer.Replace(q)
}
