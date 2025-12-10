package goap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("BuildGraphFromPlan", func(t *testing.T) {
		// Create a simple hierarchical plan
		atomicGoal1 := NewGoal("AtomicGoal1", "First atomic goal", WorldState{"a": 1}, 1.0)
		atomicGoal2 := NewGoal("AtomicGoal2", "Second atomic goal", WorldState{"b": 2}, 1.0)

		action1 := NewSimpleAction("Action1", "Do A", WorldState{}, WorldState{"a": 1}, 1.0, nil)
		action2 := NewSimpleAction("Action2", "Do B", WorldState{}, WorldState{"b": 2}, 1.0, nil)

		atomicPlan1 := &HierarchicalPlan{
			Goal:     atomicGoal1,
			Subplans: nil,
			Actions:  []Action{action1},
			Depth:    1,
		}

		atomicPlan2 := &HierarchicalPlan{
			Goal:     atomicGoal2,
			Subplans: nil,
			Actions:  []Action{action2},
			Depth:    1,
		}

		rootGoal := NewGoal("RootGoal", "Complete both", WorldState{"a": 1, "b": 2}, 10.0)
		rootPlan := &HierarchicalPlan{
			Goal:     rootGoal,
			Subplans: []*HierarchicalPlan{atomicPlan1, atomicPlan2},
			Actions:  nil,
			Depth:    0,
		}

		graph := BuildGraphFromPlan(rootPlan, "test-agent")

		if graph == nil {
			t.Fatal("Graph should not be nil")
		}

		if graph.RootNodeID == "" {
			t.Error("Root node ID should not be empty")
		}

		if len(graph.Nodes) != 3 {
			t.Errorf("Expected 3 nodes (1 root + 2 children), got %d", len(graph.Nodes))
		}

		if graph.Metadata.TotalNodes != 3 {
			t.Errorf("Expected metadata TotalNodes=3, got %d", graph.Metadata.TotalNodes)
		}

		rootNode := graph.Nodes[graph.RootNodeID]
		if rootNode == nil {
			t.Fatal("Root node not found")
		}

		if rootNode.GoalName != "RootGoal" {
			t.Errorf("Expected root goal name 'RootGoal', got %s", rootNode.GoalName)
		}

		if len(rootNode.ChildIDs) != 2 {
			t.Errorf("Expected 2 children, got %d", len(rootNode.ChildIDs))
		}

		if rootNode.IsAtomic {
			t.Error("Root node should not be atomic")
		}

		// Check children are atomic
		for _, childID := range rootNode.ChildIDs {
			child := graph.Nodes[childID]
			if !child.IsAtomic {
				t.Errorf("Child node %s should be atomic", childID)
			}
			if len(child.ActionNames) != 1 {
				t.Errorf("Expected 1 action in child, got %d", len(child.ActionNames))
			}
		}
	})

	t.Run("SaveAndLoadGraph", func(t *testing.T) {
		goal := NewGoal("TestGoal", "A test", WorldState{"test": true}, 1.0)
		action := NewSimpleAction("TestAction", "Test", WorldState{}, WorldState{"test": true}, 1.0, nil)

		plan := &HierarchicalPlan{
			Goal:     goal,
			Subplans: nil,
			Actions:  []Action{action},
			Depth:    0,
		}

		graph := BuildGraphFromPlan(plan, "test-agent")

		persistence := NewGraphPersistence(tmpDir)
		runID := "test-run-1"

		// Save
		err := persistence.SaveGraph(graph, runID)
		if err != nil {
			t.Fatalf("Failed to save graph: %v", err)
		}

		// Verify files exist
		graphPath := filepath.Join(tmpDir, runID, "graph", "plan_graph.json")
		if _, err := os.Stat(graphPath); os.IsNotExist(err) {
			t.Error("Graph file should exist")
		}

		// Load
		loadedGraph, err := persistence.LoadGraph(runID)
		if err != nil {
			t.Fatalf("Failed to load graph: %v", err)
		}

		if loadedGraph.RootNodeID != graph.RootNodeID {
			t.Error("Loaded graph should have same root node ID")
		}

		if len(loadedGraph.Nodes) != len(graph.Nodes) {
			t.Errorf("Expected %d nodes, got %d", len(graph.Nodes), len(loadedGraph.Nodes))
		}
	})

	t.Run("LoadNodeContext", func(t *testing.T) {
		// Create a plan with parent-child relationships
		childGoal := NewGoal("Child", "Child goal", WorldState{"child": true}, 1.0)
		childAction := NewSimpleAction("ChildAction", "Do child", WorldState{}, WorldState{"child": true}, 1.0, nil)

		childPlan := &HierarchicalPlan{
			Goal:     childGoal,
			Subplans: nil,
			Actions:  []Action{childAction},
			Depth:    1,
		}

		parentGoal := NewGoal("Parent", "Parent goal", WorldState{"child": true}, 10.0)
		parentPlan := &HierarchicalPlan{
			Goal:     parentGoal,
			Subplans: []*HierarchicalPlan{childPlan},
			Actions:  nil,
			Depth:    0,
		}

		graph := BuildGraphFromPlan(parentPlan, "test-agent")
		persistence := NewGraphPersistence(tmpDir)
		runID := "test-run-2"

		err := persistence.SaveGraph(graph, runID)
		if err != nil {
			t.Fatalf("Failed to save graph: %v", err)
		}

		// Load context for child node
		childNodeID := graph.Nodes[graph.RootNodeID].ChildIDs[0]
		context, err := persistence.LoadNodeContext(runID, childNodeID)
		if err != nil {
			t.Fatalf("Failed to load node context: %v", err)
		}

		if context.Node == nil {
			t.Fatal("Context node should not be nil")
		}

		if context.Node.ID != childNodeID {
			t.Error("Context should be for requested node")
		}

		if context.Parent == nil {
			t.Error("Context should include parent")
		}

		if len(context.PathFromRoot) != 2 {
			t.Errorf("Expected path length 2, got %d", len(context.PathFromRoot))
		}
	})

	t.Run("UpdateNodeStatus", func(t *testing.T) {
		goal := NewGoal("StatusTest", "Test status", WorldState{"done": true}, 1.0)
		action := NewSimpleAction("Action", "Do it", WorldState{}, WorldState{"done": true}, 1.0, nil)

		plan := &HierarchicalPlan{
			Goal:     goal,
			Subplans: nil,
			Actions:  []Action{action},
			Depth:    0,
		}

		graph := BuildGraphFromPlan(plan, "test-agent")
		persistence := NewGraphPersistence(tmpDir)
		runID := "test-run-3"

		err := persistence.SaveGraph(graph, runID)
		if err != nil {
			t.Fatalf("Failed to save graph: %v", err)
		}

		nodeID := graph.RootNodeID

		// Update status
		result := &NodeResult{
			Success:      true,
			StateChanges: map[string]interface{}{"done": true},
		}

		err = persistence.UpdateNodeStatus(runID, nodeID, StatusCompleted, result)
		if err != nil {
			t.Fatalf("Failed to update status: %v", err)
		}

		// Load and verify
		updatedGraph, err := persistence.LoadGraph(runID)
		if err != nil {
			t.Fatalf("Failed to load updated graph: %v", err)
		}

		updatedNode := updatedGraph.Nodes[nodeID]
		if updatedNode.Status != StatusCompleted {
			t.Errorf("Expected status Completed, got %s", updatedNode.Status)
		}

		if updatedNode.Result == nil {
			t.Fatal("Result should not be nil")
		}

		if !updatedNode.Result.Success {
			t.Error("Result should indicate success")
		}
	})
}

