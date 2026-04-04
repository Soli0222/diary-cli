package ai

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

const geminiDefaultModel = "gemini-3.1-flash-preview"

type GeminiProvider struct {
	client *genai.Client
	model  string
}

func NewGeminiProvider(ctx context.Context, apiKey, model string) (*GeminiProvider, error) {
	if model == "" {
		model = geminiDefaultModel
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}
	return &GeminiProvider{
		client: client,
		model:  model,
	}, nil
}

func (p *GeminiProvider) Name() string {
	return "gemini"
}

func (p *GeminiProvider) Summarize(ctx context.Context, notes string, systemPrompt string) (string, error) {
	return p.Chat(ctx, []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: notes},
	})
}

func (p *GeminiProvider) GenerateTitle(ctx context.Context, summary string) (string, error) {
	return p.Chat(ctx, []Message{
		{Role: "system", Content: titleSystemPrompt},
		{Role: "user", Content: summary},
	})
}

func geminiRequest(messages []Message) ([]*genai.Content, *genai.GenerateContentConfig) {
	var (
		systemParts []string
		contents    []*genai.Content
	)

	for _, message := range messages {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(message.Role)) {
		case "system", "developer":
			systemParts = append(systemParts, content)
		case "assistant":
			contents = append(contents, genai.NewContentFromText(content, genai.RoleModel))
		default:
			contents = append(contents, genai.NewContentFromText(content, genai.RoleUser))
		}
	}

	config := &genai.GenerateContentConfig{}
	if len(systemParts) > 0 {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: strings.Join(systemParts, "\n\n")}},
		}
	}
	return contents, config
}

func (p *GeminiProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	contents, config := geminiRequest(messages)

	response, err := p.client.Models.GenerateContent(ctx, p.model, contents, config)
	if err != nil {
		return "", fmt.Errorf("gemini generate content failed: %w", err)
	}

	text := strings.TrimSpace(response.Text())
	if text == "" {
		return "", fmt.Errorf("gemini returned empty content")
	}
	return text, nil
}
