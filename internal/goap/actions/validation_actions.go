package actions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"upside-down-research.com/oss/agentic/internal/goap"
)

// ValidateStateAction asserts that certain world state conditions are met
type ValidateStateAction struct {
	*goap.BaseAction
	requiredState goap.WorldState
	validationMsg string
}

func NewValidateStateAction(requiredState goap.WorldState, validationMsg string) *ValidateStateAction {
	return &ValidateStateAction{
		BaseAction: goap.NewBaseAction(
			"ValidateState",
			validationMsg,
			goap.WorldState{},
			goap.WorldState{"validation_passed": true},
			1.0, // Low complexity
		),
		requiredState: requiredState,
		validationMsg: validationMsg,
	}
}

func (a *ValidateStateAction) Execute(ctx context.Context, current goap.WorldState) error {
	log.Info("Validating state", "message", a.validationMsg)

	mismatches := []string{}
	for key, expectedValue := range a.requiredState {
		actualValue := current.Get(key)
		if actualValue != expectedValue {
			mismatches = append(mismatches, fmt.Sprintf("%s: expected %v, got %v", key, expectedValue, actualValue))
		}
	}

	if len(mismatches) > 0 {
		current.Set("validation_passed", false)
		current.Set("validation_errors", mismatches)
		log.Error("Validation failed", "mismatches", mismatches)
		return fmt.Errorf("state validation failed:\n%s", strings.Join(mismatches, "\n"))
	}

	current.Set("validation_passed", true)
	log.Info("Validation passed")
	return nil
}

func (a *ValidateStateAction) Clone() goap.Action {
	return NewValidateStateAction(a.requiredState.Clone(), a.validationMsg)
}

// FileExistsAction validates that specific files exist
type FileExistsAction struct {
	*goap.BaseAction
	filePaths []string
}

func NewFileExistsAction(filePaths []string) *FileExistsAction {
	return &FileExistsAction{
		BaseAction: goap.NewBaseAction(
			"ValidateFilesExist",
			fmt.Sprintf("Validate files exist: %v", filePaths),
			goap.WorldState{},
			goap.WorldState{"files_validated": true},
			1.0,
		),
		filePaths: filePaths,
	}
}

func (a *FileExistsAction) Execute(ctx context.Context, current goap.WorldState) error {
	log.Info("Validating file existence", "count", len(a.filePaths))

	missing := []string{}
	for _, filePath := range a.filePaths {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			missing = append(missing, filePath)
		}
	}

	if len(missing) > 0 {
		current.Set("files_validated", false)
		current.Set("missing_files", missing)
		log.Error("Missing files", "files", missing)
		return fmt.Errorf("missing files: %v", missing)
	}

	current.Set("files_validated", true)
	log.Info("All files exist")
	return nil
}

func (a *FileExistsAction) Clone() goap.Action {
	return NewFileExistsAction(a.filePaths)
}

// CoverageThresholdAction validates test coverage meets threshold
type CoverageThresholdAction struct {
	*goap.BaseAction
	minCoverage float64
}

func NewCoverageThresholdAction(minCoverage float64) *CoverageThresholdAction {
	return &CoverageThresholdAction{
		BaseAction: goap.NewBaseAction(
			"ValidateCoverage",
			fmt.Sprintf("Ensure test coverage >= %.1f%%", minCoverage),
			goap.WorldState{"tests_executed": true},
			goap.WorldState{"coverage_threshold_met": true},
			2.0,
		),
		minCoverage: minCoverage,
	}
}

