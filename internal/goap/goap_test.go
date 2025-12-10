package goap

import (
	"context"
	"testing"
)

func TestWorldState(t *testing.T) {
	t.Run("Create and Set", func(t *testing.T) {
		ws := NewWorldState()
		ws.Set("key1", "value1")
		ws.Set("key2", 42)

		if ws.Get("key1") != "value1" {
			t.Errorf("Expected 'value1', got %v", ws.Get("key1"))
		}
		if ws.Get("key2") != 42 {
			t.Errorf("Expected 42, got %v", ws.Get("key2"))
		}
	})

	t.Run("Clone", func(t *testing.T) {
		ws := NewWorldState()
		ws.Set("key1", "value1")

		clone := ws.Clone()
		clone.Set("key2", "value2")

		if ws.Has("key2") {
			t.Error("Original should not have key2 after clone modification")
		}
		if !clone.Has("key1") {
			t.Error("Clone should have key1 from original")
		}
	})

	t.Run("Matches", func(t *testing.T) {
		ws := NewWorldState()
		ws.Set("a", 1)
		ws.Set("b", 2)
		ws.Set("c", 3)

		conditions := NewWorldState()
		conditions.Set("a", 1)
		conditions.Set("b", 2)

		if !ws.Matches(conditions) {
			t.Error("WorldState should match conditions")
		}

		conditions.Set("d", 4)
		if ws.Matches(conditions) {
			t.Error("WorldState should not match conditions with missing key")
		}
	})

	t.Run("Distance", func(t *testing.T) {
		current := NewWorldState()
		current.Set("a", 1)
		current.Set("b", 2)

		goal := NewWorldState()
		goal.Set("a", 1)
		goal.Set("b", 3)
		goal.Set("c", 4)

		distance := current.Distance(goal)
		if distance != 2 {
			t.Errorf("Expected distance 2, got %d", distance)
		}
	})
}

func TestGoal(t *testing.T) {
	t.Run("Create Goal", func(t *testing.T) {
		desiredState := NewWorldState()
		desiredState.Set("task_complete", true)

		goal := NewGoal("TestGoal", "A test goal", desiredState, 10.0)

		if goal.Name() != "TestGoal" {
			t.Errorf("Expected name 'TestGoal', got %s", goal.Name())
		}
		if goal.Priority() != 10.0 {
			t.Errorf("Expected priority 10.0, got %f", goal.Priority())
		}
	})

	t.Run("Goal Satisfaction", func(t *testing.T) {
		desiredState := NewWorldState()
		desiredState.Set("task_complete", true)
		desiredState.Set("verified", true)

		goal := NewGoal("TestGoal", "Test", desiredState, 1.0)

		current := NewWorldState()
		current.Set("task_complete", false)

		if goal.IsSatisfied(current) {
			t.Error("Goal should not be satisfied")
		}

		current.Set("task_complete", true)
		current.Set("verified", true)

		if !goal.IsSatisfied(current) {
			t.Error("Goal should be satisfied")
		}
	})
}

func TestSimpleAction(t *testing.T) {
	t.Run("Action Execution", func(t *testing.T) {
		preconditions := NewWorldState()
		preconditions.Set("ready", true)

		effects := NewWorldState()
		effects.Set("task_done", true)

		executed := false
		action := NewSimpleAction(
			"TestAction",
			"A test action",
			preconditions,
			effects,
			1.0,
			func(ctx context.Context, ws WorldState) error {
				executed = true
				return nil
			},
		)

		current := NewWorldState()
		current.Set("ready", true)

		ctx := context.Background()
		err := action.Execute(ctx, current)
		if err != nil {
			t.Errorf("Action execution failed: %v", err)
		}

		if !executed {
			t.Error("Action function was not executed")
		}

		if !current.Get("task_done").(bool) {
			t.Error("Action effects were not applied")
		}
	})

	t.Run("Action Preconditions", func(t *testing.T) {
		preconditions := NewWorldState()
		preconditions.Set("ready", true)

		effects := NewWorldState()

		action := NewSimpleAction(
			"TestAction",
			"Test",
			preconditions,
			effects,
			1.0,
			func(ctx context.Context, ws WorldState) error {
				return nil
			},
		)

		current := NewWorldState()
		current.Set("ready", false)

		if action.CanExecute(current) {
			t.Error("Action should not be executable without preconditions")
		}

		current.Set("ready", true)
		if !action.CanExecute(current) {
			t.Error("Action should be executable with preconditions met")
		}
	})
}

