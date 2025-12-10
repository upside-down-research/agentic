package goap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

// Orchestrator is the master conductor of the GOAP system.
// It embodies the philosophy: GOFAI for reasoning, LLMs for generation.
//
// The orchestrator uses classical AI (GOAP, A*, graph search) for ALL
// deliberative reasoning, planning, and decision-making. LLMs are used
// ONLY as content generators and goal decomposers, never for planning logic.
type Orchestrator struct {
	planner       *Planner
	refiner       GoalRefiner
	persistence   *GraphPersistence
	visualization *Visualizer
	maxDepth      int
}

// NewOrchestrator creates the master orchestrator.
// This is where GOFAI reasoning meets LLM generation.
func NewOrchestrator(planner *Planner, refiner GoalRefiner, persistence *GraphPersistence, maxDepth int) *Orchestrator {
	return &Orchestrator{
		planner:       planner,
		refiner:       refiner,
		persistence:   persistence,
		visualization: NewVisualizer(),
		maxDepth:      maxDepth,
	}
}

// ExecuteGoal is the main entry point for goal-oriented planning and execution.
// It showcases the beautiful dance between GOFAI reasoning and LLM generation:
//
// 1. GOFAI Planning: Uses hierarchical planning with A* search (classic AI)
// 2. LLM Decomposition: Uses LLM to suggest goal breakdowns (generation)
// 3. GOFAI Selection: Uses A* and cost functions to select optimal path (classic AI)
// 4. LLM Execution: Uses LLM to generate content at leaf nodes (generation)
// 5. GOFAI Orchestration: Manages execution flow and state (classic AI)
func (o *Orchestrator) ExecuteGoal(ctx context.Context, initialState WorldState, goal *Goal, runID string) error {
	o.visualization.ShowBanner()
	o.visualization.ShowPhilosophy()

	log.Info("🎯 Orchestrator starting",
		"goal", goal.Name(),
		"priority", goal.Priority(),
		"runID", runID)

	// PHASE 1: GOFAI REASONING - Hierarchical Planning
	log.Info("📐 PHASE 1: GOFAI REASONING - Hierarchical Planning")
	o.visualization.ShowPhase("GOFAI Planning & Reasoning", "Using classic AI to reason about goals")

	hierarchicalPlanner := NewHierarchicalPlanner(o.planner, o.refiner, o.maxDepth)

	start := time.Now()
	plan, err := hierarchicalPlanner.PlanHierarchical(ctx, initialState, goal)
	planDuration := time.Since(start)

	if err != nil {
		return fmt.Errorf("GOFAI planning failed: %w", err)
	}

	log.Info("✓ GOFAI planning complete",
		"duration", planDuration,
		"nodes", o.countNodes(plan),
		"depth", plan.Depth)

	o.visualization.ShowPlanSummary(plan, planDuration)

	// PHASE 2: GOFAI PERSISTENCE - Graph Database
	log.Info("💾 PHASE 2: GOFAI PERSISTENCE - Storing Plan Graph")
	o.visualization.ShowPhase("Plan Persistence", "Converting plan to graph database for minimal context")

	graph := BuildGraphFromPlan(plan, runID)
	err = o.persistence.SaveGraph(graph, runID)
	if err != nil {
		return fmt.Errorf("failed to persist plan: %w", err)
	}

	log.Info("✓ Plan graph persisted",
		"nodes", len(graph.Nodes),
		"maxDepth", graph.Metadata.MaxDepth)

	// PHASE 3: GOFAI EXECUTION with LLM GENERATION
	log.Info("⚡ PHASE 3: GOFAI EXECUTION with LLM GENERATION")
	o.visualization.ShowPhase("Execution", "GOFAI orchestrates, LLM generates content")

	executor := NewGraphExecutor(o.persistence, runID)

	// Register all actions from the plan
	allActions := plan.AllActions()
	executor.RegisterActions(allActions)

	// Execute with progress tracking
	err = o.executeWithProgress(ctx, executor, initialState, runID)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	// PHASE 4: RESULTS
	log.Info("📊 PHASE 4: RESULTS")
	status, _ := executor.GetGraphStatus()
	o.visualization.ShowResults(status)

	return nil
}

