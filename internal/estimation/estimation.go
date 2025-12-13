package estimation

import (
	"fmt"
	"strings"
)

// TokenEstimate represents an estimated token count
type TokenEstimate struct {
	PromptTokens    int
	CompletionEst   int
	TotalEst        int
	ReviewCycles    int
	TotalWithReview int
}

// CostEstimate represents estimated costs
type CostEstimate struct {
	Tokens       TokenEstimate
	CostUSD      float64
	EstimatedMin int // Estimated minutes
}

// Pricing holds per-model pricing (cost per 1M tokens)
type Pricing struct {
	Model              string
	InputPer1M         float64
	OutputPer1M        float64
	AvgCompletionRatio float64 // typical output/input ratio
}

var modelPricing = map[string]Pricing{
	"gpt-4-turbo": {
		Model:              "gpt-4-turbo",
		InputPer1M:         10.0,
		OutputPer1M:        30.0,
		AvgCompletionRatio: 0.5,
	},
	"gpt-3.5-turbo": {
		Model:              "gpt-3.5-turbo",
		InputPer1M:         0.50,
		OutputPer1M:        1.50,
		AvgCompletionRatio: 0.5,
	},
	"claude-3-opus-20240229": {
		Model:              "claude-3-opus",
		InputPer1M:         15.0,
		OutputPer1M:        75.0,
		AvgCompletionRatio: 0.6,
	},
	"claude-3-haiku-20240307": {
		Model:              "claude-3-haiku",
		InputPer1M:         0.25,
		OutputPer1M:        1.25,
		AvgCompletionRatio: 0.5,
	},
	"claude-3-5-sonnet-20240620": {
		Model:              "claude-3.5-sonnet",
		InputPer1M:         3.0,
		OutputPer1M:        15.0,
		AvgCompletionRatio: 0.6,
	},
}

// EstimateTokens estimates token count from text
// Rough approximation: ~4 chars per token
func EstimateTokens(text string) int {
	return len(text) / 4
}

// EstimateCost estimates the cost of a generation task
func EstimateCost(model, prompt string, reviewCycles int) *CostEstimate {
	pricing, ok := modelPricing[model]
	if !ok {
		// Use GPT-4 pricing as conservative default
		pricing = modelPricing["gpt-4-turbo"]
	}

	promptTokens := EstimateTokens(prompt)
	completionEst := int(float64(promptTokens) * pricing.AvgCompletionRatio)
	totalEst := promptTokens + completionEst

	// Account for review cycles
	// Each review cycle: original prompt + response + review prompt + review response
	reviewOverhead := int(float64(totalEst) * 0.3) // Review adds ~30% overhead
	totalWithReview := totalEst * (reviewCycles + 1)
	totalWithReview += reviewOverhead * reviewCycles

	// Calculate cost
	inputCost := float64(promptTokens*(reviewCycles+1)) * pricing.InputPer1M / 1_000_000
	outputCost := float64(completionEst*(reviewCycles+1)) * pricing.OutputPer1M / 1_000_000
	reviewCost := float64(reviewOverhead*reviewCycles) * pricing.OutputPer1M / 1_000_000
	totalCost := inputCost + outputCost + reviewCost

	// Estimate time (very rough: ~1000 tokens per 10 seconds)
	estimatedSec := totalWithReview / 100 // ~10 tokens per second
	estimatedMin := estimatedSec / 60
	if estimatedMin < 1 {
		estimatedMin = 1
	}

	return &CostEstimate{
		Tokens: TokenEstimate{
			PromptTokens:    promptTokens,
			CompletionEst:   completionEst,
			TotalEst:        totalEst,
			ReviewCycles:    reviewCycles,
			TotalWithReview: totalWithReview,
		},
		CostUSD:      totalCost,
		EstimatedMin: estimatedMin,
	}
}

// EstimateGeneration estimates cost for a full generation run
func EstimateGeneration(model, specContent string, expectedComponents int) *CostEstimate {
	// Planning phase
	planningCost := EstimateCost(model, specContent, 2)

	// Implementation phase (per component)
	avgComponentSize := len(specContent) / 2 // Each component is roughly half the spec
	implCost := EstimateCost(model, specContent+strings.Repeat("x", avgComponentSize), 3)

	// Multiply by number of components
	totalTokens := planningCost.Tokens.TotalWithReview +
		(implCost.Tokens.TotalWithReview * expectedComponents)

	totalCost := planningCost.CostUSD + (implCost.CostUSD * float64(expectedComponents))
	totalMin := planningCost.EstimatedMin + (implCost.EstimatedMin * expectedComponents)

	return &CostEstimate{
		Tokens: TokenEstimate{
			TotalWithReview: totalTokens,
		},
		CostUSD:      totalCost,
		EstimatedMin: totalMin,
	}
}

// FormatEstimate formats a cost estimate for display
func FormatEstimate(est *CostEstimate) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Estimated cost: $%.2f\n", est.CostUSD))
	sb.WriteString(fmt.Sprintf("Estimated time: %d-%d minutes\n",
		est.EstimatedMin, est.EstimatedMin*2))
	sb.WriteString(fmt.Sprintf("Estimated tokens: ~%s",
		formatNumber(est.Tokens.TotalWithReview)))

	return sb.String()
}

// ShouldProceed checks if estimate is within limits
func ShouldProceed(est *CostEstimate, maxCost float64, maxTokens int) (bool, string) {
	if maxCost > 0 && est.CostUSD > maxCost {
		return false, fmt.Sprintf("Estimated cost ($%.2f) exceeds limit ($%.2f)",
			est.CostUSD, maxCost)
	}

	if maxTokens > 0 && est.Tokens.TotalWithReview > maxTokens {
		return false, fmt.Sprintf("Estimated tokens (%s) exceeds limit (%s)",
			formatNumber(est.Tokens.TotalWithReview), formatNumber(maxTokens))
	}

	return true, ""
}

func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	// Add commas
	var parts []string
	for i := len(s); i > 0; i -= 3 {
		start := i - 3
		if start < 0 {
			start = 0
		}
		parts = append([]string{s[start:i]}, parts...)
	}
	return strings.Join(parts, ",")
}
