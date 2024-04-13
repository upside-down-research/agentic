package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type Claude struct {
	Key   string
	Model string
}

func NewClaude(key string) *Claude {
	return &Claude{
		Key:   key,
		Model: "claude-3-opus-20240229",
	}
}

type ClaudeResponse struct {
	ID           string    `json:"id"`
	Content      []Content `json:"content"`
	Model        string    `json:"model"`
	StopReason   string    `json:"stop_reason"`
	StopSequence string    `json:"stop_sequence"`
	Usage        Usage     `json:"usage"`
}

type Content struct {
	Text  string      `json:"text"`
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Input interface{} `json:"input"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func (llm Claude) Completion(data *LLMQuery) (string, error) {
	log.Println("Claude Completion begun with model...", data.Model)
	// https://docs.anthropic.com/claude/reference/messages_post

	type Request struct {
		Model     string     `json:"model"`
		MaxTokens int        `json:"max_tokens"`
		Messages  []Messages `json:"messages"`
	}

	req := Request{
		Model:     llm.Model,
		MaxTokens: 1000,
		Messages:  data.Messages,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		fmt.Println("Error marshaling request:", err)
		return "", err
	}

	client := &http.Client{}
	httpReq, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return "", err
	}

	httpReq.Header.Set("x-api-key", llm.Key)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return "", err
	}
	var holdingData ClaudeResponse
	err = json.Unmarshal(body, &holdingData)
	if err != nil {
		return "", err
	}

	return holdingData.Content[0].Text, nil
}
