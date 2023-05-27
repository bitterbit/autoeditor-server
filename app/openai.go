package app

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/sashabaranov/go-openai"
)

type OpenAISession struct {
	client *openai.Client
}

func NewOpenAISession(apiKey string) *OpenAISession {
	config := openai.DefaultConfig(apiKey)
	config.OrgID = "org-gqYhHRPUaVx0Q5SyRyOpfrrB"

	return &OpenAISession{
		client: openai.NewClientWithConfig(config),
	}
}

func newRequest(prompt string) openai.CompletionRequest {
	return openai.CompletionRequest{
		Model:       "text-davinci-003",
		Prompt:      prompt,
		MaxTokens:   100,
		Temperature: 0.8,
	}
}

func (s *OpenAISession) ModifyCode(ctx context.Context, code, prompt, lang string) (string, error) {
	fullPrompt := fmt.Sprintf("Language:\n%s\n\nCode:\n%s\n\nPrompt:%s\n\nModified code:", lang, code, prompt)
	parameters := newRequest(fullPrompt)

	// Generate the modified code using GPT-3.5
	response, err := s.client.CreateCompletion(ctx, parameters)
	if err != nil {
		log.Fatalf("Failed to generate completion: %v", err)
	}

	// Check for successful response and retrieve the modified code
	if len(response.Choices) > 0 {
		return response.Choices[0].Text, nil
	}

	return "", errors.New("no completion response received")
}

func (s *OpenAISession) ExplainModification(ctx context.Context, prompt, modification string) (string, error) {
	fullPrompt := "Modified Cod:\n" + modification + "\n\nPrompt:\n" + prompt + "\n\nReasoning:"
	parameters := newRequest(fullPrompt)

	// Generate the modified code using GPT-3.5
	response, err := s.client.CreateCompletion(ctx, parameters)
	if err != nil {
		log.Fatalf("Failed to generate code reasoning: %v", err)
	}

	// Check for successful response and retrieve the modified code
	if len(response.Choices) > 0 {
		return response.Choices[0].Text, nil
	}

	return "", errors.New("no completion reasoning response received")
}
