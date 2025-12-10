package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"upside-down-research.com/oss/agentic/internal/goap"
)

// RetryAction wraps another action with retry logic
type RetryAction struct {
	*goap.BaseAction
	wrappedAction goap.Action
	maxRetries    int
	backoff       time.Duration
}

func NewRetryAction(action goap.Action, maxRetries int, backoff time.Duration) *RetryAction {
	return &RetryAction{
		BaseAction: goap.NewBaseAction(
			fmt.Sprintf("Retry[%s]", action.Name()),
			fmt.Sprintf("Execute %s with retry (max: %d)", action.Name(), maxRetries),
			action.Preconditions(),
			action.Effects(),
			action.Cost()+2.0, // Add retry overhead
		),
		wrappedAction: action,
		maxRetries:    maxRetries,
		backoff:       backoff,
	}
}

func (a *RetryAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for Retry[%s]", a.wrappedAction.Name())
	}

	var lastErr error
	for attempt := 0; attempt <= a.maxRetries; attempt++ {
		if attempt > 0 {
			log.Info("Retrying action", "action", a.wrappedAction.Name(), "attempt", attempt, "maxRetries", a.maxRetries)
			time.Sleep(a.backoff * time.Duration(attempt)) // Exponential backoff
		}

		err := a.wrappedAction.Execute(ctx, current)
		if err == nil {
			if attempt > 0 {
				log.Info("Action succeeded after retry", "action", a.wrappedAction.Name(), "attempts", attempt+1)
			}
			return nil
		}

		lastErr = err
		log.Warn("Action failed, will retry", "action", a.wrappedAction.Name(), "attempt", attempt+1, "error", err)
	}

	log.Error("Action failed after all retries", "action", a.wrappedAction.Name(), "maxRetries", a.maxRetries)
	return fmt.Errorf("action %s failed after %d retries: %w", a.wrappedAction.Name(), a.maxRetries, lastErr)
}

func (a *RetryAction) Clone() goap.Action {
	return NewRetryAction(a.wrappedAction.Clone(), a.maxRetries, a.backoff)
}

// FallbackAction tries primary action, falls back to alternative if it fails
type FallbackAction struct {
	*goap.BaseAction
	primaryAction   goap.Action
	fallbackAction  goap.Action
	usedFallback    bool
}

func NewFallbackAction(primary, fallback goap.Action) *FallbackAction {
	// Combine preconditions and effects
	preconditions := primary.Preconditions().Clone()
	for k, v := range fallback.Preconditions() {
		preconditions.Set(k, v)
	}

	effects := primary.Effects().Clone()
	for k, v := range fallback.Effects() {
		effects.Set(k, v)
	}

	return &FallbackAction{
		BaseAction: goap.NewBaseAction(
			fmt.Sprintf("Fallback[%s→%s]", primary.Name(), fallback.Name()),
			fmt.Sprintf("Try %s, fallback to %s", primary.Name(), fallback.Name()),
			preconditions,
			effects,
			primary.Cost()+1.0, // Small overhead
		),
		primaryAction:  primary,
		fallbackAction: fallback,
	}
}

func (a *FallbackAction) Execute(ctx context.Context, current goap.WorldState) error {
	log.Info("Attempting primary action", "action", a.primaryAction.Name())

	err := a.primaryAction.Execute(ctx, current)
	if err == nil {
		log.Info("Primary action succeeded")
		return nil
	}

	log.Warn("Primary action failed, using fallback", "primary", a.primaryAction.Name(), "fallback", a.fallbackAction.Name(), "error", err)

	a.usedFallback = true
	current.Set("used_fallback", true)
	current.Set("primary_failure_reason", err.Error())

	err = a.fallbackAction.Execute(ctx, current)
	if err != nil {
		return fmt.Errorf("both primary and fallback failed: primary=%v, fallback=%w", a.primaryAction.Name(), err)
	}

	log.Info("Fallback action succeeded")
	return nil
}

func (a *FallbackAction) Clone() goap.Action {
	return NewFallbackAction(a.primaryAction.Clone(), a.fallbackAction.Clone())
}

// ImproveCoverageAction iteratively improves test coverage
type ImproveCoverageAction struct {
	*goap.BaseAction
	ctx             *ActionContext
	workDir         string
	packagePath     string
	targetCoverage  float64
	maxIterations   int
}

func NewImproveCoverageAction(ctx *ActionContext, workDir, packagePath string, targetCoverage float64, maxIterations int) *ImproveCoverageAction {
	return &ImproveCoverageAction{
		BaseAction: goap.NewBaseAction(
			"ImproveCoverage",
			fmt.Sprintf("Improve test coverage to %.1f%%", targetCoverage),
			goap.WorldState{"code_written": true, "tests_written": true},
			goap.WorldState{"target_coverage_achieved": true},
			20.0, // Very high complexity - iterative LLM + testing
		),
		ctx:            ctx,
		workDir:        workDir,
		packagePath:    packagePath,
		targetCoverage: targetCoverage,
		maxIterations:  maxIterations,
	}
}

