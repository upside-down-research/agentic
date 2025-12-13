package validation

import (
	"fmt"
	"os"
	"path/filepath"

	"upside-down-research.com/oss/agentic/internal/config"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
	Fix     string // Suggested fix
}

func (e ValidationError) Error() string {
	msg := fmt.Sprintf("%s: %s", e.Field, e.Message)
	if e.Fix != "" {
		msg += fmt.Sprintf("\n  Fix: %s", e.Fix)
	}
	return msg
}

// ValidationResult holds validation results
type ValidationResult struct {
	Errors   []ValidationError
	Warnings []ValidationError
}

// IsValid returns true if there are no errors
func (v *ValidationResult) IsValid() bool {
	return len(v.Errors) == 0
}

// AddError adds a validation error
func (v *ValidationResult) AddError(field, message, fix string) {
	v.Errors = append(v.Errors, ValidationError{
		Field:   field,
		Message: message,
		Fix:     fix,
	})
}

// AddWarning adds a validation warning
func (v *ValidationResult) AddWarning(field, message, fix string) {
	v.Warnings = append(v.Warnings, ValidationError{
		Field:   field,
		Message: message,
		Fix:     fix,
	})
}

// ValidateConfig validates the configuration
func ValidateConfig(cfg *config.Config) *ValidationResult {
	result := &ValidationResult{}

	// Validate LLM provider
	validProviders := map[string]bool{
		"openai":  true,
		"claude":  true,
		"bedrock": true,
		"ai00":    true,
	}
	if !validProviders[cfg.LLM.Provider] {
		result.AddError("llm.provider",
			fmt.Sprintf("invalid provider '%s'", cfg.LLM.Provider),
			"use one of: openai, claude, bedrock, ai00")
	}

	// Validate API keys based on provider
	switch cfg.LLM.Provider {
	case "openai":
		if cfg.LLM.APIKey == "" {
			key := os.Getenv("OPENAI_API_KEY")
			if key == "" {
				result.AddError("llm.api_key",
					"OPENAI_API_KEY not set",
					"export OPENAI_API_KEY=sk-... or set in config file")
			}
		}
	case "claude":
		if cfg.LLM.APIKey == "" {
			key := os.Getenv("CLAUDE_API_KEY")
			if key == "" {
				result.AddError("llm.api_key",
					"CLAUDE_API_KEY not set",
					"export CLAUDE_API_KEY=... or set in config file")
			}
		}
	case "bedrock":
		// Check AWS credentials
		if os.Getenv("AWS_ACCESS_KEY_ID") == "" && os.Getenv("AWS_PROFILE") == "" {
			result.AddWarning("aws",
				"AWS credentials not found in environment",
				"export AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY, or set AWS_PROFILE")
		}
	}

	// Validate output directory
	if cfg.Output.Directory == "" {
		result.AddError("output.directory",
			"output directory not specified",
			"set output.directory in config or use --output flag")
	} else {
		// Check if directory is writable
		dir := cfg.Output.Directory
		if err := os.MkdirAll(dir, 0755); err != nil {
			result.AddError("output.directory",
				fmt.Sprintf("cannot create directory: %v", err),
				fmt.Sprintf("ensure %s is writable", dir))
		}
	}

	// Validate retry settings
	if cfg.Retry.MaxAttempts < 1 {
		result.AddError("retry.max_attempts",
			"must be at least 1",
			"set retry.max_attempts to a positive number")
	}
	if cfg.Retry.MaxAttempts > 20 {
		result.AddWarning("retry.max_attempts",
			"very high retry limit may cause long waits",
			"consider reducing to 10 or less")
	}

	if cfg.Retry.TimeoutSec < 10 {
		result.AddWarning("retry.timeout_sec",
			"timeout is very short",
			"LLM calls may time out; consider 60-120 seconds")
	}

	// Validate quality gates
	if cfg.QualityGate.MaxReviewCycles < 1 {
		result.AddError("quality_gates.max_review_cycles",
			"must be at least 1",
			"set quality_gates.max_review_cycles to a positive number")
	}

	// Validate cost limits
	if cfg.Cost.MaxCostUSD < 0 {
		result.AddError("cost.max_cost_usd",
			"cannot be negative",
			"set cost.max_cost_usd to a positive number or 0 for unlimited")
	}
	if cfg.Cost.MaxTokens < 0 {
		result.AddError("cost.max_tokens",
			"cannot be negative",
			"set cost.max_tokens to a positive number or 0 for unlimited")
	}

	return result
}

// ValidateSpecFile validates a specification file
func ValidateSpecFile(path string) *ValidationResult {
	result := &ValidationResult{}

	if path == "" {
		result.AddError("spec_file",
			"no specification file provided",
			"provide a .in file with your requirements")
		return result
	}

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			result.AddError("spec_file",
				fmt.Sprintf("file not found: %s", path),
				"check the file path and try again")
		} else {
			result.AddError("spec_file",
				fmt.Sprintf("cannot access file: %v", err),
				"check file permissions")
		}
		return result
	}

	// Check if it's a file
	if info.IsDir() {
		result.AddError("spec_file",
			fmt.Sprintf("%s is a directory", path),
			"provide a file, not a directory")
		return result
	}

	// Check if file is empty
	if info.Size() == 0 {
		result.AddError("spec_file",
			"file is empty",
			"add your requirements to the file")
		return result
	}

	// Check if file is readable
	data, err := os.ReadFile(path)
	if err != nil {
		result.AddError("spec_file",
			fmt.Sprintf("cannot read file: %v", err),
			"check file permissions")
		return result
	}

	// Warn if file is very small
	if len(data) < 50 {
		result.AddWarning("spec_file",
			"file is very short",
			"add more detail for better results")
	}

	// Warn if file is very large
	if len(data) > 50000 {
		result.AddWarning("spec_file",
			"file is very large (>50KB)",
			"consider breaking into smaller components")
	}

	return result
}

// ValidateOutputDirectory checks if output directory is usable
func ValidateOutputDirectory(path string) error {
	// Try to create directory
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("cannot create output directory: %w", err)
	}

	// Try to write a test file
	testFile := filepath.Join(path, ".agentic-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("cannot write to output directory: %w", err)
	}

	// Clean up test file
	os.Remove(testFile)

	return nil
}

// PrintValidationResult prints validation results
func PrintValidationResult(result *ValidationResult) {
	if len(result.Errors) > 0 {
		fmt.Println("❌ Validation Errors:")
		for _, err := range result.Errors {
			fmt.Printf("  • %s\n", err.Error())
		}
		fmt.Println()
	}

	if len(result.Warnings) > 0 {
		fmt.Println("⚠️  Warnings:")
		for _, warn := range result.Warnings {
			fmt.Printf("  • %s: %s\n", warn.Field, warn.Message)
			if warn.Fix != "" {
				fmt.Printf("    Suggestion: %s\n", warn.Fix)
			}
		}
		fmt.Println()
	}

	if result.IsValid() && len(result.Warnings) == 0 {
		fmt.Println("✓ All validations passed")
	}
}
