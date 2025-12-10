package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"upside-down-research.com/oss/agentic/internal/goap"
	goapactions "upside-down-research.com/oss/agentic/internal/goap/actions"
)

// This demo showcases the beautiful dance between GOFAI reasoning and LLM generation.
// Watch as classical AI planning orchestrates LLM content generation.

func main() {
	log.SetLevel(log.InfoLevel)

	fmt.Println()
	fmt.Println("🎭 GOAP Demo: Building a Feature with Quality Gates")
	fmt.Println()

	// Get working directory
	workDir, _ := os.Getwd()

	// PHASE 1: Set up the world
	initialState := goap.NewWorldState()
	initialState.Set("work_dir", workDir)
	initialState.Set("project_initialized", true)

	// PHASE 2: Create all our beautiful leaf node actions
	availableActions := createRichActionSet(workDir)

	// PHASE 3: Define our high-level goal
	goal := goap.NewGoal(
		"DeliverQualityFeature",
		"Implement a feature with full quality gates: code, tests, coverage, lint, review",
		goap.WorldState{
			"feature_designed":         true,
			"code_implemented":         true,
			"tests_written":            true,
			"go_tests_passed":          true,
			"target_coverage_achieved": true,
			"code_formatted":           true,
			"lint_passed":              true,
			"build_succeeded":          true,
			"quality_gates_passed":     true,
			"changes_committed":        true,
		},
		100.0, // High priority
	)

	// PHASE 4: Create the GOFAI planner (the reasoning monarch!)
	planner := goap.NewPlanner(availableActions)

	// PHASE 5: Create a simple refiner (in real system, would use LLM refiner)
	refiner := NewDemoRefiner()

	// PHASE 6: Set up persistence
	outputPath := "./output/goap-demo"
	os.MkdirAll(outputPath, 0755)
	persistence := goap.NewGraphPersistence(outputPath)

	// PHASE 7: Create the orchestrator - where GOFAI meets LLM
	orchestrator := goap.NewOrchestrator(planner, refiner, persistence, 5)

	// PHASE 8: Execute! Watch the magic happen
	ctx := context.Background()
	runID := fmt.Sprintf("demo-%d", time.Now().Unix())

	err := orchestrator.ExecuteGoal(ctx, initialState, goal, runID)

	if err != nil {
		log.Error("Demo execution failed", "error", err)
		os.Exit(1)
	}

	log.Info("🎉 Demo completed successfully!")
	fmt.Println()
	fmt.Println("📁 Check the plan graph at:", outputPath+"/"+runID+"/graph/")
	fmt.Println()
}

// createRichActionSet creates all our beautiful leaf nodes
func createRichActionSet(workDir string) []goap.Action {
	actions := []goap.Action{}

	// Design phase (simulated LLM generation)
	actions = append(actions, goap.NewSimpleAction(
		"DesignFeature",
		"Design the feature architecture (LLM generates design)",
		goap.WorldState{"project_initialized": true},
		goap.WorldState{"feature_designed": true},
		8.0, // LLM generation
		func(ctx context.Context, ws goap.WorldState) error {
			log.Info("🎨 LLM generating feature design...")
			time.Sleep(500 * time.Millisecond) // Simulate LLM call
			log.Info("✓ Feature design complete")
			return nil
		},
	))

	// Code implementation (LLM generation)
	actions = append(actions, goap.NewSimpleAction(
		"ImplementCode",
		"Implement the feature code (LLM generates code)",
		goap.WorldState{"feature_designed": true},
		goap.WorldState{"code_implemented": true, "code_written": true},
		12.0, // High complexity - LLM + quality gate
		func(ctx context.Context, ws goap.WorldState) error {
			log.Info("💻 LLM generating code implementation...")
			time.Sleep(800 * time.Millisecond) // Simulate LLM call
			log.Info("✓ Code implementation complete")
			return nil
		},
	))

	// Test generation (LLM generation)
	actions = append(actions, goap.NewSimpleAction(
		"WriteTests",
		"Write comprehensive tests (LLM generates tests)",
		goap.WorldState{"code_implemented": true},
		goap.WorldState{"tests_written": true},
		10.0, // LLM generation
		func(ctx context.Context, ws goap.WorldState) error {
			log.Info("🧪 LLM generating test cases...")
			time.Sleep(600 * time.Millisecond) // Simulate LLM call
			log.Info("✓ Tests written")
			return nil
		},
	))

	// Run tests (tool execution)
	actions = append(actions, goapactions.NewRunGoTestsAction(workDir, "./...", true))

	// Coverage improvement (iterative GOFAI + LLM)
	actionCtx := &goapactions.ActionContext{} // Simplified for demo
	actions = append(actions, goapactions.NewImproveCoverageAction(actionCtx, workDir, "./...", 70.0, 3))

	// Format code (tool execution)
	actions = append(actions, goapactions.NewGoFmtAction(workDir, []string{"./..."}))

	// Lint (tool execution)
	actions = append(actions, goap.NewSimpleAction(
		"LintCode",
		"Run linter on code",
		goap.WorldState{"code_written": true},
		goap.WorldState{"lint_passed": true},
		5.0,
		func(ctx context.Context, ws goap.WorldState) error {
			log.Info("🔍 Running linter...")
			time.Sleep(300 * time.Millisecond)
			log.Info("✓ Linting passed")
			return nil
		},
	))

	// Build (tool execution)
	actions = append(actions, goap.NewSimpleAction(
		"BuildProject",
		"Build the project",
		goap.WorldState{"code_written": true, "lint_passed": true},
		goap.WorldState{"build_succeeded": true},
		8.0,
		func(ctx context.Context, ws goap.WorldState) error {
			log.Info("🔨 Building project...")
			time.Sleep(700 * time.Millisecond)
			log.Info("✓ Build succeeded")
			return nil
		},
	))

	// Quality gates (GOFAI validation)
	gates := []goapactions.QualityGate{
		goapactions.TestsPassedGate(),
		goapactions.CoverageGate(70.0),
		goapactions.BuildSuccessGate(),
		goapactions.NoLintIssuesGate(),
	}

	actions = append(actions, goapactions.NewQualityGateAction(
		gates,
		goap.WorldState{
			"tests_written":   true,
			"build_succeeded": true,
			"lint_passed":     true,
		},
	))

	// Git commit (tool execution)
	actions = append(actions, goap.NewSimpleAction(
		"CommitChanges",
		"Commit changes to git",
		goap.WorldState{"quality_gates_passed": true},
		goap.WorldState{"changes_committed": true},
		3.0,
		func(ctx context.Context, ws goap.WorldState) error {
			log.Info("📝 Committing changes...")
			time.Sleep(200 * time.Millisecond)
			log.Info("✓ Changes committed")
			return nil
		},
	))

	return actions
}

// DemoRefiner is a simple refiner for the demo
type DemoRefiner struct{}

func NewDemoRefiner() *DemoRefiner {
	return &DemoRefiner{}
}

func (r *DemoRefiner) Refine(ctx context.Context, goal *goap.Goal, current goap.WorldState) ([]*goap.Goal, error) {
	// For demo, we'll keep it simple and not decompose
	// In real system, LLM would suggest decompositions
	return nil, nil
}

func (r *DemoRefiner) IsAtomic(goal *goap.Goal, current goap.WorldState) bool {
	// All goals are atomic in this demo
	return true
}
