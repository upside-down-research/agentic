package goap

import "fmt"

// Goal represents a desired state that the agent wants to achieve.
// It contains target conditions that must be satisfied in the WorldState.
type Goal struct {
	// Name is a human-readable identifier for this goal
	name string

	// Description explains what this goal accomplishes
	description string

	// DesiredState contains the WorldState conditions that must be satisfied
	desiredState WorldState

	// Priority indicates the importance of this goal (higher = more important)
	// Can be used when an agent has multiple competing goals
	priority float64
}

// NewGoal creates a new Goal with the given parameters.
func NewGoal(name, description string, desiredState WorldState, priority float64) *Goal {
	return &Goal{
		name:         name,
		description:  description,
		desiredState: desiredState,
		priority:     priority,
	}
}

// Name returns the goal's name.
func (g *Goal) Name() string {
	return g.name
}

// Description returns the goal's description.
func (g *Goal) Description() string {
	return g.description
}

// DesiredState returns the WorldState conditions this goal wants to achieve.
func (g *Goal) DesiredState() WorldState {
	return g.desiredState
}

// Priority returns the priority of this goal.
func (g *Goal) Priority() float64 {
	return g.priority
}

// IsSatisfied checks if the goal is satisfied by the current WorldState.
func (g *Goal) IsSatisfied(current WorldState) bool {
	return current.Matches(g.desiredState)
}

// Distance calculates how far the current state is from satisfying this goal.
// This is used as a heuristic for planning.
func (g *Goal) Distance(current WorldState) int {
	return current.Distance(g.desiredState)
}

// String returns a string representation of the goal.
func (g *Goal) String() string {
	return fmt.Sprintf("Goal[%s: %s, desired=%s, priority=%.2f]",
		g.name, g.description, g.desiredState, g.priority)
}

// Clone creates a copy of this goal.
func (g *Goal) Clone() *Goal {
	return &Goal{
		name:         g.name,
		description:  g.description,
		desiredState: g.desiredState.Clone(),
		priority:     g.priority,
	}
}

// GoalSet represents a collection of goals that the agent might pursue.
// Useful when the agent needs to choose between or combine multiple objectives.
type GoalSet struct {
	goals []*Goal
}

// NewGoalSet creates a new empty GoalSet.
func NewGoalSet() *GoalSet {
	return &GoalSet{
		goals: make([]*Goal, 0),
	}
}

// Add adds a goal to the set.
func (gs *GoalSet) Add(goal *Goal) {
	gs.goals = append(gs.goals, goal)
}

// Goals returns all goals in the set.
func (gs *GoalSet) Goals() []*Goal {
	return gs.goals
}

// HighestPriority returns the goal with the highest priority.
// Returns nil if the set is empty.
func (gs *GoalSet) HighestPriority() *Goal {
	if len(gs.goals) == 0 {
		return nil
	}

	highest := gs.goals[0]
	for _, goal := range gs.goals[1:] {
		if goal.priority > highest.priority {
			highest = goal
		}
	}
	return highest
}

// MostAchievable returns the goal that is closest to being satisfied,
// based on the distance heuristic from the current state.
// Returns nil if the set is empty.
func (gs *GoalSet) MostAchievable(current WorldState) *Goal {
	if len(gs.goals) == 0 {
		return nil
	}

	mostAchievable := gs.goals[0]
	minDistance := mostAchievable.Distance(current)

	for _, goal := range gs.goals[1:] {
		distance := goal.Distance(current)
		if distance < minDistance {
			minDistance = distance
			mostAchievable = goal
		}
	}

	return mostAchievable
}

// Satisfied returns all goals that are currently satisfied.
func (gs *GoalSet) Satisfied(current WorldState) []*Goal {
	satisfied := make([]*Goal, 0)
	for _, goal := range gs.goals {
		if goal.IsSatisfied(current) {
			satisfied = append(satisfied, goal)
		}
	}
	return satisfied
}

// Unsatisfied returns all goals that are not currently satisfied.
func (gs *GoalSet) Unsatisfied(current WorldState) []*Goal {
	unsatisfied := make([]*Goal, 0)
	for _, goal := range gs.goals {
		if !goal.IsSatisfied(current) {
			unsatisfied = append(unsatisfied, goal)
		}
	}
	return unsatisfied
}