// executeWithProgress executes the plan with beautiful progress visualization
func (o *Orchestrator) executeWithProgress(ctx context.Context, executor *GraphExecutor, initialState WorldState, runID string) error {
	// Start a progress tracker
	done := make(chan error, 1)

	go func() {
		done <- executor.Execute(ctx, initialState)
	}()

	// Track progress
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			return err

		case <-ticker.C:
			status, err := executor.GetGraphStatus()
			if err == nil {
				o.visualization.ShowProgress(status)
			}
		}
	}
}

// countNodes counts total nodes in hierarchical plan
func (o *Orchestrator) countNodes(plan *HierarchicalPlan) int {
	count := 1
	for _, subplan := range plan.Subplans {
		count += o.countNodes(subplan)
	}
	return count
}

// Visualizer provides beautiful terminal output
type Visualizer struct{}

func NewVisualizer() *Visualizer {
	return &Visualizer{}
}

func (v *Visualizer) ShowBanner() {
	fmt.Println()
	fmt.Println(strings.Repeat("═", 80))
	fmt.Println("  🎭 GOAP ORCHESTRATOR - Where GOFAI Reasoning Meets LLM Generation")
	fmt.Println(strings.Repeat("═", 80))
	fmt.Println()
}

func (v *Visualizer) ShowPhilosophy() {
	fmt.Println("  📜 Core Philosophy:")
	fmt.Println("     • GOFAI (Good Old Fashioned AI) = REASONING MONARCH 👑")
	fmt.Println("       → Planning, search, logic, state management")
	fmt.Println()
	fmt.Println("     • LLM (Large Language Models) = CONTENT GENERATORS 🎨")
	fmt.Println("       → Decomposition hints, code generation, text creation")
	fmt.Println()
	fmt.Println("  The GOFAI core reasons about WHAT to do and HOW to do it.")
	fmt.Println("  The LLMs serve as powerful generators when asked.")
	fmt.Println()
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()
}

func (v *Visualizer) ShowPhase(name, description string) {
	fmt.Println()
	fmt.Println(fmt.Sprintf("┌─ %s ─", name) + strings.Repeat("─", 80-len(name)-4))
	fmt.Println(fmt.Sprintf("│  %s", description))
	fmt.Println("└" + strings.Repeat("─", 79))
	fmt.Println()
}

func (v *Visualizer) ShowPlanSummary(plan *HierarchicalPlan, duration time.Duration) {
	fmt.Println()
	fmt.Println("  📋 Plan Summary:")
	fmt.Println(fmt.Sprintf("     Goal: %s", plan.Goal.Name()))
	fmt.Println(fmt.Sprintf("     Planning Time: %v", duration))
	fmt.Println(fmt.Sprintf("     Max Depth: %d", plan.Depth))

	totalActions := len(plan.AllActions())
	fmt.Println(fmt.Sprintf("     Total Actions: %d", totalActions))

	if plan.IsAtomic() {
		fmt.Println("     Type: Atomic (direct actions)")
	} else {
		fmt.Println(fmt.Sprintf("     Type: Hierarchical (%d subgoals)", len(plan.Subplans)))
	}

	fmt.Println()
	fmt.Println("  📊 Plan Structure:")
	v.showPlanTree(plan, 0)
	fmt.Println()
}

func (v *Visualizer) showPlanTree(plan *HierarchicalPlan, indent int) {
	prefix := strings.Repeat("   ", indent)

	if plan.IsAtomic() {
		fmt.Println(fmt.Sprintf("%s🎯 %s [%d actions]", prefix, plan.Goal.Name(), len(plan.Actions)))
	} else {
		fmt.Println(fmt.Sprintf("%s🎯 %s", prefix, plan.Goal.Name()))
		for _, subplan := range plan.Subplans {
			v.showPlanTree(subplan, indent+1)
		}
	}
}

