package goap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
)

// PlanGraph represents the hierarchical plan as a graph structure suitable for
// persistence. This enables saving the plan to disk and loading minimal context
// per node during execution.
type PlanGraph struct {
	RootNodeID string                `json:"root_node_id"`
	Nodes      map[string]*GraphNode `json:"nodes"`
	Metadata   GraphMetadata         `json:"metadata"`
}

// GraphNode represents a single node in the plan graph.
type GraphNode struct {
	ID           string                 `json:"id"`
	GoalName     string                 `json:"goal_name"`
	GoalDesc     string                 `json:"goal_description"`
	DesiredState map[string]interface{} `json:"desired_state"`
	ParentID     string                 `json:"parent_id,omitempty"`
	ChildIDs     []string               `json:"child_ids,omitempty"`
	ActionNames  []string               `json:"action_names,omitempty"`
	IsAtomic     bool                   `json:"is_atomic"`
	Depth        int                    `json:"depth"`
	Status       NodeStatus             `json:"status"`
	Result       *NodeResult            `json:"result,omitempty"`
}

// NodeStatus represents the execution status of a node.
type NodeStatus string

const (
	StatusPending   NodeStatus = "pending"
	StatusRunning   NodeStatus = "running"
	StatusCompleted NodeStatus = "completed"
	StatusFailed    NodeStatus = "failed"
	StatusSkipped   NodeStatus = "skipped"
)

