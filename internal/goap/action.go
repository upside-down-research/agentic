package goap

import (
	"context"
	"fmt"
)

// Action represents a single action that can be performed by the agent.
// Each action has preconditions (what must be true to execute) and effects (what changes after execution).
// Actions can contain subactions: prompts, tool uses, or other agent behaviors.
type Action interface {
	// Name returns the human-readable name of this action
	Name() string

	// Description returns a detailed description of what this action does
	Description() string

	// Preconditions returns the WorldState conditions that must be satisfied before this action can execute
	Preconditions() WorldState

	// Effects returns the WorldState changes that will occur after this action executes successfully
	Effects() WorldState

	// Cost returns the estimated cost of executing this action (used for planning optimization)
	// Lower costs are preferred during planning
	Cost() float64

	// CanExecute checks if this action can currently execute given the current world state
	CanExecute(current WorldState) bool

	// Execute performs the action, potentially modifying the world state.
	// This may involve LLM prompts, tool uses, or other agent behaviors.
	// Returns an error if execution fails.
	Execute(ctx context.Context, current WorldState) error

	// Clone creates a copy of this action
	Clone() Action
}

// BaseAction provides a default implementation of common Action methods.
// Concrete actions can embed this to avoid boilerplate.
type BaseAction struct {
	name          string
	description   string
	preconditions WorldState
	effects       WorldState
	cost          float64
}

// NewBaseAction creates a new BaseAction with the given parameters.
func NewBaseAction(name, description string, preconditions, effects WorldState, cost float64) *BaseAction {
	return &BaseAction{
		name:          name,
		description:   description,
		preconditions: preconditions,
		effects:       effects,
		cost:          cost,
	}
}

func (a *BaseAction) Name() string {
	return a.name
}

func (a *BaseAction) Description() string {
	return a.description
}

func (a *BaseAction) Preconditions() WorldState {
	return a.preconditions
}

func (a *BaseAction) Effects() WorldState {
	return a.effects
}

func (a *BaseAction) Cost() float64 {
	return a.cost
}

func (a *BaseAction) CanExecute(current WorldState) bool {
	return current.Matches(a.preconditions)
}

// ActionFunc is a function type that can be used to create simple actions.
// It receives the current WorldState and should perform the action's behavior.
type ActionFunc func(ctx context.Context, current WorldState) error

// SimpleAction wraps a BaseAction with an execution function.
// This is useful for creating actions without defining a full struct.
type SimpleAction struct {
	*BaseAction
	executeFunc ActionFunc
}

// NewSimpleAction creates a SimpleAction with the given parameters and execution function.
func NewSimpleAction(name, description string, preconditions, effects WorldState, cost float64, fn ActionFunc) *SimpleAction {
	return &SimpleAction{
		BaseAction:  NewBaseAction(name, description, preconditions, effects, cost),
		executeFunc: fn,
	}
}

func (a *SimpleAction) Execute(ctx context.Context, current WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("action '%s' cannot execute: preconditions not met", a.Name())
	}

	if a.executeFunc == nil {
		return fmt.Errorf("action '%s' has no execution function", a.Name())
	}

	// Execute the action
	if err := a.executeFunc(ctx, current); err != nil {
		return fmt.Errorf("action '%s' execution failed: %w", a.Name(), err)
	}

	// Apply effects to the current world state
	current.Apply(a.effects)

	return nil
}

func (a *SimpleAction) Clone() Action {
	return &SimpleAction{
		BaseAction:  NewBaseAction(a.name, a.description, a.preconditions.Clone(), a.effects.Clone(), a.cost),
		executeFunc: a.executeFunc,
	}
}

// CompositeAction represents an action that consists of multiple subactions.
// Subactions are executed in sequence. This is useful for complex operations
// that involve multiple prompts, tool uses, or other agent behaviors.
type CompositeAction struct {
	*BaseAction
	subactions []Action
}

// NewCompositeAction creates a CompositeAction with the given parameters and subactions.
func NewCompositeAction(name, description string, preconditions, effects WorldState, cost float64, subactions []Action) *CompositeAction {
	return &CompositeAction{
		BaseAction: NewBaseAction(name, description, preconditions, effects, cost),
		subactions: subactions,
	}
}

// AddSubaction adds a subaction to this composite action.
func (a *CompositeAction) AddSubaction(subaction Action) {
	a.subactions = append(a.subactions, subaction)
}

// Subactions returns the list of subactions.
func (a *CompositeAction) Subactions() []Action {
	return a.subactions
}

func (a *CompositeAction) Execute(ctx context.Context, current WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("composite action '%s' cannot execute: preconditions not met", a.Name())
	}

	// Execute each subaction in sequence
	for i, subaction := range a.subactions {
		select {
		case <-ctx.Done():
			return fmt.Errorf("composite action '%s' interrupted at subaction %d: %w", a.Name(), i, ctx.Err())
		default:
			if err := subaction.Execute(ctx, current); err != nil {
				return fmt.Errorf("composite action '%s' failed at subaction %d (%s): %w", a.Name(), i, subaction.Name(), err)
			}
		}
	}

	// Apply the composite action's effects
	current.Apply(a.effects)

	return nil
}

func (a *CompositeAction) Clone() Action {
	clonedSubactions := make([]Action, len(a.subactions))
	for i, sub := range a.subactions {
		clonedSubactions[i] = sub.Clone()
	}

	return &CompositeAction{
		BaseAction: NewBaseAction(a.name, a.description, a.preconditions.Clone(), a.effects.Clone(), a.cost),
		subactions: clonedSubactions,
	}
}
