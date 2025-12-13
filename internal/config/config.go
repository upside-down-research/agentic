package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	LLM         LLMConfig         `yaml:"llm"`
	Output      OutputConfig      `yaml:"output"`
	Retry       RetryConfig       `yaml:"retry"`
	QualityGate QualityGateConfig `yaml:"quality_gates"`
	Cost        CostConfig        `yaml:"cost"`
}

// LLMConfig holds LLM provider settings
type LLMConfig struct {
	Provider string  `yaml:"provider"` // openai, claude, bedrock, ai00
	Model    *string `yaml:"model"`    // optional, uses sensible defaults
	APIKey   string  `yaml:"api_key"`  // supports ${ENV_VAR} interpolation
	AWSRegion string `yaml:"aws_region"`
}

// OutputConfig holds output settings
type OutputConfig struct {
	Directory       string `yaml:"directory"`
	PreserveHistory bool   `yaml:"preserve_history"`
}

// RetryConfig holds retry behavior settings
type RetryConfig struct {
	MaxAttempts int `yaml:"max_attempts"`
	TimeoutSec  int `yaml:"timeout_sec"`
}

// QualityGateConfig holds quality gate settings
type QualityGateConfig struct {
	RequireCompilation bool `yaml:"require_compilation"`
	RunTests           bool `yaml:"run_tests"`
	MaxReviewCycles    int  `yaml:"max_review_cycles"`
}

// CostConfig holds cost control settings
type CostConfig struct {
	MaxCostUSD  float64 `yaml:"max_cost_usd"`
	MaxTokens   int     `yaml:"max_tokens"`
	WarnOnCost  bool    `yaml:"warn_on_cost"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		LLM: LLMConfig{
			Provider:  "openai",
			AWSRegion: "us-east-1",
		},
		Output: OutputConfig{
			Directory:       "./output",
			PreserveHistory: true,
		},
		Retry: RetryConfig{
			MaxAttempts: 5,
			TimeoutSec:  120,
		},
		QualityGate: QualityGateConfig{
			RequireCompilation: false,
			RunTests:           false,
			MaxReviewCycles:    10,
		},
		Cost: CostConfig{
			MaxCostUSD: 10.0,
			MaxTokens:  100000,
			WarnOnCost: true,
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Use defaults if file doesn't exist
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables in the config
	expanded := os.ExpandEnv(string(data))

	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// SaveConfig saves configuration to a YAML file
func SaveConfig(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ExampleConfig returns a commented example config
func ExampleConfig() string {
	return `# Agentic Configuration File
# Priority: CLI flags > environment variables > config file > defaults

llm:
  # Provider: openai, claude, bedrock, ai00
  provider: openai

  # Model: leave empty for sensible defaults
  # OpenAI: gpt-4-turbo, gpt-3.5-turbo
  # Claude: claude-3-opus-20240229, claude-3-haiku-20240307
  # Bedrock: anthropic.claude-3-5-sonnet-20240620-v1:0
  model: ""

  # API Key: supports ${ENV_VAR} interpolation
  api_key: ${OPENAI_API_KEY}

  # AWS Region (for Bedrock only)
  aws_region: us-east-1

output:
  # Directory for run artifacts
  directory: ./output

  # Preserve full history of LLM interactions
  preserve_history: true

retry:
  # Maximum retry attempts for LLM calls
  max_attempts: 5

  # Timeout for each LLM call (seconds)
  timeout_sec: 120

quality_gates:
  # Require code to compile before accepting
  require_compilation: false

  # Run tests and require passing
  run_tests: false

  # Maximum review cycles before giving up
  max_review_cycles: 10

cost:
  # Abort if estimated cost exceeds this (USD)
  max_cost_usd: 10.0

  # Abort if estimated tokens exceed this
  max_tokens: 100000

  # Warn before running expensive operations
  warn_on_cost: true
`
}
