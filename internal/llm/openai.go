package llm

import (
	"bytes"
	"encoding/json"
	"github.com/charmbracelet/log"
	"io/ioutil"
	"net/http"
	"time"
)

type OpenAI struct {
	Key          string
	_model       string
	_middlewares []Middleware
}

func (llm OpenAI) Middlewares() []Middleware {
	return llm._middlewares
}
func (llm OpenAI) PushMiddleware(mw Middleware) {
	llm._middlewares = append(llm._middlewares, mw)
}

func (llm OpenAI) Model() string {
	return llm._model
}
func NewOpenAI(key string, model string) *OpenAI {
	return &OpenAI{
		Key:    key,
		_model: model,
	}
}

func (llm OpenAI) Completion(data *Query) (string, error) {
	TimedCompletion := TimeWrapper(llm.Model())
	return TimedCompletion(data, llm._completion)
}

func (llm OpenAI) _completion(data *Query) (string, error) {
	log.Info("OpenAI Completion begun...")
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

	type ResponseFormat struct {
		Type string `json:"type"`
	}
	type OpenAIQuery struct {
		Model    string     `json:"model"`
		Messages []Messages `json:"messages"`

		ResponseFormat `json:"response_format"`
	}

	payload := &OpenAIQuery{
		Model:    llm.Model(),
		Messages: data.Messages,
		ResponseFormat: ResponseFormat{
			Type: "json_object",
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	client := &http.Client{
		Timeout: time.Duration(120 * time.Second),
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+llm.Key)

	log.Info("OpenAI Completion request...")
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	log.Debugf("reading the response")
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var CompletionResponseData CompletionResponse
	err = json.Unmarshal(body, &CompletionResponseData)
	if err != nil {
		return "", err
	}

	if len(CompletionResponseData.Choices) == 0 {
		log.Error("No results given", "body", string(body), "query", payload)
		return "", nil
	}

	return string(CompletionResponseData.Choices[0].Message.Content), nil
}