func (a *CoverageThresholdAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for ValidateCoverage")
	}

	coverage, ok := current.Get("test_coverage").(float64)
	if !ok {
		return fmt.Errorf("test_coverage not found in world state")
	}

	log.Info("Validating coverage", "actual", fmt.Sprintf("%.1f%%", coverage), "minimum", fmt.Sprintf("%.1f%%", a.minCoverage))

	if coverage < a.minCoverage {
		current.Set("coverage_threshold_met", false)
		current.Set("coverage_gap", a.minCoverage-coverage)
		log.Warn("Coverage below threshold", "gap", fmt.Sprintf("%.1f%%", a.minCoverage-coverage))
		return fmt.Errorf("coverage %.1f%% below threshold %.1f%%", coverage, a.minCoverage)
	}

	current.Set("coverage_threshold_met", true)
	log.Info("Coverage threshold met", "coverage", fmt.Sprintf("%.1f%%", coverage))
	return nil
}

func (a *CoverageThresholdAction) Clone() goap.Action {
	return NewCoverageThresholdAction(a.minCoverage)
}

// DirectoryStructureAction validates expected directory structure
type DirectoryStructureAction struct {
	*goap.BaseAction
	basePath         string
	requiredDirs     []string
	requiredPatterns []string
}

func NewDirectoryStructureAction(basePath string, requiredDirs, requiredPatterns []string) *DirectoryStructureAction {
	return &DirectoryStructureAction{
		BaseAction: goap.NewBaseAction(
			"ValidateDirectoryStructure",
			"Validate project directory structure",
			goap.WorldState{},
			goap.WorldState{"structure_validated": true},
			2.0,
		),
		basePath:         basePath,
		requiredDirs:     requiredDirs,
		requiredPatterns: requiredPatterns,
	}
}

func (a *DirectoryStructureAction) Execute(ctx context.Context, current goap.WorldState) error {
	log.Info("Validating directory structure", "basePath", a.basePath)

	missing := []string{}

	// Check required directories
	for _, dir := range a.requiredDirs {
		fullPath := filepath.Join(a.basePath, dir)
		info, err := os.Stat(fullPath)
		if os.IsNotExist(err) || !info.IsDir() {
			missing = append(missing, dir)
		}
	}

	// Check required file patterns
	for _, pattern := range a.requiredPatterns {
		matches, err := filepath.Glob(filepath.Join(a.basePath, pattern))
		if err != nil || len(matches) == 0 {
			missing = append(missing, pattern)
		}
	}

	if len(missing) > 0 {
		current.Set("structure_validated", false)
		current.Set("missing_structure", missing)
		log.Error("Directory structure validation failed", "missing", missing)
		return fmt.Errorf("missing structure elements: %v", missing)
	}

	current.Set("structure_validated", true)
	log.Info("Directory structure validated")
	return nil
}

func (a *DirectoryStructureAction) Clone() goap.Action {
	return NewDirectoryStructureAction(a.basePath, a.requiredDirs, a.requiredPatterns)
}

// NoErrorsAction validates that no errors are present in world state
type NoErrorsAction struct {
	*goap.BaseAction
	errorKeys []string
}

func NewNoErrorsAction(errorKeys []string) *NoErrorsAction {
	return &NoErrorsAction{
		BaseAction: goap.NewBaseAction(
			"ValidateNoErrors",
			"Ensure no errors in process",
			goap.WorldState{},
			goap.WorldState{"no_errors_validated": true},
			1.0,
		),
		errorKeys: errorKeys,
	}
}

func (a *NoErrorsAction) Execute(ctx context.Context, current goap.WorldState) error {
	log.Info("Validating no errors present")

	foundErrors := []string{}
	for _, key := range a.errorKeys {
		if val := current.Get(key); val != nil {
			if errStr, ok := val.(string); ok && errStr != "" {
				foundErrors = append(foundErrors, fmt.Sprintf("%s: %s", key, errStr))
			}
		}
	}

	if len(foundErrors) > 0 {
		current.Set("no_errors_validated", false)
		current.Set("found_errors", foundErrors)
		log.Error("Errors found", "errors", foundErrors)
		return fmt.Errorf("errors present:\n%s", strings.Join(foundErrors, "\n"))
	}

	current.Set("no_errors_validated", true)
	log.Info("No errors found")
	return nil
}

func (a *NoErrorsAction) Clone() goap.Action {
	return NewNoErrorsAction(a.errorKeys)
}
