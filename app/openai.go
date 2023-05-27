package app

import (
	"context"
	"errors"
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

func (s *OpenAISession) ModifyCode(ctx context.Context, code, prompt string) (string, error) {
	// Prepare the GPT-3.5 prompt format
	fullPrompt := "Code:\n" + code + "\n\nPrompt:\n" + prompt + "\n\nModified code:"

	// Set the GPT-3.5 parameters
	parameters := openai.CompletionRequest{
		Model:       "text-davinci-003", // Set the desired GPT model
		Prompt:      fullPrompt,
		MaxTokens:   100, // Set the desired maximum number of tokens in the response
		Temperature: 0.8, // Set the desired temperature for generating creative responses
	}

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
