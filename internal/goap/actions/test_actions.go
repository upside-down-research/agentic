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

// RunTestsAction executes tests for a project
type RunTestsAction struct {
	*goap.BaseAction
	workDir     string
	testCommand string
	testArgs    []string
}

func NewRunTestsAction(workDir, testCommand string, args []string) *RunTestsAction {
	return &RunTestsAction{
		BaseAction: goap.NewBaseAction(
			"RunTests",
			fmt.Sprintf("Execute test suite: %s %v", testCommand, args),
			goap.WorldState{"code_implemented": true},
			goap.WorldState{"tests_executed": true},
			7.0, // Medium-high complexity
		),
		workDir:     workDir,
		testCommand: testCommand,
		testArgs:    args,
	}
}

func (a *RunTestsAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for RunTests")
	}

	log.Info("Running tests", "command", a.testCommand, "args", a.testArgs)

	start := time.Now()
	cmd := exec.CommandContext(ctx, a.testCommand, a.testArgs...)
	cmd.Dir = a.workDir

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	current.Set("tests_executed", true)
	current.Set("test_output", string(output))
	current.Set("test_duration", duration.Seconds())

	if err != nil {
		current.Set("tests_passed", false)
		current.Set("test_error", err.Error())
		log.Error("Tests failed", "error", err, "duration", duration)
		return fmt.Errorf("tests failed: %w\nOutput:\n%s", err, output)
	}

	current.Set("tests_passed", true)
	log.Info("Tests passed", "duration", duration)
	return nil
}

func (a *RunTestsAction) Clone() goap.Action {
	return NewRunTestsAction(a.workDir, a.testCommand, a.testArgs)
}

// RunGoTestsAction runs Go tests with coverage
type RunGoTestsAction struct {
	*goap.BaseAction
	workDir      string
	packagePath  string
	withCoverage bool
}

func NewRunGoTestsAction(workDir, packagePath string, withCoverage bool) *RunGoTestsAction {
	desc := fmt.Sprintf("Run Go tests for %s", packagePath)
	if withCoverage {
		desc += " with coverage"
	}

	return &RunGoTestsAction{
		BaseAction: goap.NewBaseAction(
			"RunGoTests",
			desc,
			goap.WorldState{"code_written": true},
			goap.WorldState{"go_tests_passed": true},
			8.0,
		),
		workDir:      workDir,
		packagePath:  packagePath,
		withCoverage: withCoverage,
	}
}

func (a *RunGoTestsAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for RunGoTests")
	}

	args := []string{"test", "-v"}
	if a.withCoverage {
		args = append(args, "-cover")
	}
	args = append(args, a.packagePath)

	log.Info("Running Go tests", "package", a.packagePath, "coverage", a.withCoverage)

	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = a.workDir

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	current.Set("go_tests_executed", true)
	current.Set("test_output", string(output))
	current.Set("test_duration", duration.Seconds())

	if err != nil {
		current.Set("go_tests_passed", false)
		log.Error("Go tests failed", "error", err, "duration", duration)
		return fmt.Errorf("go tests failed: %w\nOutput:\n%s", err, output)
	}

	// Parse coverage if present
	if a.withCoverage {
		coverage := parseCoverage(string(output))
		current.Set("test_coverage", coverage)
		log.Info("Go tests passed", "duration", duration, "coverage", fmt.Sprintf("%.1f%%", coverage))
	} else {
		log.Info("Go tests passed", "duration", duration)
	}

	current.Set("go_tests_passed", true)
	return nil
}

func (a *RunGoTestsAction) Clone() goap.Action {
	return NewRunGoTestsAction(a.workDir, a.packagePath, a.withCoverage)
}

// BenchmarkAction runs performance benchmarks
type BenchmarkAction struct {
	*goap.BaseAction
	workDir     string
	benchTarget string
}

func NewBenchmarkAction(workDir, benchTarget string) *BenchmarkAction {
	return &BenchmarkAction{
		BaseAction: goap.NewBaseAction(
			"RunBenchmarks",
			fmt.Sprintf("Run benchmarks: %s", benchTarget),
			goap.WorldState{"code_implemented": true},
			goap.WorldState{"benchmarks_run": true},
			6.0,
		),
		workDir:     workDir,
		benchTarget: benchTarget,
	}
}

func (a *BenchmarkAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for RunBenchmarks")
	}

	log.Info("Running benchmarks", "target", a.benchTarget)

	cmd := exec.CommandContext(ctx, "go", "test", "-bench", a.benchTarget, "-benchmem")
	cmd.Dir = a.workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		current.Set("benchmarks_run", true)
		current.Set("benchmark_failed", true)
		return fmt.Errorf("benchmarks failed: %w\nOutput:\n%s", err, output)
	}

	current.Set("benchmarks_run", true)
	current.Set("benchmark_output", string(output))
	current.Set("benchmark_failed", false)

	log.Info("Benchmarks completed")
	return nil
}

func (a *BenchmarkAction) Clone() goap.Action {
	return NewBenchmarkAction(a.workDir, a.benchTarget)
}

// parseCoverage extracts coverage percentage from go test output
func parseCoverage(output string) float64 {
	// Look for "coverage: XX.X% of statements"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "coverage:") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "coverage:" && i+1 < len(parts) {
					coverageStr := strings.TrimSuffix(parts[i+1], "%")
					var coverage float64
					fmt.Sscanf(coverageStr, "%f", &coverage)
					return coverage
				}
			}
		}
	}
	return 0.0
}
