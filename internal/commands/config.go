package commands

import (
	"fmt"
	"os"

	"upside-down-research.com/oss/agentic/internal/config"
)

// ConfigCommand manages configuration
type ConfigCommand struct {
	Init ConfigInitCommand `cmd:"" help:"Create a new configuration file"`
}

// ConfigInitCommand creates a new config file
type ConfigInitCommand struct {
	Output string `name:"output" help:"Output path for config file" default:"agentic.yaml"`
	Force  bool   `name:"force" help:"Overwrite existing file"`
}

// Run executes the config init command
func (cmd *ConfigInitCommand) Run() error {
	// Check if file exists
	if _, err := os.Stat(cmd.Output); err == nil && !cmd.Force {
		return fmt.Errorf("config file already exists: %s (use --force to overwrite)", cmd.Output)
	}

	// Write example config
	err := os.WriteFile(cmd.Output, []byte(config.ExampleConfig()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("✓ Created configuration file: %s\n", cmd.Output)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit the config file to set your API keys")
	fmt.Println("  2. Run 'agentic doctor' to verify configuration")
	fmt.Println("  3. Run 'agentic generate <spec-file>' to start coding")

	return nil
}
