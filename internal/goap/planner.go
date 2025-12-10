package goap

import (
	"container/heap"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
)

// Plan represents a sequence of actions that will achieve a goal.
type Plan struct {
	Actions []Action
	Cost    float64
}

// String returns a string representation of the plan.
func (p *Plan) String() string {
	if len(p.Actions) == 0 {
		return "Empty Plan"
	}

	parts := make([]string, len(p.Actions))
	for i, action := range p.Actions {
		parts[i] = fmt.Sprintf("%d. %s", i+1, action.Name())
	}

	return fmt.Sprintf("Plan (cost: %.2f):\n%s", p.Cost, strings.Join(parts, "\n"))
}

// Planner finds a sequence of actions to achieve a goal using A* pathfinding.
type Planner struct {
	actions []Action
}

// NewPlanner creates a new Planner with the given available actions.
func NewPlanner(actions []Action) *Planner {
	return &Planner{
		actions: actions,
	}
}

// AddAction adds an action to the planner's available actions.
func (p *Planner) AddAction(action Action) {
	p.actions = append(p.actions, action)
}

// Actions returns the list of available actions.
func (p *Planner) Actions() []Action {
	return p.actions
}

// FindPlan uses A* pathfinding to find the optimal sequence of actions
// that will transform the current WorldState to satisfy the goal.
// Returns nil if no plan can be found.
func (p *Planner) FindPlan(current WorldState, goal *Goal) *Plan {
	log.Info("Starting plan search", "goal", goal.Name(), "current", current.String())

	// Check if goal is already satisfied
	if goal.IsSatisfied(current) {
		log.Info("Goal already satisfied, no actions needed")
		return &Plan{Actions: []Action{}, Cost: 0}
	}

	// Initialize A* data structures
	openSet := &PriorityQueue{}
	heap.Init(openSet)

	// Create starting node
	startNode := &Node{
		state:    current.Clone(),
		actions:  []Action{},
		gCost:    0,
		hCost:    float64(goal.Distance(current)),
		parent:   nil,
	}

	heap.Push(openSet, startNode)
	visited := make(map[string]bool)

	iterations := 0
	maxIterations := 1000 // Prevent infinite loops

	for openSet.Len() > 0 && iterations < maxIterations {
		iterations++

		// Get node with lowest f-cost
		currentNode := heap.Pop(openSet).(*Node)
		stateKey := currentNode.state.String()

		// Check if we've already visited this state
		if visited[stateKey] {
			continue
		}
		visited[stateKey] = true

		log.Debug("Exploring node", "depth", len(currentNode.actions), "fCost", currentNode.FCost(), "state", stateKey)

		// Check if goal is satisfied
		if goal.IsSatisfied(currentNode.state) {
			log.Info("Plan found", "actions", len(currentNode.actions), "cost", currentNode.gCost, "iterations", iterations)
			return &Plan{
				Actions: currentNode.actions,
				Cost:    currentNode.gCost,
			}
		}

		// Expand neighbors by trying each available action
		for _, action := range p.actions {
			if !action.CanExecute(currentNode.state) {
				continue
			}

			// Create new state by applying action effects
			newState := currentNode.state.Clone()
			newState.Apply(action.Effects())

			// Check if we've already visited this state
			newStateKey := newState.String()
			if visited[newStateKey] {
				continue
			}

			// Create new path by adding this action
			newActions := make([]Action, len(currentNode.actions)+1)
			copy(newActions, currentNode.actions)
			newActions[len(currentNode.actions)] = action

			// Calculate costs
			newGCost := currentNode.gCost + action.Cost()
			newHCost := float64(goal.Distance(newState))

			// Create neighbor node
			neighborNode := &Node{
				state:   newState,
				actions: newActions,
				gCost:   newGCost,
				hCost:   newHCost,
				parent:  currentNode,
			}

			heap.Push(openSet, neighborNode)
		}
	}

	if iterations >= maxIterations {
		log.Warn("Plan search reached max iterations", "maxIterations", maxIterations)
	} else {
		log.Warn("No plan found to achieve goal", "goal", goal.Name())
	}

	return nil
}

// Node represents a state in the A* search.
type Node struct {
	state   WorldState
	actions []Action
	gCost   float64 // Cost from start to this node
	hCost   float64 // Heuristic cost from this node to goal
	parent  *Node
	index   int // Required for heap interface
}

// FCost returns the total estimated cost (g + h).
func (n *Node) FCost() float64 {
	return n.gCost + n.hCost
}

// PriorityQueue implements a min-heap for A* nodes based on f-cost.
type PriorityQueue []*Node

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// Lower f-cost has higher priority
	return pq[i].FCost() < pq[j].FCost()
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	node := x.(*Node)
	node.index = n
	*pq = append(*pq, node)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	node := old[n-1]
	old[n-1] = nil  // Avoid memory leak
	node.index = -1 // For safety
	*pq = old[0 : n-1]
	return node
}
