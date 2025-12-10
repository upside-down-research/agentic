package goap

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
)

// GoalRefiner is responsible for taking high-level goals and decomposing them
// into more refined subgoals. This enables hierarchical planning where complex
// goals are progressively broken down into simpler, more concrete goals.
type GoalRefiner interface {
	// Refine takes a goal and the current world state and returns a set of
	// subgoals that, when achieved, will accomplish the parent goal.
	// Returns nil if the goal cannot be refined further (it's atomic).
	Refine(ctx context.Context, goal *Goal, current WorldState) ([]*Goal, error)

	// IsAtomic determines if a goal is atomic (cannot be refined further).
	IsAtomic(goal *Goal, current WorldState) bool
}

// HierarchicalPlanner combines goal refinement with action planning to create
// a hierarchical planning system. It recursively decomposes goals into subgoals
// until reaching atomic goals that can be achieved by actions.
type HierarchicalPlanner struct {
	planner *Planner
	refiner GoalRefiner
	maxDepth int
}

// NewHierarchicalPlanner creates a new hierarchical planner.
func NewHierarchicalPlanner(planner *Planner, refiner GoalRefiner, maxDepth int) *HierarchicalPlanner {
	return &HierarchicalPlanner{
		planner:  planner,
		refiner:  refiner,
		maxDepth: maxDepth,
	}
}

// PlanHierarchical creates a hierarchical plan to achieve a goal.
// It recursively refines goals into subgoals until reaching atomic goals,
// then uses the action planner to find action sequences for each atomic goal.
func (hp *HierarchicalPlanner) PlanHierarchical(ctx context.Context, current WorldState, goal *Goal) (*HierarchicalPlan, error) {
	log.Info("Starting hierarchical planning", "goal", goal.Name())
	return hp.planRecursive(ctx, current, goal, 0)
}

func (hp *HierarchicalPlanner) planRecursive(ctx context.Context, current WorldState, goal *Goal, depth int) (*HierarchicalPlan, error) {
	if depth > hp.maxDepth {
		return nil, fmt.Errorf("maximum planning depth exceeded: %d", hp.maxDepth)
	}

	log.Info("Planning at depth", "depth", depth, "goal", goal.Name())

	// Check if goal is already satisfied
	if goal.IsSatisfied(current) {
		log.Info("Goal already satisfied", "goal", goal.Name())
		return &HierarchicalPlan{
			Goal:     goal,
			Subplans: nil,
			Actions:  nil,
			Depth:    depth,
		}, nil
	}

	// Check if this is an atomic goal
	if hp.refiner.IsAtomic(goal, current) {
		log.Info("Goal is atomic, finding action plan", "goal", goal.Name())

		// Use the action planner to find a sequence of actions
		actionPlan := hp.planner.FindPlan(current, goal)
		if actionPlan == nil {
			return nil, fmt.Errorf("no action plan found for atomic goal: %s", goal.Name())
		}

		return &HierarchicalPlan{
			Goal:     goal,
			Subplans: nil,
			Actions:  actionPlan.Actions,
			Depth:    depth,
		}, nil
	}

	// Goal is not atomic, refine it into subgoals
	log.Info("Refining goal into subgoals", "goal", goal.Name())
	subgoals, err := hp.refiner.Refine(ctx, goal, current)
	if err != nil {
		return nil, fmt.Errorf("failed to refine goal %s: %w", goal.Name(), err)
	}

	if len(subgoals) == 0 {
		return nil, fmt.Errorf("goal refinement produced no subgoals: %s", goal.Name())
	}

	log.Info("Goal refined", "goal", goal.Name(), "numSubgoals", len(subgoals))

	// Recursively plan for each subgoal
	subplans := make([]*HierarchicalPlan, 0, len(subgoals))
	workingState := current.Clone()

	for i, subgoal := range subgoals {
		log.Info("Planning subgoal", "index", i, "subgoal", subgoal.Name())

		subplan, err := hp.planRecursive(ctx, workingState, subgoal, depth+1)
		if err != nil {
			return nil, fmt.Errorf("failed to plan subgoal %s: %w", subgoal.Name(), err)
		}

		subplans = append(subplans, subplan)

		// Update working state with the effects of this subplan
		// This ensures subsequent subgoals can depend on earlier ones
		if subplan.Actions != nil {
			for _, action := range subplan.Actions {
				workingState.Apply(action.Effects())
			}
		}
	}

	return &HierarchicalPlan{
		Goal:     goal,
		Subplans: subplans,
		Actions:  nil, // No direct actions for non-atomic goals
		Depth:    depth,
	}, nil
}

// HierarchicalPlan represents a hierarchical plan that may contain subplans.
// Leaf nodes (atomic goals) have Actions but no Subplans.
// Internal nodes (composite goals) have Subplans but no Actions.
type HierarchicalPlan struct {
	Goal     *Goal
	Subplans []*HierarchicalPlan
	Actions  []Action
	Depth    int
}

// IsAtomic returns true if this plan node is atomic (has actions, no subplans).
func (hp *HierarchicalPlan) IsAtomic() bool {
	return len(hp.Subplans) == 0
}

// AllActions returns all actions in this plan and its subplans, in execution order.
func (hp *HierarchicalPlan) AllActions() []Action {
	if hp.IsAtomic() {
		return hp.Actions
	}

	var allActions []Action
	for _, subplan := range hp.Subplans {
		allActions = append(allActions, subplan.AllActions()...)
	}
	return allActions
}

// Execute executes the hierarchical plan, running all actions in order.
func (hp *HierarchicalPlan) Execute(ctx context.Context, current WorldState) error {
	if hp.IsAtomic() {
		log.Info("Executing atomic plan", "goal", hp.Goal.Name(), "numActions", len(hp.Actions))
		for i, action := range hp.Actions {
			log.Info("Executing action", "index", i, "action", action.Name())
			if err := action.Execute(ctx, current); err != nil {
				return fmt.Errorf("action %s failed: %w", action.Name(), err)
			}
		}
		return nil
	}

	log.Info("Executing composite plan", "goal", hp.Goal.Name(), "numSubplans", len(hp.Subplans))
	for i, subplan := range hp.Subplans {
		log.Info("Executing subplan", "index", i, "subgoal", subplan.Goal.Name())
		if err := subplan.Execute(ctx, current); err != nil {
			return fmt.Errorf("subplan %s failed: %w", subplan.Goal.Name(), err)
		}
	}
	return nil
}

// String returns a string representation of the hierarchical plan.
func (hp *HierarchicalPlan) String() string {
	return hp.stringWithIndent(0)
}

func (hp *HierarchicalPlan) stringWithIndent(indent int) string {
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}

	result := fmt.Sprintf("%sGoal: %s\n", prefix, hp.Goal.Name())

	if hp.IsAtomic() {
		result += fmt.Sprintf("%s  Actions (%d):\n", prefix, len(hp.Actions))
		for i, action := range hp.Actions {
			result += fmt.Sprintf("%s    %d. %s\n", prefix, i+1, action.Name())
		}
	} else {
		result += fmt.Sprintf("%s  Subgoals (%d):\n", prefix, len(hp.Subplans))
		for i, subplan := range hp.Subplans {
			result += fmt.Sprintf("%s  %d.\n", prefix, i+1)
			result += subplan.stringWithIndent(indent + 2)
		}
	}

	return result
}
