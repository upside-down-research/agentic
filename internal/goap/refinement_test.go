package goap

import (
	"context"
	"testing"
)

// MockGoalRefiner is a simple mock for testing hierarchical planning
type MockGoalRefiner struct {
	refinements map[string][]*Goal
}

func NewMockGoalRefiner() *MockGoalRefiner {
	return &MockGoalRefiner{
		refinements: make(map[string][]*Goal),
	}
}

func (m *MockGoalRefiner) AddRefinement(goalName string, subgoals []*Goal) {
	m.refinements[goalName] = subgoals
}

func (m *MockGoalRefiner) Refine(ctx context.Context, goal *Goal, current WorldState) ([]*Goal, error) {
	subgoals, exists := m.refinements[goal.Name()]
	if !exists {
		return nil, nil
	}
	return subgoals, nil
}

func (m *MockGoalRefiner) IsAtomic(goal *Goal, current WorldState) bool {
	_, exists := m.refinements[goal.Name()]
	return !exists
}

func TestHierarchicalPlanner(t *testing.T) {
	t.Run("PlanAtomicGoal", func(t *testing.T) {
		// Create simple action and planner
		action := NewSimpleAction(
			"DoTask",
			"Complete task",
			WorldState{},
			WorldState{"task_done": true},
			1.0,
			func(ctx context.Context, ws WorldState) error { return nil },
		)

		planner := NewPlanner([]Action{action})
		refiner := NewMockGoalRefiner()

		hp := NewHierarchicalPlanner(planner, refiner, 5)

		// Create atomic goal
		goal := NewGoal("CompleteTask", "Complete the task", WorldState{"task_done": true}, 1.0)

		current := NewWorldState()
		ctx := context.Background()

		plan, err := hp.PlanHierarchical(ctx, current, goal)
		if err != nil {
			t.Fatalf("Planning failed: %v", err)
		}

		if plan == nil {
			t.Fatal("Plan should not be nil")
		}

		if !plan.IsAtomic() {
			t.Error("Plan should be atomic")
		}

		if len(plan.Actions) != 1 {
			t.Errorf("Expected 1 action, got %d", len(plan.Actions))
		}

		if plan.Actions[0].Name() != "DoTask" {
			t.Errorf("Expected action 'DoTask', got %s", plan.Actions[0].Name())
		}
	})

	t.Run("PlanCompositeGoal", func(t *testing.T) {
		// Create actions
		action1 := NewSimpleAction(
			"SubTask1",
			"Do subtask 1",
			WorldState{},
			WorldState{"sub1_done": true},
			1.0,
			func(ctx context.Context, ws WorldState) error { return nil },
		)

		action2 := NewSimpleAction(
			"SubTask2",
			"Do subtask 2",
			WorldState{},
			WorldState{"sub2_done": true},
			1.0,
			func(ctx context.Context, ws WorldState) error { return nil },
		)

		planner := NewPlanner([]Action{action1, action2})

		// Set up refiner with decomposition
		refiner := NewMockGoalRefiner()

		subgoal1 := NewGoal("Subgoal1", "First subgoal", WorldState{"sub1_done": true}, 2.0)
		subgoal2 := NewGoal("Subgoal2", "Second subgoal", WorldState{"sub2_done": true}, 1.0)

		refiner.AddRefinement("MainGoal", []*Goal{subgoal1, subgoal2})

		hp := NewHierarchicalPlanner(planner, refiner, 5)

		mainGoal := NewGoal("MainGoal", "Main goal", WorldState{"sub1_done": true, "sub2_done": true}, 10.0)

		current := NewWorldState()
		ctx := context.Background()

		plan, err := hp.PlanHierarchical(ctx, current, mainGoal)
		if err != nil {
			t.Fatalf("Planning failed: %v", err)
		}

		if plan == nil {
			t.Fatal("Plan should not be nil")
		}

		if plan.IsAtomic() {
			t.Error("Plan should be composite")
		}

		if len(plan.Subplans) != 2 {
			t.Errorf("Expected 2 subplans, got %d", len(plan.Subplans))
		}

		// Check subplans are atomic
		for i, subplan := range plan.Subplans {
			if !subplan.IsAtomic() {
				t.Errorf("Subplan %d should be atomic", i)
			}
		}

		// Check all actions
		allActions := plan.AllActions()
		if len(allActions) != 2 {
			t.Errorf("Expected 2 total actions, got %d", len(allActions))
		}
	})

	t.Run("DeepHierarchy", func(t *testing.T) {
		// Create leaf action
		leafAction := NewSimpleAction(
			"LeafAction",
			"Leaf level action",
			WorldState{},
			WorldState{"leaf": true},
			1.0,
			func(ctx context.Context, ws WorldState) error { return nil },
		)

		planner := NewPlanner([]Action{leafAction})
		refiner := NewMockGoalRefiner()

		// Level 2: atomic goal
		level2Goal := NewGoal("Level2", "Level 2 goal", WorldState{"leaf": true}, 1.0)

		// Level 1: decomposes to level 2
		level1Goal := NewGoal("Level1", "Level 1 goal", WorldState{"leaf": true}, 2.0)
		refiner.AddRefinement("Level1", []*Goal{level2Goal})

		// Level 0: decomposes to level 1
		rootGoal := NewGoal("Root", "Root goal", WorldState{"leaf": true}, 3.0)
		refiner.AddRefinement("Root", []*Goal{level1Goal})

		hp := NewHierarchicalPlanner(planner, refiner, 5)

		current := NewWorldState()
		ctx := context.Background()

		plan, err := hp.PlanHierarchical(ctx, current, rootGoal)
		if err != nil {
			t.Fatalf("Planning failed: %v", err)
		}

		if plan.Depth != 0 {
			t.Errorf("Expected root depth 0, got %d", plan.Depth)
		}

		// Should have 1 subplan at depth 1
		if len(plan.Subplans) != 1 {
			t.Errorf("Expected 1 subplan, got %d", len(plan.Subplans))
		}

		level1Plan := plan.Subplans[0]
		if level1Plan.Depth != 1 {
			t.Errorf("Expected depth 1, got %d", level1Plan.Depth)
		}

		// Level 1 should have 1 subplan at depth 2
		if len(level1Plan.Subplans) != 1 {
			t.Errorf("Expected 1 subplan at level 1, got %d", len(level1Plan.Subplans))
		}

		level2Plan := level1Plan.Subplans[0]
		if level2Plan.Depth != 2 {
			t.Errorf("Expected depth 2, got %d", level2Plan.Depth)
		}

		if !level2Plan.IsAtomic() {
			t.Error("Leaf plan should be atomic")
		}
	})

	t.Run("MaxDepthExceeded", func(t *testing.T) {
		action := NewSimpleAction(
			"Action",
			"Action",
			WorldState{},
			WorldState{"done": true},
			1.0,
			func(ctx context.Context, ws WorldState) error { return nil },
		)

		planner := NewPlanner([]Action{action})
		refiner := NewMockGoalRefiner()

		// Create infinite refinement loop
		goal1 := NewGoal("Goal1", "G1", WorldState{"done": true}, 1.0)
		goal2 := NewGoal("Goal2", "G2", WorldState{"done": true}, 1.0)

		refiner.AddRefinement("Goal1", []*Goal{goal2})
		refiner.AddRefinement("Goal2", []*Goal{goal1})

		hp := NewHierarchicalPlanner(planner, refiner, 3) // Low max depth

		current := NewWorldState()
		ctx := context.Background()

		_, err := hp.PlanHierarchical(ctx, current, goal1)
		if err == nil {
			t.Error("Should fail when max depth exceeded")
		}
	})

	t.Run("GoalAlreadySatisfied", func(t *testing.T) {
		planner := NewPlanner([]Action{})
		refiner := NewMockGoalRefiner()
		hp := NewHierarchicalPlanner(planner, refiner, 5)

		goal := NewGoal("AlreadyDone", "Already satisfied", WorldState{"done": true}, 1.0)

		current := NewWorldState()
		current.Set("done", true)

		ctx := context.Background()

		plan, err := hp.PlanHierarchical(ctx, current, goal)
		if err != nil {
			t.Fatalf("Planning failed: %v", err)
		}

		if plan == nil {
			t.Fatal("Plan should not be nil")
		}

		if len(plan.Subplans) != 0 || len(plan.Actions) != 0 {
			t.Error("Plan should be empty for already satisfied goal")
		}
	})
}

