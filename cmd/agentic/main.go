package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/log"
	"upside-down-research.com/oss/agentic/internal/commands"
)

var CLI struct {
	Generate commands.GenerateCommand `cmd:"" help:"Generate code from specification" default:"withargs"`
	Doctor   commands.DoctorCommand   `cmd:"" help:"Run system diagnostics"`
	Validate commands.ValidateCommand `cmd:"" help:"Validate a specification file"`
	Estimate commands.EstimateCommand `cmd:"" help:"Estimate cost and time"`
	Config   commands.ConfigCommand   `cmd:"" help:"Manage configuration"`
}

const banner = `
   _                    _   _
  /_\   __ _  ___ _ __ | |_(_) ___
 //_\\ / _' |/ _ \ '_ \| __| |/ __|
/  _  \ (_| |  __/ | | | |_| | (__
\_/ \_/\__, |\___|_| |_|\__|_|\___|
       |___/

AI Software Engineer - Function Over Form
`

func main() {
	log.SetLevel(log.InfoLevel)

	ctx := kong.Parse(&CLI,
		kong.Name("agentic"),
		kong.Description("Agentic - AI Software Engineer\n\nGenerate code from specifications using LLMs with quality gates."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: false,
			Summary: true,
		}),
	)

	// Show banner for main help
	if ctx.Command() == "" {
		fmt.Println(banner)
		fmt.Println("Quick start:")
		fmt.Println("  $ agentic config init           # Create config file")
		fmt.Println("  $ agentic doctor                # Verify setup")
		fmt.Println("  $ agentic validate spec.in      # Check specification")
		fmt.Println("  $ agentic estimate spec.in      # See cost estimate")
		fmt.Println("  $ agentic generate spec.in      # Generate code")
		fmt.Println()
		fmt.Println("Run 'agentic --help' for all commands")
		os.Exit(0)
	}

	err := ctx.Run()
	if err != nil {
		log.Error("Command failed", "error", err)
		os.Exit(1)
	}
}