func (v *Visualizer) ShowProgress(status *GraphStatus) {
	total := status.TotalNodes
	completed := status.CompletedNodes
	failed := status.FailedNodes
	running := status.RunningNodes

	percent := 0.0
	if total > 0 {
		percent = float64(completed) / float64(total) * 100
	}

	bar := v.makeProgressBar(percent, 40)

	fmt.Printf("\r  Progress: %s %.1f%% | %d/%d nodes | Running: %d | Failed: %d",
		bar, percent, completed, total, running, failed)
}

func (v *Visualizer) makeProgressBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	empty := width - filled

	bar := "["
	bar += strings.Repeat("█", filled)
	bar += strings.Repeat("░", empty)
	bar += "]"

	return bar
}

func (v *Visualizer) ShowResults(status *GraphStatus) {
	fmt.Println()
	fmt.Println()
	fmt.Println(strings.Repeat("═", 80))
	fmt.Println("  ✨ EXECUTION COMPLETE")
	fmt.Println(strings.Repeat("═", 80))
	fmt.Println()
	fmt.Println("  📊 Final Statistics:")
	fmt.Println(fmt.Sprintf("     Total Nodes: %d", status.TotalNodes))
	fmt.Println(fmt.Sprintf("     ✓ Completed: %d", status.CompletedNodes))
	fmt.Println(fmt.Sprintf("     ⊘ Skipped: %d", status.SkippedNodes))
	fmt.Println(fmt.Sprintf("     ✗ Failed: %d", status.FailedNodes))

	if status.IsComplete() {
		if status.HasFailures() {
			fmt.Println()
			fmt.Println("  ⚠️  STATUS: COMPLETE WITH FAILURES")
		} else {
			fmt.Println()
			fmt.Println("  ✅ STATUS: SUCCESS - ALL GOALS ACHIEVED")
		}
	} else {
		fmt.Println()
		fmt.Println("  ⏸️  STATUS: INCOMPLETE")
	}

	fmt.Println()
	fmt.Println(strings.Repeat("═", 80))
	fmt.Println()
}

// PlanAnalyzer provides insights into plans
type PlanAnalyzer struct{}

// AnalyzePlan provides statistics and insights about a plan
func (pa *PlanAnalyzer) AnalyzePlan(plan *HierarchicalPlan) *PlanAnalysis {
	analysis := &PlanAnalysis{
		TotalNodes:    0,
		AtomicNodes:   0,
		MaxDepth:      0,
		TotalActions:  0,
		TotalCost:     0.0,
		GoalsByDepth:  make(map[int]int),
	}

	pa.analyzePlanRecursive(plan, analysis, 0)

	return analysis
}

func (pa *PlanAnalyzer) analyzePlanRecursive(plan *HierarchicalPlan, analysis *PlanAnalysis, depth int) {
	analysis.TotalNodes++
	analysis.GoalsByDepth[depth]++

	if depth > analysis.MaxDepth {
		analysis.MaxDepth = depth
	}

	if plan.IsAtomic() {
		analysis.AtomicNodes++
		analysis.TotalActions += len(plan.Actions)

		for _, action := range plan.Actions {
			analysis.TotalCost += action.Cost()
		}
	} else {
		for _, subplan := range plan.Subplans {
			pa.analyzePlanRecursive(subplan, analysis, depth+1)
		}
	}
}

// PlanAnalysis contains statistics about a plan
type PlanAnalysis struct {
	TotalNodes   int
	AtomicNodes  int
	MaxDepth     int
	TotalActions int
	TotalCost    float64
	GoalsByDepth map[int]int
}

func (pa *PlanAnalysis) String() string {
	return fmt.Sprintf("PlanAnalysis{nodes=%d, atomic=%d, depth=%d, actions=%d, cost=%.1f}",
		pa.TotalNodes, pa.AtomicNodes, pa.MaxDepth, pa.TotalActions, pa.TotalCost)
}