func TestHierarchicalPlanExecution(t *testing.T) {
	t.Run("ExecuteAtomicPlan", func(t *testing.T) {
		executed := false
		action := NewSimpleAction(
			"Action",
			"Do it",
			WorldState{},
			WorldState{"done": true},
			1.0,
			func(ctx context.Context, ws WorldState) error {
				executed = true
				return nil
			},
		)

		goal := NewGoal("Goal", "A goal", WorldState{"done": true}, 1.0)
		plan := &HierarchicalPlan{
			Goal:     goal,
			Subplans: nil,
			Actions:  []Action{action},
			Depth:    0,
		}

		current := NewWorldState()
		ctx := context.Background()

		err := plan.Execute(ctx, current)
		if err != nil {
			t.Fatalf("Execution failed: %v", err)
		}

		if !executed {
			t.Error("Action should have been executed")
		}
	})

	t.Run("ExecuteCompositePlan", func(t *testing.T) {
		exec1 := false
		exec2 := false

		action1 := NewSimpleAction(
			"Action1",
			"A1",
			WorldState{},
			WorldState{"a": 1},
			1.0,
			func(ctx context.Context, ws WorldState) error {
				exec1 = true
				return nil
			},
		)

		action2 := NewSimpleAction(
			"Action2",
			"A2",
			WorldState{},
			WorldState{"b": 2},
			1.0,
			func(ctx context.Context, ws WorldState) error {
				exec2 = true
				return nil
			},
		)

		subplan1 := &HierarchicalPlan{
			Goal:     NewGoal("G1", "G1", WorldState{"a": 1}, 1.0),
			Subplans: nil,
			Actions:  []Action{action1},
			Depth:    1,
		}

		subplan2 := &HierarchicalPlan{
			Goal:     NewGoal("G2", "G2", WorldState{"b": 2}, 1.0),
			Subplans: nil,
			Actions:  []Action{action2},
			Depth:    1,
		}

		compositePlan := &HierarchicalPlan{
			Goal:     NewGoal("Root", "Root", WorldState{"a": 1, "b": 2}, 10.0),
			Subplans: []*HierarchicalPlan{subplan1, subplan2},
			Actions:  nil,
			Depth:    0,
		}

		current := NewWorldState()
		ctx := context.Background()

		err := compositePlan.Execute(ctx, current)
		if err != nil {
			t.Fatalf("Execution failed: %v", err)
		}

		if !exec1 || !exec2 {
			t.Error("Both actions should have been executed")
		}
	})
}

