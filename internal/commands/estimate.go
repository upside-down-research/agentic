package commands

import (
	"fmt"
	"os"

	"upside-down-research.com/oss/agentic/internal/config"
	"upside-down-research.com/oss/agentic/internal/estimation"
	"upside-down-research.com/oss/agentic/internal/validation"
)

// EstimateCommand estimates cost and time for a specification
type EstimateCommand struct {
	SpecFile   string `arg:"" name:"spec" help:"Specification file" type:"path"`
	Config     string `name:"config" help:"Configuration file path" type:"path"`
	Model      string `name:"model" help:"Override model from config"`
	Components int    `name:"components" help:"Expected number of components" default:"3"`
}

// Run executes the estimate command
func (cmd *EstimateCommand) Run() error {
	fmt.Printf("💰 Estimating cost for: %s\n\n", cmd.SpecFile)

	// Validate spec file
	result := validation.ValidateSpecFile(cmd.SpecFile)
	if !result.IsValid() {
		validation.PrintValidationResult(result)
		return fmt.Errorf("validation failed")
	}

	// Load config
	cfg, err := config.LoadConfig(cmd.Config)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Determine model
	model := cmd.Model
	if model == "" && cfg.LLM.Model != nil {
		model = *cfg.LLM.Model
	}
	if model == "" {
		// Use default based on provider
		model = getDefaultModel(cfg.LLM.Provider)
	}

	// Read spec file
	data, err := os.ReadFile(cmd.SpecFile)
	if err != nil {
		return fmt.Errorf("failed to read spec file: %w", err)
	}

	// Estimate
	est := estimation.EstimateGeneration(model, string(data), cmd.Components)

	fmt.Println(estimation.FormatEstimate(est))
	fmt.Println()

	// Check against limits
	if cfg.Cost.MaxCostUSD > 0 || cfg.Cost.MaxTokens > 0 {
		ok, reason := estimation.ShouldProceed(est, cfg.Cost.MaxCostUSD, cfg.Cost.MaxTokens)
		if !ok {
			fmt.Printf("⚠️  Warning: %s\n", reason)
			fmt.Println("   Adjust limits in config or use --max-cost/--max-tokens flags")
		} else {
			fmt.Println("✓ Estimate is within configured limits")
		}
	}

	return nil
}

func getDefaultModel(provider string) string {
	defaults := map[string]string{
		"openai":  "gpt-4-turbo",
		"claude":  "claude-3-opus-20240229",
		"bedrock": "anthropic.claude-3-5-sonnet-20240620-v1:0",
		"ai00":    "rwkv",
	}
	if m, ok := defaults[provider]; ok {
		return m
	}
	return "gpt-4-turbo"
}
