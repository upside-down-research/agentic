package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/log"
	"io/ioutil"
	"net/http"
	"time"
)

type Claude struct {
	Key          string
	_model       string
	_middlewares []Middleware
}

func (llm Claude) Middlewares() []Middleware {
	return llm._middlewares
}

func (llm Claude) PushMiddleware(mw Middleware) {
	llm._middlewares = append(llm._middlewares, mw)
}

func NewClaude(key, model string) *Claude {
	return &Claude{
		Key:    key,
		_model: model,
	}
}

func (llm Claude) Model() string {
	return llm._model
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

func (llm Claude) Completion(data *Query) (string, error) {
	TimedCompletion := TimeWrapper(llm.Model())
	return TimedCompletion(data, llm._completion)
}

func (llm Claude) _completion(data *Query) (string, error) {
	log.Printf("Claude Completion begun with model...%s.\n", llm.Model())
	// https://docs.anthropic.com/claude/reference/messages_post

	type ClaudeRequest struct {
		Model     string     `json:"model"`
		MaxTokens int        `json:"max_tokens"`
		Messages  []Messages `json:"messages"`
		// https://docs.anthropic.com/claude/docs/system-prompts
		System string `json:"system"`
	}

	req := ClaudeRequest{
		Model:     llm.Model(),
		MaxTokens: 4096,
		Messages:  data.Messages,
		// Claude doesn't like json.
		System: `You will respond to ALL human messages in JSON. 
                    Make sure the response correctly follows the JSON format.
                    If comments are to be made, they will go in a "comments" block in the JSON objects.

                    Remember these rules: building JSON: 
                   The first is that newline is not allowed in a JSON string. 
                   Use the two bytes \n to specify a newline, not an actual newline. 
                   If you use an interpreted string literal, then the \ must be quoted with a \. Example:
                   "Hello\\nWorld"

                    Always begin with a { or a [.`,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		fmt.Println("Error marshaling request:", err)
		return "", err
	}

	client := &http.Client{
		Timeout: 120 * time.Second,
	}
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