func TestPlanString(t *testing.T) {
	t.Run("EmptyPlan", func(t *testing.T) {
		plan := &Plan{
			Actions: []Action{},
			Cost:    0,
		}

		str := plan.String()
		if str != "Empty Plan" {
			t.Errorf("Expected 'Empty Plan', got %s", str)
		}
	})

	t.Run("PlanWithActions", func(t *testing.T) {
		action1 := NewSimpleAction("Action1", "A1", WorldState{}, WorldState{}, 1.0, nil)
		action2 := NewSimpleAction("Action2", "A2", WorldState{}, WorldState{}, 2.0, nil)

		plan := &Plan{
			Actions: []Action{action1, action2},
			Cost:    3.0,
		}

		str := plan.String()
		if str == "Empty Plan" {
			t.Error("Should not be empty plan")
		}

		// Should contain action names
		if !contains(str, "Action1") || !contains(str, "Action2") {
			t.Error("Plan string should contain action names")
		}
	})
}

func TestHierarchicalPlanString(t *testing.T) {
	action := NewSimpleAction("TestAction", "Test", WorldState{}, WorldState{}, 1.0, nil)

	atomicPlan := &HierarchicalPlan{
		Goal:     NewGoal("AtomicGoal", "Atomic", WorldState{}, 1.0),
		Subplans: nil,
		Actions:  []Action{action},
		Depth:    0,
	}

	str := atomicPlan.String()
	if !contains(str, "AtomicGoal") {
		t.Error("String should contain goal name")
	}

	if !contains(str, "TestAction") {
		t.Error("String should contain action name")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