// NodeResult stores the execution result of a node.
type NodeResult struct {
	Success      bool                   `json:"success"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	StateChanges map[string]interface{} `json:"state_changes,omitempty"`
}

// GraphMetadata contains metadata about the plan graph.
type GraphMetadata struct {
	AgentID       string `json:"agent_id"`
	CreatedAt     string `json:"created_at"`
	TotalNodes    int    `json:"total_nodes"`
	MaxDepth      int    `json:"max_depth"`
}

// NewPlanGraph creates a new empty plan graph.
func NewPlanGraph(agentID string) *PlanGraph {
	return &PlanGraph{
		Nodes: make(map[string]*GraphNode),
		Metadata: GraphMetadata{
			AgentID: agentID,
		},
	}
}

// BuildGraphFromPlan converts a HierarchicalPlan into a PlanGraph for persistence.
func BuildGraphFromPlan(plan *HierarchicalPlan, agentID string) *PlanGraph {
	graph := NewPlanGraph(agentID)
	nodeCounter := 0

	var buildNode func(*HierarchicalPlan, string) string
	buildNode = func(hp *HierarchicalPlan, parentID string) string {
		nodeCounter++
		nodeID := fmt.Sprintf("node_%d", nodeCounter)

		// Convert desired state to map
		desiredState := make(map[string]interface{})
		for k, v := range hp.Goal.DesiredState() {
			desiredState[k] = v
		}

		// Extract action names if atomic
		actionNames := []string{}
		if hp.IsAtomic() && hp.Actions != nil {
			for _, action := range hp.Actions {
				actionNames = append(actionNames, action.Name())
			}
		}

		// Build child nodes
		childIDs := []string{}
		if !hp.IsAtomic() {
			for _, subplan := range hp.Subplans {
				childID := buildNode(subplan, nodeID)
				childIDs = append(childIDs, childID)
			}
		}

		// Create graph node
		node := &GraphNode{
			ID:           nodeID,
			GoalName:     hp.Goal.Name(),
			GoalDesc:     hp.Goal.Description(),
			DesiredState: desiredState,
			ParentID:     parentID,
			ChildIDs:     childIDs,
			ActionNames:  actionNames,
			IsAtomic:     hp.IsAtomic(),
			Depth:        hp.Depth,
			Status:       StatusPending,
		}

		graph.Nodes[nodeID] = node

		return nodeID
	}

	// Build the graph starting from root
	rootID := buildNode(plan, "")
	graph.RootNodeID = rootID

	// Update metadata
	graph.Metadata.TotalNodes = nodeCounter
	graph.Metadata.MaxDepth = calculateMaxDepth(graph)

	return graph
}

func calculateMaxDepth(graph *PlanGraph) int {
	maxDepth := 0
	for _, node := range graph.Nodes {
		if node.Depth > maxDepth {
			maxDepth = node.Depth
		}
	}
	return maxDepth
}

// GraphPersistence handles saving and loading plan graphs to/from disk.
type GraphPersistence struct {
	basePath string
}

// NewGraphPersistence creates a new graph persistence handler.
func NewGraphPersistence(basePath string) *GraphPersistence {
	return &GraphPersistence{
		basePath: basePath,
	}
}

// SaveGraph saves a plan graph to disk.
func (gp *GraphPersistence) SaveGraph(graph *PlanGraph, runID string) error {
	graphDir := filepath.Join(gp.basePath, runID, "graph")
	err := os.MkdirAll(graphDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create graph directory: %w", err)
	}

	// Save the main graph structure
	graphPath := filepath.Join(graphDir, "plan_graph.json")
	graphJSON, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal graph: %w", err)
	}

	err = os.WriteFile(graphPath, graphJSON, 0644)
	if err != nil {
		return fmt.Errorf("failed to write graph file: %w", err)
	}

	// Save individual node files for minimal context loading
	nodesDir := filepath.Join(graphDir, "nodes")
	err = os.MkdirAll(nodesDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create nodes directory: %w", err)
	}

	for nodeID := range graph.Nodes {
		nodeContext := gp.buildNodeContext(graph, nodeID)
		nodePath := filepath.Join(nodesDir, nodeID+".json")
		nodeJSON, err := json.MarshalIndent(nodeContext, "", "  ")
		if err != nil {
			log.Error("Failed to marshal node context", "nodeID", nodeID, "error", err)
			continue
		}

		err = os.WriteFile(nodePath, nodeJSON, 0644)
		if err != nil {
			log.Error("Failed to write node file", "nodeID", nodeID, "error", err)
			continue
		}
	}

	log.Info("Plan graph saved", "path", graphDir, "nodes", len(graph.Nodes))
	return nil
}

// LoadGraph loads a plan graph from disk.
func (gp *GraphPersistence) LoadGraph(runID string) (*PlanGraph, error) {
	graphPath := filepath.Join(gp.basePath, runID, "graph", "plan_graph.json")

	data, err := os.ReadFile(graphPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read graph file: %w", err)
	}

	var graph PlanGraph
	err = json.Unmarshal(data, &graph)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal graph: %w", err)
	}

	log.Info("Plan graph loaded", "nodes", len(graph.Nodes))
	return &graph, nil
}

// LoadNodeContext loads minimal context for a specific node.
// This enables focused LLM execution without loading the entire plan.
func (gp *GraphPersistence) LoadNodeContext(runID, nodeID string) (*NodeContext, error) {
	nodePath := filepath.Join(gp.basePath, runID, "graph", "nodes", nodeID+".json")

	data, err := os.ReadFile(nodePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read node context: %w", err)
	}

	var context NodeContext
	err = json.Unmarshal(data, &context)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal node context: %w", err)
	}

	return &context, nil
}

// UpdateNodeStatus updates the status of a node in the graph.
func (gp *GraphPersistence) UpdateNodeStatus(runID, nodeID string, status NodeStatus, result *NodeResult) error {
	graph, err := gp.LoadGraph(runID)
	if err != nil {
		return err
	}

	node, exists := graph.Nodes[nodeID]
	if !exists {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	node.Status = status
	node.Result = result

	return gp.SaveGraph(graph, runID)
}

// NodeContext represents the minimal context needed to execute a single node.
// This keeps LLM context focused and efficient.
type NodeContext struct {
	Node         *GraphNode   `json:"node"`
	Parent       *GraphNode   `json:"parent,omitempty"`
	Children     []*GraphNode `json:"children,omitempty"`
	Siblings     []*GraphNode `json:"siblings,omitempty"`
	PathFromRoot []string     `json:"path_from_root"`
}

// buildNodeContext creates minimal context for a node.
func (gp *GraphPersistence) buildNodeContext(graph *PlanGraph, nodeID string) *NodeContext {
	node, exists := graph.Nodes[nodeID]
	if !exists {
		return nil
	}

	context := &NodeContext{
		Node:     node,
		Children: []*GraphNode{},
		Siblings: []*GraphNode{},
	}

	// Add parent
	if node.ParentID != "" {
		if parent, exists := graph.Nodes[node.ParentID]; exists {
			context.Parent = parent

			// Add siblings
			for _, siblingID := range parent.ChildIDs {
				if siblingID != nodeID {
					if sibling, exists := graph.Nodes[siblingID]; exists {
						context.Siblings = append(context.Siblings, sibling)
					}
				}
			}
		}
	}

	// Add children
	for _, childID := range node.ChildIDs {
		if child, exists := graph.Nodes[childID]; exists {
			context.Children = append(context.Children, child)
		}
	}

	// Build path from root
	context.PathFromRoot = gp.buildPathFromRoot(graph, nodeID)

	return context
}

func (gp *GraphPersistence) buildPathFromRoot(graph *PlanGraph, nodeID string) []string {
	path := []string{nodeID}
	currentID := nodeID

	for {
		node, exists := graph.Nodes[currentID]
		if !exists || node.ParentID == "" {
			break
		}
		path = append([]string{node.ParentID}, path...)
		currentID = node.ParentID
	}

	return path
}
