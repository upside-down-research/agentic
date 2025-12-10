package actions

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"upside-down-research.com/oss/agentic/internal/goap"
)

// HumanReviewAction pauses execution for human approval
type HumanReviewAction struct {
	*goap.BaseAction
	reviewPrompt string
	reviewKey    string
}

func NewHumanReviewAction(reviewPrompt, reviewKey string, preconditions goap.WorldState) *HumanReviewAction {
	return &HumanReviewAction{
		BaseAction: goap.NewBaseAction(
			"HumanReview",
			fmt.Sprintf("Request human review: %s", reviewPrompt),
			preconditions,
			goap.WorldState{reviewKey + "_approved": true},
			0.0, // No cost - human action
		),
		reviewPrompt: reviewPrompt,
		reviewKey:    reviewKey,
	}
}

func (a *HumanReviewAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for HumanReview")
	}

	log.Info("Requesting human review", "prompt", a.reviewPrompt)

	fmt.Printf("\n" + strings.Repeat("=", 70) + "\n")
	fmt.Printf("🔍 HUMAN REVIEW REQUIRED\n")
	fmt.Printf(strings.Repeat("=", 70) + "\n")
	fmt.Printf("%s\n", a.reviewPrompt)
	fmt.Printf(strings.Repeat("-", 70) + "\n")
	fmt.Printf("Approve? (yes/no): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))

	if response == "yes" || response == "y" {
		current.Set(a.reviewKey+"_approved", true)
		current.Set(a.reviewKey+"_response", response)
		log.Info("Human review approved")
		return nil
	}

	current.Set(a.reviewKey+"_approved", false)
	current.Set(a.reviewKey+"_response", response)
	log.Warn("Human review rejected")
	return fmt.Errorf("human review rejected")
}

func (a *HumanReviewAction) Clone() goap.Action {
	return NewHumanReviewAction(a.reviewPrompt, a.reviewKey, a.Preconditions().Clone())
}

// AutoReviewAction performs automated code review using criteria
type AutoReviewAction struct {
	*goap.BaseAction
	reviewCriteria []string
	targetKey      string
}

func NewAutoReviewAction(reviewCriteria []string, targetKey string, preconditions goap.WorldState) *AutoReviewAction {
	return &AutoReviewAction{
		BaseAction: goap.NewBaseAction(
			"AutoReview",
			"Automated code review",
			preconditions,
			goap.WorldState{targetKey + "_reviewed": true},
			4.0,
		),
		reviewCriteria: reviewCriteria,
		targetKey:      targetKey,
	}
}

func (a *AutoReviewAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for AutoReview")
	}

	log.Info("Performing automated review", "criteria", len(a.reviewCriteria))

	passed := []string{}
	failed := []string{}

	for _, criterion := range a.reviewCriteria {
		log.Debug("Checking criterion", "criterion", criterion)

		// Check if criterion is met (simplified - in reality would analyze code)
		// For now, check if related state exists
		met := a.checkCriterion(criterion, current)

		if met {
			passed = append(passed, criterion)
		} else {
			failed = append(failed, criterion)
		}
	}

	current.Set(a.targetKey+"_review_passed", passed)
	current.Set(a.targetKey+"_review_failed", failed)

	if len(failed) > 0 {
		current.Set(a.targetKey+"_reviewed", false)
		log.Warn("Automated review found issues", "failed", len(failed))
		return fmt.Errorf("review failed %d criteria: %v", len(failed), failed)
	}

	current.Set(a.targetKey+"_reviewed", true)
	log.Info("Automated review passed", "criteria", len(passed))
	return nil
}

func (a *AutoReviewAction) checkCriterion(criterion string, current goap.WorldState) bool {
	// Simplified criterion checking
	criterion = strings.ToLower(criterion)

	if strings.Contains(criterion, "test") {
		return current.Get("tests_passed") == true
	}
	if strings.Contains(criterion, "build") {
		return current.Get("build_succeeded") == true
	}
	if strings.Contains(criterion, "lint") {
		return current.Get("lint_passed") == true
	}
	if strings.Contains(criterion, "format") {
		return current.Get("code_formatted") == true
	}

	// Default to true for unknown criteria
	return true
}

func (a *AutoReviewAction) Clone() goap.Action {
	return NewAutoReviewAction(a.reviewCriteria, a.targetKey, a.Preconditions().Clone())
}

// PeerReviewAction simulates or requests peer review
type PeerReviewAction struct {
	*goap.BaseAction
	reviewers []string
	codeKey   string
}

