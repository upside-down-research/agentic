package goap

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
)

// GraphExecutor executes a persisted plan graph with minimal context loading.
// It loads only the necessary context for each node during execution, keeping
// LLM context windows focused and efficient.
type GraphExecutor struct {
	persistence *GraphPersistence
	actions     map[string]Action
	runID       string
}

// NewGraphExecutor creates a new graph executor.
func NewGraphExecutor(persistence *GraphPersistence, runID string) *GraphExecutor {
	return &GraphExecutor{
		persistence: persistence,
		actions:     make(map[string]Action),
		runID:       runID,
	}
}

// RegisterAction registers an action that can be executed by name.
func (ge *GraphExecutor) RegisterAction(action Action) {
	ge.actions[action.Name()] = action
}

// RegisterActions registers multiple actions.
func (ge *GraphExecutor) RegisterActions(actions []Action) {
	for _, action := range actions {
		ge.RegisterAction(action)
	}
}

// Execute executes the plan graph starting from the root node.
func (ge *GraphExecutor) Execute(ctx context.Context, initialState WorldState) error {
	graph, err := ge.persistence.LoadGraph(ge.runID)
	if err != nil {
		return fmt.Errorf("failed to load graph: %w", err)
	}

	log.Info("Starting graph execution", "rootNode", graph.RootNodeID, "totalNodes", graph.Metadata.TotalNodes)

	// Execute from root
	currentState := initialState.Clone()
	return ge.executeNode(ctx, graph, graph.RootNodeID, currentState)
}

// executeNode executes a single node and its children recursively.
func (ge *GraphExecutor) executeNode(ctx context.Context, graph *PlanGraph, nodeID string, currentState WorldState) error {
	// Load minimal context for this node
	nodeContext, err := ge.persistence.LoadNodeContext(ge.runID, nodeID)
	if err != nil {
		return fmt.Errorf("failed to load node context: %w", err)
	}

	node := nodeContext.Node
	log.Info("Executing node",
		"id", node.ID,
		"goal", node.GoalName,
		"depth", node.Depth,
		"atomic", node.IsAtomic,
	)

	// Update status to running
	err = ge.persistence.UpdateNodeStatus(ge.runID, nodeID, StatusRunning, nil)
	if err != nil {
		log.Warn("Failed to update node status", "error", err)
	}

	// Check if goal is already satisfied
	goalState := NewWorldState()
	for k, v := range node.DesiredState {
		goalState.Set(k, v)
	}

	if currentState.Matches(goalState) {
		log.Info("Goal already satisfied, skipping node", "nodeID", nodeID)
		err = ge.persistence.UpdateNodeStatus(ge.runID, nodeID, StatusSkipped, &NodeResult{
			Success: true,
		})
		if err != nil {
			log.Warn("Failed to update node status", "error", err)
		}
		return nil
	}

	// Execute based on node type
	var execErr error
	if node.IsAtomic {
		execErr = ge.executeAtomicNode(ctx, node, currentState)
	} else {
		execErr = ge.executeCompositeNode(ctx, graph, node, currentState)
	}

	// Update status based on result
	if execErr != nil {
		log.Error("Node execution failed", "nodeID", nodeID, "error", execErr)
		err = ge.persistence.UpdateNodeStatus(ge.runID, nodeID, StatusFailed, &NodeResult{
			Success:      false,
			ErrorMessage: execErr.Error(),
		})
		if err != nil {
			log.Warn("Failed to update node status", "error", err)
		}
		return execErr
	}

	// Success - capture state changes
	stateChanges := make(map[string]interface{})
	for k, v := range goalState {
		if currentState.Get(k) != v {
			stateChanges[k] = v
		}
	}

	err = ge.persistence.UpdateNodeStatus(ge.runID, nodeID, StatusCompleted, &NodeResult{
		Success:      true,
		StateChanges: stateChanges,
	})
	if err != nil {
		log.Warn("Failed to update node status", "error", err)
	}

	log.Info("Node execution completed", "nodeID", nodeID, "goal", node.GoalName)
	return nil
}

// executeAtomicNode executes an atomic node by running its actions.
func (ge *GraphExecutor) executeAtomicNode(ctx context.Context, node *GraphNode, currentState WorldState) error {
	log.Info("Executing atomic node actions", "nodeID", node.ID, "numActions", len(node.ActionNames))

	for i, actionName := range node.ActionNames {
		action, exists := ge.actions[actionName]
		if !exists {
			return fmt.Errorf("action not found: %s", actionName)
		}

		log.Info("Executing action", "index", i, "action", actionName)

		err := action.Execute(ctx, currentState)
		if err != nil {
			return fmt.Errorf("action %s failed: %w", actionName, err)
		}

		// Small delay between actions to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// executeCompositeNode executes a composite node by executing its children.
func (ge *GraphExecutor) executeCompositeNode(ctx context.Context, graph *PlanGraph, node *GraphNode, currentState WorldState) error {
	log.Info("Executing composite node children", "nodeID", node.ID, "numChildren", len(node.ChildIDs))

	for i, childID := range node.ChildIDs {
		log.Info("Executing child node", "index", i, "childID", childID)

		err := ge.executeNode(ctx, graph, childID, currentState)
		if err != nil {
			return fmt.Errorf("child node %s failed: %w", childID, err)
		}
	}

	return nil
}

// GetGraphStatus returns the current execution status of the graph.
func (ge *GraphExecutor) GetGraphStatus() (*GraphStatus, error) {
	graph, err := ge.persistence.LoadGraph(ge.runID)
	if err != nil {
		return nil, fmt.Errorf("failed to load graph: %w", err)
	}

	status := &GraphStatus{
		TotalNodes:     len(graph.Nodes),
		PendingNodes:   0,
		RunningNodes:   0,
		CompletedNodes: 0,
		FailedNodes:    0,
		SkippedNodes:   0,
	}

	for _, node := range graph.Nodes {
		switch node.Status {
		case StatusPending:
			status.PendingNodes++
		case StatusRunning:
			status.RunningNodes++
		case StatusCompleted:
			status.CompletedNodes++
		case StatusFailed:
			status.FailedNodes++
		case StatusSkipped:
			status.SkippedNodes++
		}
	}

	return status, nil
}

// GraphStatus represents the execution status of a plan graph.
type GraphStatus struct {
	TotalNodes     int `json:"total_nodes"`
	PendingNodes   int `json:"pending_nodes"`
	RunningNodes   int `json:"running_nodes"`
	CompletedNodes int `json:"completed_nodes"`
	FailedNodes    int `json:"failed_nodes"`
	SkippedNodes   int `json:"skipped_nodes"`
}

// IsComplete returns true if all nodes are either completed or skipped.
func (gs *GraphStatus) IsComplete() bool {
	return gs.PendingNodes == 0 && gs.RunningNodes == 0
}

// HasFailures returns true if any nodes failed.
func (gs *GraphStatus) HasFailures() bool {
	return gs.FailedNodes > 0
}
