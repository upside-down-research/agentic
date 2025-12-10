package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/charmbracelet/log"
)

type Bedrock struct {
	client       *bedrockruntime.Client
	_model       string
	_middlewares []Middleware
	region       string
}

func (llm Bedrock) Middlewares() []Middleware {
	return llm._middlewares
}

func (llm Bedrock) PushMiddleware(mw Middleware) {
	llm._middlewares = append(llm._middlewares, mw)
}

func (llm Bedrock) Model() string {
	return llm._model
}

// NewBedrock creates a new Bedrock client using AWS credentials from the environment
// Credentials are loaded from:
// - Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN)
// - Shared credentials file (~/.aws/credentials)
// - IAM role for EC2/ECS/Lambda
func NewBedrock(region string, model string) (*Bedrock, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config: %w", err)
	}

	client := bedrockruntime.NewFromConfig(cfg)

	return &Bedrock{
		client: client,
		_model: model,
		region: region,
	}, nil
}

func (llm Bedrock) Completion(data *Query) (string, error) {
	TimedCompletion := TimeWrapper(llm.Model())
	return TimedCompletion(data, llm._completion)
}

func (llm Bedrock) _completion(data *Query) (string, error) {
	log.Infof("Bedrock Completion begun with model %s in region %s...", llm.Model(), llm.region)

	// Convert our standard Messages format to Bedrock Converse API format
	var messages []types.Message
	for _, msg := range data.Messages {
		messages = append(messages, types.Message{
			Role: types.ConversationRole(msg.Role),
			Content: []types.ContentBlock{
				&types.ContentBlockMemberText{
					Value: msg.Content,
				},
			},
		})
	}

	// System prompt to enforce JSON output (similar to Claude implementation)
	systemPrompt := `You will respond to ALL human messages in JSON.
Make sure the response correctly follows the JSON format.
If comments are to be made, they will go in a "comments" block in the JSON objects.

Remember these rules for building JSON:
The first is that newline is not allowed in a JSON string.
Use the two bytes \n to specify a newline, not an actual newline.
If you use an interpreted string literal, then the \ must be quoted with a \. Example:
"Hello\\nWorld"

Always begin with a { or a [.`

	// Build the Converse API request
	input := &bedrockruntime.ConverseInput{
		ModelId: aws.String(llm.Model()),
		Messages: messages,
		System: []types.SystemContentBlock{
			&types.SystemContentBlockMemberText{
				Value: systemPrompt,
			},
		},
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens:   aws.Int32(4096),
			Temperature: aws.Float32(float32(data.Temperature)),
		},
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Call the Bedrock Converse API
	log.Info("Bedrock Converse API request...")
	output, err := llm.client.Converse(ctx, input)
	if err != nil {
		return "", fmt.Errorf("bedrock converse error: %w", err)
	}

	// Extract the response text
	if output.Output == nil {
		return "", fmt.Errorf("bedrock returned nil output")
	}

	// The output should be a ContentBlockMemberText
	switch v := output.Output.(type) {
	case *types.ConverseOutputMemberMessage:
		if len(v.Value.Content) == 0 {
			log.Error("No content in Bedrock response")
			return "", fmt.Errorf("no content in bedrock response")
		}

		// Get the first content block (should be text)
		switch content := v.Value.Content[0].(type) {
		case *types.ContentBlockMemberText:
			responseText := content.Value
			log.Debugf("Bedrock response received, length: %d", len(responseText))

			// Log usage metrics if available
			if output.Usage != nil {
				log.Debugf("Bedrock usage - Input tokens: %d, Output tokens: %d",
					output.Usage.InputTokens,
					output.Usage.OutputTokens)
			}

			return responseText, nil
		default:
			return "", fmt.Errorf("unexpected content block type: %T", content)
		}
	default:
		return "", fmt.Errorf("unexpected output type: %T", v)
	}
}

// BedrockModelIDs contains common Bedrock model identifiers
// Users can override these with --model flag
var BedrockModelIDs = struct {
	// Claude 3 models
	Claude3Opus    string
	Claude3Sonnet  string
	Claude3Haiku   string
	Claude35Sonnet string

	// Amazon Titan models
	TitanTextLite   string
	TitanTextExpress string

	// Meta Llama models
	Llama2_13B  string
	Llama2_70B  string
	Llama3_8B   string
	Llama3_70B  string
}{
	Claude3Opus:      "anthropic.claude-3-opus-20240229-v1:0",
	Claude3Sonnet:    "anthropic.claude-3-sonnet-20240229-v1:0",
	Claude3Haiku:     "anthropic.claude-3-haiku-20240307-v1:0",
	Claude35Sonnet:   "anthropic.claude-3-5-sonnet-20240620-v1:0",
	TitanTextLite:    "amazon.titan-text-lite-v1",
	TitanTextExpress: "amazon.titan-text-express-v1",
	Llama2_13B:       "meta.llama2-13b-chat-v1",
	Llama2_70B:       "meta.llama2-70b-chat-v1",
	Llama3_8B:        "meta.llama3-8b-instruct-v1:0",
	Llama3_70B:       "meta.llama3-70b-instruct-v1:0",
}

// Helper function to validate if a string is a valid Bedrock model ID
func IsValidBedrockModel(modelID string) bool {
	validPrefixes := []string{
		"anthropic.claude",
		"amazon.titan",
		"meta.llama",
		"ai21.j2",
		"cohere.command",
		"stability.stable-diffusion",
	}

	for _, prefix := range validPrefixes {
		if len(modelID) > len(prefix) && modelID[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// MarshalJSON implements custom JSON marshaling to avoid exposing internal client
func (llm Bedrock) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Model  string `json:"model"`
		Region string `json:"region"`
	}{
		Model:  llm._model,
		Region: llm.region,
	})
}
