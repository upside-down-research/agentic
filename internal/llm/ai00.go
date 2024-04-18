package llm

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/log"
	"net/http"
)

type AI00Server struct {
	Host        string
	middlewares []Middleware
}

func (llm AI00Server) Middlewares() []Middleware {
	return llm.middlewares
}

func (llm AI00Server) PushMiddleware(mw Middleware) {
	llm.middlewares = append(llm.middlewares, mw)
}

func (llm AI00Server) Model() string {
	return "ai00"
}

type AI00Response struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
		Message      struct {
			Content string `json:"content"`
			Role    string `json:"role"`
		} `json:"message"`
	} `json:"choices"`
	Model  string `json:"model"`
	Object string `json:"object"`
	Usage  struct {
		CompletionTokens int `json:"completion_tokens"`
		PromptTokens     int `json:"prompt_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func parseEvent(rawEvent string) (*AI00Response, error) {
	const dataPrefix = "data:"
	if len(rawEvent) > len(dataPrefix) && rawEvent[:len(dataPrefix)] == dataPrefix {
		var response AI00Response
		err := json.Unmarshal([]byte(rawEvent[len(dataPrefix):]), &response)
		if err != nil {
			return nil, err
		}
		return &response, nil
		//return rawEvent[len(dataPrefix):], nil
	}
	return nil, fmt.Errorf("invalid event format")
}

func (llm AI00Server) Completion(data *Query) (string, error) {
	TimedCompletion := TimeWrapper(llm.Model())
	return TimedCompletion(data, llm._completion)
}

func (llm AI00Server) _completion(data *Query) (string, error) {

	payloadBytes, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		panic(err)
	}

	inputBody := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/oai/chat/completions", llm.Host), inputBody)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Authorization", "Bearer ai00")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", llm.Host)
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Referer", llm.Host)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", "Agentic 1")

	client := &http.Client{
		Transport: &http.Transport{
			// Define a custom TLSClientConfig
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Skip TLS certificate verification
			},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Process the response
	if resp.StatusCode != http.StatusOK {
		// read the entire inputBody
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)

		log.Fatalf("Unexpected response status: %s - %s", resp.Status, buf.String())
	}

	// read the entire response body
	var ai00Response AI00Response
	err = json.NewDecoder(resp.Body).Decode(&ai00Response)
	if err != nil {
		return "", err
	}

	return ai00Response.Choices[0].Message.Content, nil
}
