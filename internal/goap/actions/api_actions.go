package actions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"upside-down-research.com/oss/agentic/internal/goap"
)

// HTTPRequestAction performs HTTP requests to external APIs
type HTTPRequestAction struct {
	*goap.BaseAction
	method   string
	url      string
	headers  map[string]string
	body     []byte
	resultKey string
}

func NewHTTPRequestAction(method, url string, headers map[string]string, body []byte, resultKey string, preconditions goap.WorldState) *HTTPRequestAction {
	return &HTTPRequestAction{
		BaseAction: goap.NewBaseAction(
			"HTTPRequest",
			fmt.Sprintf("%s %s", method, url),
			preconditions,
			goap.WorldState{resultKey + "_fetched": true},
			5.0, // Medium complexity - network operation
		),
		method:    method,
		url:       url,
		headers:   headers,
		body:      body,
		resultKey: resultKey,
	}
}

func (a *HTTPRequestAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for HTTPRequest")
	}

	log.Info("Making HTTP request", "method", a.method, "url", a.url)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var req *http.Request
	var err error

	if a.body != nil {
		req, err = http.NewRequestWithContext(ctx, a.method, a.url, bytes.NewReader(a.body))
	} else {
		req, err = http.NewRequestWithContext(ctx, a.method, a.url, nil)
	}

	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range a.headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	current.Set(a.resultKey+"_fetched", true)
	current.Set(a.resultKey+"_status", resp.StatusCode)
	current.Set(a.resultKey+"_body", string(respBody))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Warn("HTTP request returned error status", "status", resp.StatusCode)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	log.Info("HTTP request succeeded", "status", resp.StatusCode)
	return nil
}

func (a *HTTPRequestAction) Clone() goap.Action {
	return NewHTTPRequestAction(a.method, a.url, a.headers, a.body, a.resultKey, a.Preconditions().Clone())
}

// LLMPromptAction calls an LLM (as a generator, not a reasoner!)
type LLMPromptAction struct {
	*goap.BaseAction
	ctx       *ActionContext
	prompt    string
	resultKey string
}

func NewLLMPromptAction(ctx *ActionContext, prompt, resultKey string, preconditions goap.WorldState) *LLMPromptAction {
	return &LLMPromptAction{
		BaseAction: goap.NewBaseAction(
			"LLMPrompt",
			"Generate content via LLM",
			preconditions,
			goap.WorldState{resultKey + "_generated": true},
			7.0, // LLM call - moderate cost
		),
		ctx:       ctx,
		prompt:    prompt,
		resultKey: resultKey,
	}
}

func (a *LLMPromptAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for LLMPrompt")
	}

	log.Info("Generating content via LLM (as generator, not reasoner)")

	// Use the LLM as a GENERATOR, not for planning/reasoning
	// The GOAP system (GOFAI) does the reasoning
	// The LLM just generates content based on our logical plan

	// This would call the actual LLM here
	// For now, simulate the call
	current.Set(a.resultKey+"_generated", true)
	current.Set(a.resultKey+"_prompt", a.prompt)

	log.Info("LLM generation complete (content created, not reasoning)")
	return nil
}

func (a *LLMPromptAction) Clone() goap.Action {
	return NewLLMPromptAction(a.ctx, a.prompt, a.resultKey, a.Preconditions().Clone())
}

// WebhookAction sends notifications to webhooks
type WebhookAction struct {
	*goap.BaseAction
	webhookURL string
	payload    interface{}
	eventType  string
}

func NewWebhookAction(webhookURL, eventType string, payload interface{}, preconditions goap.WorldState) *WebhookAction {
	return &WebhookAction{
		BaseAction: goap.NewBaseAction(
			"Webhook",
			fmt.Sprintf("Send webhook: %s", eventType),
			preconditions,
			goap.WorldState{"webhook_sent": true},
			3.0,
		),
		webhookURL: webhookURL,
		payload:    payload,
		eventType:  eventType,
	}
}

func (a *WebhookAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for Webhook")
	}

	log.Info("Sending webhook", "type", a.eventType, "url", a.webhookURL)

	payloadJSON, err := json.Marshal(map[string]interface{}{
		"event": a.eventType,
		"data":  a.payload,
		"timestamp": time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST", a.webhookURL, bytes.NewReader(payloadJSON))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	current.Set("webhook_sent", true)
	current.Set("webhook_status", resp.StatusCode)

	log.Info("Webhook sent", "status", resp.StatusCode)
	return nil
}

func (a *WebhookAction) Clone() goap.Action {
	return NewWebhookAction(a.webhookURL, a.eventType, a.payload, a.Preconditions().Clone())
}
