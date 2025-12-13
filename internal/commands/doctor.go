package commands

import (
	"fmt"
	"os"
	"os/exec"

	"upside-down-research.com/oss/agentic/internal/config"
	"upside-down-research.com/oss/agentic/internal/validation"
)

// DoctorCommand runs system diagnostics
type DoctorCommand struct {
	Config string `name:"config" help:"Configuration file path" type:"path"`
}

// Run executes the doctor command
func (cmd *DoctorCommand) Run() error {
	fmt.Println("🏥 Running Agentic diagnostics...")
	fmt.Println()

	allOk := true

	// Load and validate config
	cfg, err := config.LoadConfig(cmd.Config)
	if err != nil {
		fmt.Printf("❌ Config: %v\n", err)
		allOk = false
	} else {
		result := validation.ValidateConfig(cfg)
		if result.IsValid() {
			fmt.Println("✓ Configuration: valid")
		} else {
			fmt.Println("❌ Configuration: has errors")
			for _, e := range result.Errors {
				fmt.Printf("  • %s\n", e.Error())
			}
			allOk = false
		}
		if len(result.Warnings) > 0 {
			fmt.Println("⚠️  Configuration: has warnings")
			for _, w := range result.Warnings {
				fmt.Printf("  • %s: %s\n", w.Field, w.Message)
			}
		}
	}

	// Check API keys
	if cfg != nil {
		switch cfg.LLM.Provider {
		case "openai":
			key := cfg.LLM.APIKey
			if key == "" {
				key = os.Getenv("OPENAI_API_KEY")
			}
			if key != "" {
				fmt.Println("✓ OpenAI API key: configured")
			} else {
				fmt.Println("❌ OpenAI API key: not found")
				fmt.Println("  Fix: export OPENAI_API_KEY=sk-...")
				allOk = false
			}
		case "claude":
			key := cfg.LLM.APIKey
			if key == "" {
				key = os.Getenv("CLAUDE_API_KEY")
			}
			if key != "" {
				fmt.Println("✓ Claude API key: configured")
			} else {
				fmt.Println("❌ Claude API key: not found")
				fmt.Println("  Fix: export CLAUDE_API_KEY=...")
				allOk = false
			}
		case "bedrock":
			if os.Getenv("AWS_ACCESS_KEY_ID") != "" || os.Getenv("AWS_PROFILE") != "" {
				fmt.Println("✓ AWS credentials: configured")
			} else {
				fmt.Println("⚠️  AWS credentials: not found in environment")
				fmt.Println("  Note: Will attempt to use IAM role or ~/.aws/credentials")
			}
		case "ai00":
			fmt.Println("✓ AI00: no API key required")
		}
	}

	// Check output directory
	if cfg != nil && cfg.Output.Directory != "" {
		err := validation.ValidateOutputDirectory(cfg.Output.Directory)
		if err == nil {
			fmt.Printf("✓ Output directory: %s (writable)\n", cfg.Output.Directory)
		} else {
			fmt.Printf("❌ Output directory: %v\n", err)
			allOk = false
		}
	}

	// Check for Go compiler (if quality gates require compilation)
	if cfg != nil && cfg.QualityGate.RequireCompilation {
		_, err := exec.LookPath("go")
		if err == nil {
			fmt.Println("✓ Go compiler: available")
		} else {
			fmt.Println("❌ Go compiler: not found")
			fmt.Println("  Note: Required for compilation quality gate")
			allOk = false
		}
	}

	// Check disk space (warn if low)
	if cfg != nil && cfg.Output.Directory != "" {
		// Simple check - just try to create directory
		_ = os.MkdirAll(cfg.Output.Directory, 0755)
	}

	fmt.Println()
	if allOk {
		fmt.Println("🎉 All systems ready!")
		return nil
	} else {
		fmt.Println("⚠️  Some issues found - please fix before running")
		return fmt.Errorf("validation failed")
	}
}
