package ai

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/openai/openai-go/v3"
	openaioption "github.com/openai/openai-go/v3/option"
)

const openAIDefaultModel = "gpt-5.4-mini"

type OpenAIProvider struct {
	client openai.Client
	model  string
}

func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	if model == "" {
		model = openAIDefaultModel
	}
	return &OpenAIProvider{
		client: openai.NewClient(openaioption.WithAPIKey(apiKey)),
		model:  model,
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) Summarize(ctx context.Context, notes string, systemPrompt string) (string, error) {
	return p.Chat(ctx, []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: notes},
	})
}

func (p *OpenAIProvider) GenerateTitle(ctx context.Context, summary string) (string, error) {
	return p.Chat(ctx, []Message{
		{Role: "system", Content: titleSystemPrompt},
		{Role: "user", Content: summary},
	})
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	response, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(p.model),
		Messages: openAIMessages(messages),
	})
	if err != nil {
		return "", fmt.Errorf("openai chat completions API failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}
	text := strings.TrimSpace(response.Choices[0].Message.Content)
	if text == "" {
		return "", fmt.Errorf("openai returned empty content")
	}
	return text, nil
}

func openAIMessages(messages []Message) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, message := range messages {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(message.Role)) {
		case "system", "developer":
			out = append(out, openai.DeveloperMessage(content))
		case "assistant":
			out = append(out, openai.AssistantMessage(content))
		default:
			out = append(out, openai.UserMessage(content))
		}
	}
	return out
}