func NewPeerReviewAction(reviewers []string, codeKey string, preconditions goap.WorldState) *PeerReviewAction {
	return &PeerReviewAction{
		BaseAction: goap.NewBaseAction(
			"PeerReview",
			fmt.Sprintf("Request peer review from: %v", reviewers),
			preconditions,
			goap.WorldState{codeKey + "_peer_reviewed": true},
			10.0, // High complexity - requires human interaction
		),
		reviewers: reviewers,
		codeKey:   codeKey,
	}
}

func (a *PeerReviewAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for PeerReview")
	}

	log.Info("Requesting peer review", "reviewers", a.reviewers)

	// In a real implementation, this would:
	// - Create a pull request
	// - Request reviews from specified people
	// - Wait for approvals
	// - Check review comments

	fmt.Printf("\n" + strings.Repeat("=", 70) + "\n")
	fmt.Printf("👥 PEER REVIEW\n")
	fmt.Printf(strings.Repeat("=", 70) + "\n")
	fmt.Printf("Code review requested from: %v\n", a.reviewers)
	fmt.Printf("Key: %s\n", a.codeKey)
	fmt.Printf(strings.Repeat("-", 70) + "\n")
	fmt.Printf("Simulate approval? (yes/no): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))

	if response == "yes" || response == "y" {
		current.Set(a.codeKey+"_peer_reviewed", true)
		current.Set(a.codeKey+"_reviewers", a.reviewers)
		log.Info("Peer review approved")
		return nil
	}

	current.Set(a.codeKey+"_peer_reviewed", false)
	log.Warn("Peer review rejected")
	return fmt.Errorf("peer review rejected")
}

func (a *PeerReviewAction) Clone() goap.Action {
	return NewPeerReviewAction(a.reviewers, a.codeKey, a.Preconditions().Clone())
}

// QualityGateAction enforces multiple quality criteria
type QualityGateAction struct {
	*goap.BaseAction
	gates []QualityGate
}

type QualityGate struct {
	Name      string
	Condition func(goap.WorldState) bool
	Message   string
}

func NewQualityGateAction(gates []QualityGate, preconditions goap.WorldState) *QualityGateAction {
	return &QualityGateAction{
		BaseAction: goap.NewBaseAction(
			"QualityGate",
			fmt.Sprintf("Enforce %d quality gates", len(gates)),
			preconditions,
			goap.WorldState{"quality_gates_passed": true},
			3.0,
		),
		gates: gates,
	}
}

func (a *QualityGateAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for QualityGate")
	}

	log.Info("Checking quality gates", "count", len(a.gates))

	passed := []string{}
	failed := []string{}

	for _, gate := range a.gates {
		log.Debug("Checking gate", "name", gate.Name)

		if gate.Condition(current) {
			passed = append(passed, gate.Name)
			log.Debug("Gate passed", "name", gate.Name)
		} else {
			failed = append(failed, fmt.Sprintf("%s: %s", gate.Name, gate.Message))
			log.Warn("Gate failed", "name", gate.Name, "message", gate.Message)
		}
	}

	current.Set("quality_gates_passed_list", passed)
	current.Set("quality_gates_failed_list", failed)

	if len(failed) > 0 {
		current.Set("quality_gates_passed", false)
		log.Error("Quality gates failed", "failed", len(failed))
		return fmt.Errorf("quality gates failed:\n%s", strings.Join(failed, "\n"))
	}

	current.Set("quality_gates_passed", true)
	log.Info("All quality gates passed", "count", len(passed))
	return nil
}

func (a *QualityGateAction) Clone() goap.Action {
	return NewQualityGateAction(a.gates, a.Preconditions().Clone())
}

// Common quality gate conditions

func TestsPassedGate() QualityGate {
	return QualityGate{
		Name: "TestsPassed",
		Condition: func(ws goap.WorldState) bool {
			return ws.Get("tests_passed") == true
		},
		Message: "All tests must pass",
	}
}

func CoverageGate(minCoverage float64) QualityGate {
	return QualityGate{
		Name: fmt.Sprintf("Coverage>=%.1f%%", minCoverage),
		Condition: func(ws goap.WorldState) bool {
			if cov, ok := ws.Get("test_coverage").(float64); ok {
				return cov >= minCoverage
			}
			return false
		},
		Message: fmt.Sprintf("Test coverage must be >= %.1f%%", minCoverage),
	}
}

func BuildSuccessGate() QualityGate {
	return QualityGate{
		Name: "BuildSuccess",
		Condition: func(ws goap.WorldState) bool {
			return ws.Get("build_succeeded") == true
		},
		Message: "Build must succeed",
	}
}

func NoLintIssuesGate() QualityGate {
	return QualityGate{
		Name: "NoLintIssues",
		Condition: func(ws goap.WorldState) bool {
			return ws.Get("lint_passed") == true
		},
		Message: "No linting issues allowed",
	}
}
