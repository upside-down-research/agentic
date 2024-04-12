package llm

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type LLMQuery struct {
	Model            string     `json:"model,omitempty"`
	Messages         []Messages `json:"messages"`
	MaxTokens        int        `json:"max_tokens"`
	Temperature      int        `json:"temperature"`
	TopP             float64    `json:"top_p,omitempty"`
	PresencePenalty  float64    `json:"presence_penalty"`
	FrequencyPenalty float64    `json:"frequency_penalty"`
	PenaltyDecay     float64    `json:"penalty_decay"`
	Stop             []string   `json:"stop"`
	Stream           bool       `json:"stream"`
	Names            Names      `json:"names"`
}

func NewChatQuery(n Names, m []Messages) *LLMQuery {
	r := &LLMQuery{
		Messages:         m,
		MaxTokens:        1000,
		TopP:             0.5,
		Temperature:      1,
		PresencePenalty:  0.3,
		FrequencyPenalty: 0.3,
		PenaltyDecay:     0.9982686325973925,
		Stop:             []string{"↵User:", "User:", "\n\n"},
		Stream:           true,
		Names:            n,
	}
	return r
}

type LLMServer interface {
	Completion(data *LLMQuery) (string, error)
}

type Messages struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type Names struct {
	User      string `json:"user"`
	Assistant string `json:"assistant"`
}

type OpenAI struct {
	Key string
}

func (llm OpenAI) Completion(data *LLMQuery) (string, error) {
	log.Println("OpenAI Completion begun...")
	type CompletionResponse struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int    `json:"created"`
		Model   string `json:"model"`
		Choices []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			Logprobs     interface{} `json:"logprobs"`
			FinishReason string      `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		SystemFingerprint string `json:"system_fingerprint"`
	}
	url := "https://api.openai.com/v1/chat/completions"
	method := "POST"

	type OpenAIQuery struct {
		Model    string     `json:"model"`
		Messages []Messages `json:"messages"`
	}

	payload := &OpenAIQuery{
		Model:    "gpt-4-turbo",
		Messages: data.Messages,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	client := &http.Client{
		Timeout: time.Duration(60 * time.Second),
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+llm.Key)

	log.Println("OpenAI Completion request...")

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var CompletionResponseData CompletionResponse
	err = json.Unmarshal(body, &CompletionResponseData)
	if err != nil {
		return "", err
	}

	return string(CompletionResponseData.Choices[0].Message.Content), nil
}

type Claude struct {
	Key string
}

type VertexAI struct {
	// no idea
}
