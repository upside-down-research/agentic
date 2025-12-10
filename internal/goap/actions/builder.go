package actions

import (
	"upside-down-research.com/oss/agentic/internal/goap"
)

// ActionBuilder builds the available actions for the GOAP planner.
// It creates actions dynamically based on the context and requirements.
type ActionBuilder struct {
	ctx               *ActionContext
	ticketPath        string
	runID             string
	plannerPrompt     string
	implementPrompt   string
	maxPlanElements   int
}

// NewActionBuilder creates a new ActionBuilder.
func NewActionBuilder(ctx *ActionContext, ticketPath, runID, plannerPrompt, implementPrompt string) *ActionBuilder {
	return &ActionBuilder{
		ctx:             ctx,
		ticketPath:      ticketPath,
		runID:           runID,
		plannerPrompt:   plannerPrompt,
		implementPrompt: implementPrompt,
		maxPlanElements: 20, // Default max plan elements to generate actions for
	}
}

// SetMaxPlanElements sets the maximum number of plan elements to generate actions for.
func (b *ActionBuilder) SetMaxPlanElements(max int) {
	b.maxPlanElements = max
}

// BuildInitialActions builds the core actions needed to start the workflow.
// This includes reading the ticket, generating the plan, and writing the plan.
func (b *ActionBuilder) BuildInitialActions() []goap.Action {
	actions := []goap.Action{
		NewReadTicketAction(b.ctx, b.ticketPath),
		NewGeneratePlanAction(b.ctx, b.plannerPrompt),
		NewWritePlanAction(b.ctx, b.runID),
	}
	return actions
}

// BuildImplementationActions builds actions for implementing and writing code.
// Since we don't know how many plan elements there will be ahead of time,
// we create actions for a reasonable maximum number.
func (b *ActionBuilder) BuildImplementationActions() []goap.Action {
	actions := []goap.Action{}

	for i := 0; i < b.maxPlanElements; i++ {
		actions = append(actions,
			NewImplementCodeAction(b.ctx, b.implementPrompt, i),
			NewWriteCodeAction(b.ctx, i, b.runID),
		)
	}

	return actions
}

// BuildAllActions builds all available actions for the planner.
func (b *ActionBuilder) BuildAllActions() []goap.Action {
	actions := []goap.Action{}
	actions = append(actions, b.BuildInitialActions()...)
	actions = append(actions, b.BuildImplementationActions()...)
	return actions
}

// BuildGoalForCompletePipeline creates a goal that represents completing
// the entire pipeline: read ticket, generate plan, implement all code, write everything.
func (b *ActionBuilder) BuildGoalForCompletePipeline(numPlanElements int) *goap.Goal {
	desiredState := goap.NewWorldState()

	// Core requirements
	desiredState.Set("ticket_read", true)
	desiredState.Set("plan_generated", true)
	desiredState.Set("plan_written", true)

	// All plan elements should be implemented and written
	for i := 0; i < numPlanElements; i++ {
		desiredState.Set("code_implemented_"+string(rune('0'+i)), true)
		desiredState.Set("code_written_"+string(rune('0'+i)), true)
	}

	return goap.NewGoal(
		"CompletePipeline",
		"Complete the full agentic pipeline: read, plan, implement, and write all code",
		desiredState,
		100.0, // High priority
	)
}

// BuildGoalForPlanning creates a goal that only requires planning (no implementation).
func (b *ActionBuilder) BuildGoalForPlanning() *goap.Goal {
	desiredState := goap.NewWorldState()
	desiredState.Set("ticket_read", true)
	desiredState.Set("plan_generated", true)
	desiredState.Set("plan_written", true)

	return goap.NewGoal(
		"PlanOnly",
		"Generate and write a plan without implementing code",
		desiredState,
		80.0,
	)
}

// BuildGoalForImplementation creates a goal for implementing a specific number of plan elements.
func (b *ActionBuilder) BuildGoalForImplementation(numPlanElements int) *goap.Goal {
	desiredState := goap.NewWorldState()

	// Assume planning is already done
	desiredState.Set("plan_generated", true)

	// All plan elements should be implemented and written
	for i := 0; i < numPlanElements; i++ {
		desiredState.Set("code_implemented_"+string(rune('0'+i)), true)
		desiredState.Set("code_written_"+string(rune('0'+i)), true)
	}

	return goap.NewGoal(
		"ImplementCode",
		"Implement all code for the generated plan",
		desiredState,
		90.0,
	)
}