func TestHierarchicalPlan(t *testing.T) {
	t.Run("IsAtomic", func(t *testing.T) {
		atomicGoal := NewGoal("Atomic", "Atomic goal", WorldState{"a": 1}, 1.0)
		action := NewSimpleAction("Action", "Do it", WorldState{}, WorldState{"a": 1}, 1.0, nil)

		atomicPlan := &HierarchicalPlan{
			Goal:     atomicGoal,
			Subplans: nil,
			Actions:  []Action{action},
			Depth:    0,
		}

		if !atomicPlan.IsAtomic() {
			t.Error("Plan with actions and no subplans should be atomic")
		}

		compositeGoal := NewGoal("Composite", "Composite goal", WorldState{"a": 1}, 1.0)
		compositePlan := &HierarchicalPlan{
			Goal:     compositeGoal,
			Subplans: []*HierarchicalPlan{atomicPlan},
			Actions:  nil,
			Depth:    0,
		}

		if compositePlan.IsAtomic() {
			t.Error("Plan with subplans should not be atomic")
		}
	})

	t.Run("AllActions", func(t *testing.T) {
		action1 := NewSimpleAction("Action1", "A1", WorldState{}, WorldState{"a": 1}, 1.0, nil)
		action2 := NewSimpleAction("Action2", "A2", WorldState{}, WorldState{"b": 2}, 1.0, nil)

		atomicPlan1 := &HierarchicalPlan{
			Goal:     NewGoal("G1", "G1", WorldState{"a": 1}, 1.0),
			Subplans: nil,
			Actions:  []Action{action1},
			Depth:    1,
		}

		atomicPlan2 := &HierarchicalPlan{
			Goal:     NewGoal("G2", "G2", WorldState{"b": 2}, 1.0),
			Subplans: nil,
			Actions:  []Action{action2},
			Depth:    1,
		}

		compositePlan := &HierarchicalPlan{
			Goal:     NewGoal("Root", "Root", WorldState{"a": 1, "b": 2}, 10.0),
			Subplans: []*HierarchicalPlan{atomicPlan1, atomicPlan2},
			Actions:  nil,
			Depth:    0,
		}

		allActions := compositePlan.AllActions()

		if len(allActions) != 2 {
			t.Errorf("Expected 2 actions total, got %d", len(allActions))
		}

		if allActions[0].Name() != "Action1" {
			t.Errorf("Expected Action1 first, got %s", allActions[0].Name())
		}

		if allActions[1].Name() != "Action2" {
			t.Errorf("Expected Action2 second, got %s", allActions[1].Name())
		}
	})
}

func TestGraphStatus(t *testing.T) {
	t.Run("IsComplete", func(t *testing.T) {
		status := &GraphStatus{
			TotalNodes:     5,
			CompletedNodes: 3,
			SkippedNodes:   2,
			PendingNodes:   0,
			RunningNodes:   0,
			FailedNodes:    0,
		}

		if !status.IsComplete() {
			t.Error("Status should be complete when no pending or running nodes")
		}

		status.PendingNodes = 1
		if status.IsComplete() {
			t.Error("Status should not be complete with pending nodes")
		}
	})

	t.Run("HasFailures", func(t *testing.T) {
		status := &GraphStatus{
			TotalNodes:     5,
			CompletedNodes: 4,
			FailedNodes:    1,
		}

		if !status.HasFailures() {
			t.Error("Status should have failures")
		}

		status.FailedNodes = 0
		if status.HasFailures() {
			t.Error("Status should not have failures")
		}
	})
}