func (a *ImproveCoverageAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for ImproveCoverage")
	}

	log.Info("Starting iterative coverage improvement", "target", fmt.Sprintf("%.1f%%", a.targetCoverage), "maxIterations", a.maxIterations)

	for iteration := 1; iteration <= a.maxIterations; iteration++ {
		log.Info("Coverage improvement iteration", "iteration", iteration)

		// Run tests with coverage
		testAction := NewRunGoTestsAction(a.workDir, a.packagePath, true)
		err := testAction.Execute(ctx, current)
		if err != nil {
			log.Warn("Tests failed during coverage improvement", "iteration", iteration, "error", err)
			// Continue to try to add tests even if some fail
		}

		currentCoverage, ok := current.Get("test_coverage").(float64)
		if !ok {
			currentCoverage = 0.0
		}

		log.Info("Current coverage", "coverage", fmt.Sprintf("%.1f%%", currentCoverage), "target", fmt.Sprintf("%.1f%%", a.targetCoverage))

		if currentCoverage >= a.targetCoverage {
			log.Info("Target coverage achieved!", "coverage", fmt.Sprintf("%.1f%%", currentCoverage))
			current.Set("target_coverage_achieved", true)
			current.Set("final_coverage", currentCoverage)
			current.Set("coverage_iterations", iteration)
			return nil
		}

		// Use LLM to identify uncovered code and generate tests
		gap := a.targetCoverage - currentCoverage
		log.Info("Generating additional tests to close coverage gap", "gap", fmt.Sprintf("%.1f%%", gap))

		// This is a simplified version - in a real implementation,
		// you'd use the LLM to generate and add tests with a prompt like:
		// "The current test coverage is X%, but we need Y%. Generate tests..."
		log.Info("LLM would generate additional tests here (simplified in this implementation)",
			"iteration", iteration,
			"currentCoverage", currentCoverage,
			"target", a.targetCoverage,
			"packagePath", a.packagePath)

		// Simulate adding tests (in real implementation, would write test files)
		current.Set("coverage_improvement_attempt", iteration)

		// Small delay between iterations
		time.Sleep(500 * time.Millisecond)
	}

	currentCoverage, _ := current.Get("test_coverage").(float64)
	log.Warn("Max iterations reached without achieving target coverage",
		"final", fmt.Sprintf("%.1f%%", currentCoverage),
		"target", fmt.Sprintf("%.1f%%", a.targetCoverage))

	current.Set("target_coverage_achieved", false)
	current.Set("final_coverage", currentCoverage)
	current.Set("coverage_iterations", a.maxIterations)

	return fmt.Errorf("failed to achieve %.1f%% coverage after %d iterations (reached %.1f%%)",
		a.targetCoverage, a.maxIterations, currentCoverage)
}

func (a *ImproveCoverageAction) Clone() goap.Action {
	return NewImproveCoverageAction(a.ctx, a.workDir, a.packagePath, a.targetCoverage, a.maxIterations)
}

// TimeoutAction wraps an action with a timeout
type TimeoutAction struct {
	*goap.BaseAction
	wrappedAction goap.Action
	timeout       time.Duration
}

func NewTimeoutAction(action goap.Action, timeout time.Duration) *TimeoutAction {
	return &TimeoutAction{
		BaseAction: goap.NewBaseAction(
			fmt.Sprintf("Timeout[%s]", action.Name()),
			fmt.Sprintf("Execute %s with %v timeout", action.Name(), timeout),
			action.Preconditions(),
			action.Effects(),
			action.Cost(),
		),
		wrappedAction: action,
		timeout:       timeout,
	}
}

func (a *TimeoutAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for Timeout[%s]", a.wrappedAction.Name())
	}

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	log.Info("Executing with timeout", "action", a.wrappedAction.Name(), "timeout", a.timeout)

	done := make(chan error, 1)
	go func() {
		done <- a.wrappedAction.Execute(timeoutCtx, current)
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("action failed: %w", err)
		}
		log.Info("Action completed within timeout", "action", a.wrappedAction.Name())
		return nil

	case <-timeoutCtx.Done():
		log.Error("Action timed out", "action", a.wrappedAction.Name(), "timeout", a.timeout)
		return fmt.Errorf("action %s timed out after %v", a.wrappedAction.Name(), a.timeout)
	}
}

func (a *TimeoutAction) Clone() goap.Action {
	return NewTimeoutAction(a.wrappedAction.Clone(), a.timeout)
}
