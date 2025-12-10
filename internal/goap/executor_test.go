package goap

import (
	"context"
	"testing"
)

func TestGraphExecutor(t *testing.T) {
	tmpDir := t.TempDir()
	persistence := NewGraphPersistence(tmpDir)

	t.Run("RegisterActions", func(t *testing.T) {
		executor := NewGraphExecutor(persistence, "test-run")

		action1 := NewSimpleAction("Action1", "A1", WorldState{}, WorldState{"a": 1}, 1.0, nil)
		action2 := NewSimpleAction("Action2", "A2", WorldState{}, WorldState{"b": 2}, 1.0, nil)

		executor.RegisterAction(action1)
		executor.RegisterActions([]Action{action2})

		if len(executor.actions) != 2 {
			t.Errorf("Expected 2 registered actions, got %d", len(executor.actions))
		}

		if executor.actions["Action1"] == nil {
			t.Error("Action1 should be registered")
		}

		if executor.actions["Action2"] == nil {
			t.Error("Action2 should be registered")
		}
	})

	t.Run("ExecuteAtomicPlan", func(t *testing.T) {
		runID := "test-exec-atomic"

		executed := false
		action := NewSimpleAction(
			"TestAction",
			"Test",
			WorldState{},
			WorldState{"done": true},
			1.0,
			func(ctx context.Context, ws WorldState) error {
				executed = true
				return nil
			},
		)

		goal := NewGoal("TestGoal", "Test", WorldState{"done": true}, 1.0)
		plan := &HierarchicalPlan{
			Goal:     goal,
			Subplans: nil,
			Actions:  []Action{action},
			Depth:    0,
		}

		graph := BuildGraphFromPlan(plan, "test-agent")
		err := persistence.SaveGraph(graph, runID)
		if err != nil {
			t.Fatalf("Failed to save graph: %v", err)
		}

		executor := NewGraphExecutor(persistence, runID)
		executor.RegisterAction(action)

		initialState := NewWorldState()
		ctx := context.Background()

		err = executor.Execute(ctx, initialState)
		if err != nil {
			t.Fatalf("Execution failed: %v", err)
		}

		if !executed {
			t.Error("Action should have been executed")
		}

		// Verify status was updated
		status, err := executor.GetGraphStatus()
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}

		if status.CompletedNodes != 1 {
			t.Errorf("Expected 1 completed node, got %d", status.CompletedNodes)
		}
	})

	t.Run("ExecuteCompositePlan", func(t *testing.T) {
		runID := "test-exec-composite"

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

		atomicPlan1 := &HierarchicalPlan{
			Goal:     NewGoal("G1", "Goal1", WorldState{"a": 1}, 1.0),
			Subplans: nil,
			Actions:  []Action{action1},
			Depth:    1,
		}

		atomicPlan2 := &HierarchicalPlan{
			Goal:     NewGoal("G2", "Goal2", WorldState{"b": 2}, 1.0),
			Subplans: nil,
			Actions:  []Action{action2},
			Depth:    1,
		}

		compositePlan := &HierarchicalPlan{
			Goal:     NewGoal("Root", "Root goal", WorldState{"a": 1, "b": 2}, 10.0),
			Subplans: []*HierarchicalPlan{atomicPlan1, atomicPlan2},
			Actions:  nil,
			Depth:    0,
		}

		graph := BuildGraphFromPlan(compositePlan, "test-agent")
		err := persistence.SaveGraph(graph, runID)
		if err != nil {
			t.Fatalf("Failed to save graph: %v", err)
		}

		executor := NewGraphExecutor(persistence, runID)
		executor.RegisterActions([]Action{action1, action2})

		initialState := NewWorldState()
		ctx := context.Background()

		err = executor.Execute(ctx, initialState)
		if err != nil {
			t.Fatalf("Execution failed: %v", err)
		}

		if !exec1 || !exec2 {
			t.Error("Both actions should have been executed")
		}

		status, err := executor.GetGraphStatus()
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}

		if status.CompletedNodes != 3 {
			t.Errorf("Expected 3 completed nodes, got %d", status.CompletedNodes)
		}
	})

	t.Run("SkipSatisfiedGoal", func(t *testing.T) {
		runID := "test-skip-satisfied"

		action := NewSimpleAction(
			"Action",
			"Should not execute",
			WorldState{},
			WorldState{"done": true},
			1.0,
			func(ctx context.Context, ws WorldState) error {
				t.Error("Action should not execute when goal already satisfied")
				return nil
			},
		)

		goal := NewGoal("AlreadyDone", "Already satisfied", WorldState{"done": true}, 1.0)
		plan := &HierarchicalPlan{
			Goal:     goal,
			Subplans: nil,
			Actions:  []Action{action},
			Depth:    0,
		}

		graph := BuildGraphFromPlan(plan, "test-agent")
		err := persistence.SaveGraph(graph, runID)
		if err != nil {
			t.Fatalf("Failed to save graph: %v", err)
		}

		executor := NewGraphExecutor(persistence, runID)
		executor.RegisterAction(action)

		// Initial state already satisfies the goal
		initialState := NewWorldState()
		initialState.Set("done", true)

		ctx := context.Background()
		err = executor.Execute(ctx, initialState)
		if err != nil {
			t.Fatalf("Execution failed: %v", err)
		}

		status, err := executor.GetGraphStatus()
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}

		if status.SkippedNodes != 1 {
			t.Errorf("Expected 1 skipped node, got %d", status.SkippedNodes)
		}
	})

	t.Run("HandleActionFailure", func(t *testing.T) {
		runID := "test-action-failure"

		action := NewSimpleAction(
			"FailingAction",
			"Fails",
			WorldState{},
			WorldState{"done": true},
			1.0,
			func(ctx context.Context, ws WorldState) error {
				return nil // Will fail due to preconditions in Execute
			},
		)

		// Set preconditions that won't be met
		action.BaseAction.preconditions = WorldState{"prerequisite": true}

		goal := NewGoal("WillFail", "Will fail", WorldState{"done": true}, 1.0)
		plan := &HierarchicalPlan{
			Goal:     goal,
			Subplans: nil,
			Actions:  []Action{action},
			Depth:    0,
		}

		graph := BuildGraphFromPlan(plan, "test-agent")
		err := persistence.SaveGraph(graph, runID)
		if err != nil {
			t.Fatalf("Failed to save graph: %v", err)
		}

		executor := NewGraphExecutor(persistence, runID)
		executor.RegisterAction(action)

		initialState := NewWorldState()
		ctx := context.Background()

		err = executor.Execute(ctx, initialState)
		if err == nil {
			t.Error("Execution should fail when action fails")
		}

		status, err := executor.GetGraphStatus()
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}

		if status.FailedNodes != 1 {
			t.Errorf("Expected 1 failed node, got %d", status.FailedNodes)
		}

		if !status.HasFailures() {
			t.Error("Status should indicate failures")
		}
	})
}