func TestPlanner(t *testing.T) {
	t.Run("Simple Plan", func(t *testing.T) {
		// Create actions
		action1 := NewSimpleAction(
			"Action1",
			"First action",
			NewWorldState(), // No preconditions
			WorldState{"step1": true},
			1.0,
			func(ctx context.Context, ws WorldState) error { return nil },
		)

		action2 := NewSimpleAction(
			"Action2",
			"Second action",
			WorldState{"step1": true}, // Requires step1
			WorldState{"step2": true},
			1.0,
			func(ctx context.Context, ws WorldState) error { return nil },
		)

		// Create planner
		planner := NewPlanner([]Action{action1, action2})

		// Create goal
		goal := NewGoal(
			"CompleteTask",
			"Complete both steps",
			WorldState{"step1": true, "step2": true},
			10.0,
		)

		// Find plan
		current := NewWorldState()
		plan := planner.FindPlan(current, goal)

		if plan == nil {
			t.Fatal("Planner should find a plan")
		}

		if len(plan.Actions) != 2 {
			t.Errorf("Expected 2 actions, got %d", len(plan.Actions))
		}

		if plan.Actions[0].Name() != "Action1" {
			t.Errorf("Expected Action1 first, got %s", plan.Actions[0].Name())
		}
		if plan.Actions[1].Name() != "Action2" {
			t.Errorf("Expected Action2 second, got %s", plan.Actions[1].Name())
		}
	})

	t.Run("Goal Already Satisfied", func(t *testing.T) {
		planner := NewPlanner([]Action{})

		goal := NewGoal(
			"AlreadyDone",
			"Already satisfied",
			WorldState{"done": true},
			1.0,
		)

		current := NewWorldState()
		current.Set("done", true)

		plan := planner.FindPlan(current, goal)

		if plan == nil {
			t.Fatal("Should return empty plan for satisfied goal")
		}

		if len(plan.Actions) != 0 {
			t.Errorf("Expected 0 actions for satisfied goal, got %d", len(plan.Actions))
		}
	})

	t.Run("No Plan Exists", func(t *testing.T) {
		action := NewSimpleAction(
			"WrongAction",
			"Does something else",
			NewWorldState(),
			WorldState{"wrong": true},
			1.0,
			func(ctx context.Context, ws WorldState) error { return nil },
		)

		planner := NewPlanner([]Action{action})

		goal := NewGoal(
			"ImpossibleGoal",
			"Cannot be achieved",
			WorldState{"correct": true},
			1.0,
		)

		current := NewWorldState()
		plan := planner.FindPlan(current, goal)

		if plan != nil {
			t.Error("Should return nil when no plan exists")
		}
	})
}

func TestCompositeAction(t *testing.T) {
	t.Run("Composite Action Execution", func(t *testing.T) {
		step1Done := false
		step2Done := false

		sub1 := NewSimpleAction(
			"Subaction1",
			"First sub",
			NewWorldState(),
			WorldState{"sub1": true},
			1.0,
			func(ctx context.Context, ws WorldState) error {
				step1Done = true
				return nil
			},
		)

		sub2 := NewSimpleAction(
			"Subaction2",
			"Second sub",
			WorldState{"sub1": true},
			WorldState{"sub2": true},
			1.0,
			func(ctx context.Context, ws WorldState) error {
				step2Done = true
				return nil
			},
		)

		composite := NewCompositeAction(
			"CompositeAction",
			"Runs multiple subactions",
			NewWorldState(),
			WorldState{"composite_done": true},
			2.0,
			[]Action{sub1, sub2},
		)

		current := NewWorldState()
		ctx := context.Background()

		err := composite.Execute(ctx, current)
		if err != nil {
			t.Errorf("Composite execution failed: %v", err)
		}

		if !step1Done || !step2Done {
			t.Error("Subactions were not executed")
		}

		if !current.Get("composite_done").(bool) {
			t.Error("Composite effects were not applied")
		}
	})
}

func TestGoalSet(t *testing.T) {
	t.Run("Highest Priority", func(t *testing.T) {
		gs := NewGoalSet()

		goal1 := NewGoal("Goal1", "Low priority", NewWorldState(), 5.0)
		goal2 := NewGoal("Goal2", "High priority", NewWorldState(), 15.0)
		goal3 := NewGoal("Goal3", "Medium priority", NewWorldState(), 10.0)

		gs.Add(goal1)
		gs.Add(goal2)
		gs.Add(goal3)

		highest := gs.HighestPriority()
		if highest.Name() != "Goal2" {
			t.Errorf("Expected Goal2, got %s", highest.Name())
		}
	})

	t.Run("Most Achievable", func(t *testing.T) {
		current := NewWorldState()
		current.Set("a", 1)

		gs := NewGoalSet()

		goal1 := NewGoal("Goal1", "Far", WorldState{"a": 1, "b": 2, "c": 3}, 1.0)
		goal2 := NewGoal("Goal2", "Close", WorldState{"a": 1, "b": 2}, 1.0)
		goal3 := NewGoal("Goal3", "Very far", WorldState{"x": 1, "y": 2, "z": 3}, 1.0)

		gs.Add(goal1)
		gs.Add(goal2)
		gs.Add(goal3)

		mostAchievable := gs.MostAchievable(current)
		if mostAchievable.Name() != "Goal2" {
			t.Errorf("Expected Goal2, got %s", mostAchievable.Name())
		}
	})
}
