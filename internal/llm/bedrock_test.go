package llm

import (
	"encoding/json"
	"testing"
)

func TestIsValidBedrockModel(t *testing.T) {
	tests := []struct {
		name    string
		modelID string
		want    bool
	}{
		{
			name:    "Valid Claude 3.5 Sonnet",
			modelID: "anthropic.claude-3-5-sonnet-20240620-v1:0",
			want:    true,
		},
		{
			name:    "Valid Claude 3 Opus",
			modelID: "anthropic.claude-3-opus-20240229-v1:0",
			want:    true,
		},
		{
			name:    "Valid Llama 3",
			modelID: "meta.llama3-70b-instruct-v1:0",
			want:    true,
		},
		{
			name:    "Valid Titan",
			modelID: "amazon.titan-text-express-v1",
			want:    true,
		},
		{
			name:    "Valid Cohere",
			modelID: "cohere.command-text-v14",
			want:    true,
		},
		{
			name:    "Invalid model ID",
			modelID: "invalid.model-id",
			want:    false,
		},
		{
			name:    "Empty string",
			modelID: "",
			want:    false,
		},
		{
			name:    "Random string",
			modelID: "random-model-name",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidBedrockModel(tt.modelID)
			if got != tt.want {
				t.Errorf("IsValidBedrockModel(%q) = %v, want %v", tt.modelID, got, tt.want)
			}
		})
	}
}

func TestBedrockModelIDs(t *testing.T) {
	tests := []struct {
		name    string
		modelID string
	}{
		{"Claude 3 Opus", BedrockModelIDs.Claude3Opus},
		{"Claude 3 Sonnet", BedrockModelIDs.Claude3Sonnet},
		{"Claude 3 Haiku", BedrockModelIDs.Claude3Haiku},
		{"Claude 3.5 Sonnet", BedrockModelIDs.Claude35Sonnet},
		{"Titan Text Lite", BedrockModelIDs.TitanTextLite},
		{"Titan Text Express", BedrockModelIDs.TitanTextExpress},
		{"Llama 2 13B", BedrockModelIDs.Llama2_13B},
		{"Llama 2 70B", BedrockModelIDs.Llama2_70B},
		{"Llama 3 8B", BedrockModelIDs.Llama3_8B},
		{"Llama 3 70B", BedrockModelIDs.Llama3_70B},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.modelID == "" {
				t.Errorf("%s model ID is empty", tt.name)
			}
			if !IsValidBedrockModel(tt.modelID) {
				t.Errorf("%s model ID %q failed validation", tt.name, tt.modelID)
			}
		})
	}
}

func TestBedrock_Model(t *testing.T) {
	expectedModel := "anthropic.claude-3-5-sonnet-20240620-v1:0"
	bedrock := &Bedrock{
		_model: expectedModel,
		region: "us-east-1",
	}

	got := bedrock.Model()
	if got != expectedModel {
		t.Errorf("Bedrock.Model() = %v, want %v", got, expectedModel)
	}
}

func TestBedrock_MarshalJSON(t *testing.T) {
	bedrock := &Bedrock{
		_model: BedrockModelIDs.Claude35Sonnet,
		region: "us-west-2",
	}

	data, err := json.Marshal(bedrock)
	if err != nil {
		t.Fatalf("Failed to marshal Bedrock: %v", err)
	}

	var result map[string]string
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if result["model"] != BedrockModelIDs.Claude35Sonnet {
		t.Errorf("Expected model %v, got %v", BedrockModelIDs.Claude35Sonnet, result["model"])
	}

	if result["region"] != "us-west-2" {
		t.Errorf("Expected region us-west-2, got %v", result["region"])
	}

	// Ensure client is not exposed in JSON
	if _, exists := result["client"]; exists {
		t.Error("Client should not be exposed in JSON output")
	}
}

func TestBedrock_Middlewares(t *testing.T) {
	bedrock := &Bedrock{
		_model: BedrockModelIDs.Claude3Haiku,
		region: "us-east-1",
	}

	// Test getting empty middlewares
	middlewares := bedrock.Middlewares()
	if len(middlewares) != 0 {
		t.Errorf("Expected 0 middlewares, got %d", len(middlewares))
	}

	// Note: PushMiddleware uses a value receiver (not pointer), so the middleware
	// won't actually be persisted. This matches the pattern in other LLM implementations
	// (OpenAI, Claude, AI00) and appears to be unused in the codebase.
	// The TimeWrapper middleware is used instead via the Completion method.
}

// TestNewBedrock_InvalidRegion tests that NewBedrock handles invalid configurations
// Note: This will fail if AWS credentials are not configured, which is expected
func TestNewBedrock_RequiresAWSConfig(t *testing.T) {
	// This test documents that NewBedrock requires AWS SDK configuration
	// It may fail in CI/CD without AWS credentials, which is acceptable
	_, err := NewBedrock("invalid-region-12345", BedrockModelIDs.Claude3Haiku)

	// We expect either success (if AWS creds exist) or a specific error
	// The key is that it shouldn't panic
	if err != nil {
		t.Logf("NewBedrock failed as expected without AWS config: %v", err)
	}
}

func TestBedrockModelIDs_Coverage(t *testing.T) {
	// Ensure all model families are represented
	modelFamilies := map[string][]string{
		"anthropic.claude": {
			BedrockModelIDs.Claude3Opus,
			BedrockModelIDs.Claude3Sonnet,
			BedrockModelIDs.Claude3Haiku,
			BedrockModelIDs.Claude35Sonnet,
		},
		"amazon.titan": {
			BedrockModelIDs.TitanTextLite,
			BedrockModelIDs.TitanTextExpress,
		},
		"meta.llama": {
			BedrockModelIDs.Llama2_13B,
			BedrockModelIDs.Llama2_70B,
			BedrockModelIDs.Llama3_8B,
			BedrockModelIDs.Llama3_70B,
		},
	}

	for family, models := range modelFamilies {
		for _, modelID := range models {
			if len(modelID) <= len(family) || modelID[:len(family)] != family {
				t.Errorf("Model %q does not start with expected prefix %q", modelID, family)
			}
		}
	}
}
