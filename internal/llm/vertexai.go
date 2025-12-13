package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/log"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type VertexAI struct {
	ProjectID    string
	Location     string
	_model       string
	_middlewares []Middleware
}

func (llm VertexAI) Middlewares() []Middleware {
	return llm._middlewares
}

func (llm VertexAI) PushMiddleware(mw Middleware) {
	llm._middlewares = append(llm._middlewares, mw)
}

func NewVertexAI(projectID, location, model string) *VertexAI {
	return &VertexAI{
		ProjectID: projectID,
		Location:  location,
		_model:    model,
	}
}

func (llm VertexAI) Model() string {
	return llm._model
}

// VertexAI API request/response structures for Gemini models
type VertexAIRequest struct {
	Contents         []VertexAIContent    `json:"contents"`
	GenerationConfig GenerationConfig     `json:"generation_config,omitempty"`
	SafetySettings   []SafetySetting      `json:"safety_settings,omitempty"`
}

type VertexAIContent struct {
	Role  string        `json:"role"`
	Parts []ContentPart `json:"parts"`
}

type ContentPart struct {
	Text string `json:"text"`
}

type GenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	ResponseMimeType string `json:"responseMimeType,omitempty"`
}

type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type VertexAIResponse struct {
	Candidates     []Candidate    `json:"candidates"`
	UsageMetadata  UsageMetadata  `json:"usageMetadata"`
	ModelVersion   string         `json:"modelVersion"`
}

type Candidate struct {
	Content       VertexAIContent `json:"content"`
	FinishReason  string          `json:"finishReason"`
	SafetyRatings []SafetyRating  `json:"safetyRatings"`
}

type SafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

func (llm VertexAI) Completion(data *Query) (string, error) {
	TimedCompletion := TimeWrapper(llm.Model())
	return TimedCompletion(data, llm._completion)
}

func (llm VertexAI) _completion(data *Query) (string, error) {
	log.Printf("VertexAI Completion begun with model...%s.\n", llm.Model())

	// Get access token for authentication
	accessToken, err := llm.getAccessToken()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	// Convert messages to Vertex AI format
	contents := make([]VertexAIContent, 0, len(data.Messages))
	for _, msg := range data.Messages {
		role := msg.Role
		// Vertex AI uses "user" and "model" roles, not "assistant"
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, VertexAIContent{
			Role: role,
			Parts: []ContentPart{
				{Text: msg.Content},
			},
		})
	}

	// Prepare request with JSON response format
	req := VertexAIRequest{
		Contents: contents,
		GenerationConfig: GenerationConfig{
			Temperature:      0.7,
			TopP:             0.95,
			MaxOutputTokens:  4096,
			ResponseMimeType: "application/json",
		},
		SafetySettings: []SafetySetting{
			{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	// Build API endpoint URL
	url := fmt.Sprintf(
		"https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent",
		llm.Location,
		llm.ProjectID,
		llm.Location,
		llm.Model(),
	)

	client := &http.Client{
		Timeout: 120 * time.Second,
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	log.Info("VertexAI Completion request...")
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var vertexResp VertexAIResponse
	err = json.Unmarshal(body, &vertexResp)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling response: %w", err)
	}

	if len(vertexResp.Candidates) == 0 {
		log.Error("No results given", "body", string(body))
		return "", fmt.Errorf("no candidates in response")
	}

	if len(vertexResp.Candidates[0].Content.Parts) == 0 {
		log.Error("No content parts in response", "body", string(body))
		return "", fmt.Errorf("no content parts in response")
	}

	return vertexResp.Candidates[0].Content.Parts[0].Text, nil
}

// getAccessToken retrieves an access token for Vertex AI API authentication
// It tries to use gcloud auth print-access-token, which works with Application Default Credentials
func (llm VertexAI) getAccessToken() (string, error) {
	// First, check if there's an explicit GOOGLE_VERTEX_TOKEN environment variable
	if token := os.Getenv("GOOGLE_VERTEX_TOKEN"); token != "" {
		return token, nil
	}

	// Try to use gcloud CLI to get access token
	cmd := exec.Command("gcloud", "auth", "print-access-token")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get access token from gcloud (make sure gcloud is installed and authenticated): %w", err)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("empty access token received from gcloud")
	}

	return token, nil
}
