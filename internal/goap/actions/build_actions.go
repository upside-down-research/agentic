package actions

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"upside-down-research.com/oss/agentic/internal/goap"
)

// BuildAction compiles code
type BuildAction struct {
	*goap.BaseAction
	workDir      string
	buildCommand string
	buildArgs    []string
	outputPath   string
}

func NewBuildAction(workDir, buildCommand string, args []string, outputPath string) *BuildAction {
	return &BuildAction{
		BaseAction: goap.NewBaseAction(
			"Build",
			fmt.Sprintf("Build project: %s %v", buildCommand, args),
			goap.WorldState{"code_written": true},
			goap.WorldState{"build_succeeded": true},
			8.0, // Medium-high complexity
		),
		workDir:      workDir,
		buildCommand: buildCommand,
		buildArgs:    args,
		outputPath:   outputPath,
	}
}

func (a *BuildAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for Build")
	}

	log.Info("Building project", "command", a.buildCommand)

	start := time.Now()
	cmd := exec.CommandContext(ctx, a.buildCommand, a.buildArgs...)
	cmd.Dir = a.workDir

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	current.Set("build_executed", true)
	current.Set("build_output", string(output))
	current.Set("build_duration", duration.Seconds())

	if err != nil {
		current.Set("build_succeeded", false)
		current.Set("build_errors", string(output))
		log.Error("Build failed", "error", err, "duration", duration)
		return fmt.Errorf("build failed: %w\nOutput:\n%s", err, output)
	}

	current.Set("build_succeeded", true)
	if a.outputPath != "" {
		current.Set("build_output_path", a.outputPath)
	}

	log.Info("Build succeeded", "duration", duration)
	return nil
}

func (a *BuildAction) Clone() goap.Action {
	return NewBuildAction(a.workDir, a.buildCommand, a.buildArgs, a.outputPath)
}

// GoBuildAction builds a Go project
type GoBuildAction struct {
	*goap.BaseAction
	workDir    string
	outputPath string
	mainPath   string
}

func NewGoBuildAction(workDir, outputPath, mainPath string) *GoBuildAction {
	return &GoBuildAction{
		BaseAction: goap.NewBaseAction(
			"GoBuild",
			fmt.Sprintf("Build Go binary: %s -> %s", mainPath, outputPath),
			goap.WorldState{"code_written": true},
			goap.WorldState{"go_build_succeeded": true},
			8.0,
		),
		workDir:    workDir,
		outputPath: outputPath,
		mainPath:   mainPath,
	}
}

func (a *GoBuildAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for GoBuild")
	}

	log.Info("Building Go project", "main", a.mainPath, "output", a.outputPath)

	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", a.outputPath, a.mainPath)
	cmd.Dir = a.workDir

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	current.Set("go_build_executed", true)
	current.Set("build_duration", duration.Seconds())

	if err != nil {
		current.Set("go_build_succeeded", false)
		current.Set("build_errors", string(output))
		log.Error("Go build failed", "error", err, "duration", duration)
		return fmt.Errorf("go build failed: %w\nOutput:\n%s", err, output)
	}

	current.Set("go_build_succeeded", true)
	current.Set("binary_path", a.outputPath)

	log.Info("Go build succeeded", "duration", duration, "binary", a.outputPath)
	return nil
}

func (a *GoBuildAction) Clone() goap.Action {
	return NewGoBuildAction(a.workDir, a.outputPath, a.mainPath)
}

// LintAction runs code linters
type LintAction struct {
	*goap.BaseAction
	workDir string
	linter  string
	paths   []string
}

func NewLintAction(workDir, linter string, paths []string) *LintAction {
	return &LintAction{
		BaseAction: goap.NewBaseAction(
			"Lint",
			fmt.Sprintf("Run linter: %s on %v", linter, paths),
			goap.WorldState{"code_written": true},
			goap.WorldState{"lint_passed": true},
			5.0,
		),
		workDir: workDir,
		linter:  linter,
		paths:   paths,
	}
}

func (a *LintAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for Lint")
	}

	log.Info("Running linter", "linter", a.linter, "paths", a.paths)

	args := append([]string{}, a.paths...)
	cmd := exec.CommandContext(ctx, a.linter, args...)
	cmd.Dir = a.workDir

	output, err := cmd.CombinedOutput()

	current.Set("lint_executed", true)
	current.Set("lint_output", string(output))

	if err != nil {
		current.Set("lint_passed", false)
		current.Set("lint_issues", string(output))
		log.Warn("Linter found issues", "output", string(output))
		return fmt.Errorf("linter found issues:\n%s", output)
	}

	current.Set("lint_passed", true)
	log.Info("Linting passed")
	return nil
}

func (a *LintAction) Clone() goap.Action {
	return NewLintAction(a.workDir, a.linter, a.paths)
}

// GoFmtAction formats Go code
type GoFmtAction struct {
	*goap.BaseAction
	workDir string
	paths   []string
}

func NewGoFmtAction(workDir string, paths []string) *GoFmtAction {
	return &GoFmtAction{
		BaseAction: goap.NewBaseAction(
			"GoFmt",
			"Format Go code with gofmt",
			goap.WorldState{"code_written": true},
			goap.WorldState{"code_formatted": true},
			2.0, // Low complexity
		),
		workDir: workDir,
		paths:   paths,
	}
}

func (a *GoFmtAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for GoFmt")
	}

	log.Info("Formatting Go code", "paths", a.paths)

	args := append([]string{"-w"}, a.paths...)
	cmd := exec.CommandContext(ctx, "gofmt", args...)
	cmd.Dir = a.workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gofmt failed: %w\nOutput:\n%s", err, output)
	}

	current.Set("code_formatted", true)
	log.Info("Code formatted successfully")
	return nil
}

func (a *GoFmtAction) Clone() goap.Action {
	return NewGoFmtAction(a.workDir, a.paths)
}

// CompileCheckAction checks if code compiles without building
type CompileCheckAction struct {
	*goap.BaseAction
	workDir string
	pkgPath string
}

func NewCompileCheckAction(workDir, pkgPath string) *CompileCheckAction {
	return &CompileCheckAction{
		BaseAction: goap.NewBaseAction(
			"CompileCheck",
			fmt.Sprintf("Check compilation: %s", pkgPath),
			goap.WorldState{"code_written": true},
			goap.WorldState{"compile_check_passed": true},
			4.0,
		),
		workDir: workDir,
		pkgPath: pkgPath,
	}
}

func (a *CompileCheckAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for CompileCheck")
	}

	log.Info("Checking compilation", "package", a.pkgPath)

	cmd := exec.CommandContext(ctx, "go", "build", "-o", "/dev/null", a.pkgPath)
	cmd.Dir = a.workDir

	output, err := cmd.CombinedOutput()

	current.Set("compile_check_executed", true)

	if err != nil {
		current.Set("compile_check_passed", false)
		current.Set("compile_errors", parseCompileErrors(string(output)))
		log.Error("Compilation check failed", "errors", string(output))
		return fmt.Errorf("compilation errors:\n%s", output)
	}

	current.Set("compile_check_passed", true)
	log.Info("Compilation check passed")
	return nil
}

func (a *CompileCheckAction) Clone() goap.Action {
	return NewCompileCheckAction(a.workDir, a.pkgPath)
}

// parseCompileErrors extracts structured error information from compiler output
func parseCompileErrors(output string) []string {
	lines := strings.Split(output, "\n")
	errors := []string{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			errors = append(errors, line)
		}
	}

	return errors
}
