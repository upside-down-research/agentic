package commands

import (
	"fmt"

	"upside-down-research.com/oss/agentic/internal/validation"
)

// ValidateCommand validates a specification file
type ValidateCommand struct {
	SpecFile string `arg:"" name:"spec" help:"Specification file to validate" type:"path"`
}

// Run executes the validate command
func (cmd *ValidateCommand) Run() error {
	fmt.Printf("📋 Validating specification file: %s\n\n", cmd.SpecFile)

	result := validation.ValidateSpecFile(cmd.SpecFile)
	validation.PrintValidationResult(result)

	if !result.IsValid() {
		return fmt.Errorf("validation failed")
	}

	return nil
}
