package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/charmbracelet/log"
	"upside-down-research.com/oss/agentic/internal/goap"
	"upside-down-research.com/oss/agentic/internal/llm"
)

// ActionContext holds shared resources needed by actions to execute.
// This includes the LLM server, run tracking, prompts, etc.
type ActionContext struct {
	LLM        llm.Server
	Run        RunTracker
	Jobname    string
	AgentID    string
	OutputPath string
}

// RunTracker defines the interface for tracking LLM runs and answers.
type RunTracker interface {
	AnswerAndVerify(params *llm.AnswerMeParams, finalOutput any) (string, error)
	AppendRecord(query string, answer string, takes []string)
}

// ReadTicketAction reads the input ticket/specification file.
// Complexity: Low (simple file read, no LLM calls)
type ReadTicketAction struct {
	*goap.BaseAction
	ctx        *ActionContext
	ticketPath string
}

func NewReadTicketAction(ctx *ActionContext, ticketPath string) *ReadTicketAction {
	return &ReadTicketAction{
		BaseAction: goap.NewBaseAction(
			"ReadTicket",
			"Read the input ticket/specification file",
			goap.WorldState{}, // No preconditions
			goap.WorldState{
				"ticket_read":    true,
				"ticket_content": "",
			},
			1.0, // Low complexity: simple file read
		),
		ctx:        ctx,
		ticketPath: ticketPath,
	}
}

func (a *ReadTicketAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("action '%s' cannot execute: preconditions not met", a.Name())
	}

	log.Info("Reading ticket file", "path", a.ticketPath)

	bytes, err := os.ReadFile(a.ticketPath)
	if err != nil {
		return fmt.Errorf("failed to read ticket: %w", err)
	}

	current.Set("ticket_read", true)
	current.Set("ticket_content", string(bytes))

	log.Info("Ticket read successfully", "size", len(bytes))
	return nil
}

func (a *ReadTicketAction) Clone() goap.Action {
	return NewReadTicketAction(a.ctx, a.ticketPath)
}

// GeneratePlanAction generates a plan using LLM with review quality gate.
// Complexity: High (multiple LLM calls with review loop)
type GeneratePlanAction struct {
	*goap.BaseAction
	ctx          *ActionContext
	plannerPrompt string
}

func NewGeneratePlanAction(ctx *ActionContext, plannerPrompt string) *GeneratePlanAction {
	return &GeneratePlanAction{
		BaseAction: goap.NewBaseAction(
			"GeneratePlan",
			"Generate a plan from the ticket using LLM with self-review quality gate",
			goap.WorldState{
				"ticket_read": true,
			},
			goap.WorldState{
				"plan_generated": true,
				"plan_data":      nil,
			},
			10.0, // High complexity: multiple LLM calls with review
		),
		ctx:           ctx,
		plannerPrompt: plannerPrompt,
	}
}

func (a *GeneratePlanAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("action '%s' cannot execute: preconditions not met", a.Name())
	}

	ticketContent := current.Get("ticket_content").(string)
	query := a.plannerPrompt + "\n" + ticketContent

	log.Info("Generating plan via LLM with quality gate")

	// This uses the AnswerAndVerify which includes a review quality gate
	var plans PlanCollection
	_, err := a.ctx.Run.AnswerAndVerify(
		&llm.AnswerMeParams{
			LLM:     a.ctx.LLM,
			Jobname: a.ctx.Jobname,
			AgentId: a.ctx.AgentID,
			Query:   query,
		},
		&plans,
	)

	if err != nil {
		return fmt.Errorf("failed to generate plan: %w", err)
	}

	current.Set("plan_generated", true)
	current.Set("plan_data", plans)

	log.Info("Plan generated successfully", "numPlans", len(plans.Plans))
	return nil
}

func (a *GeneratePlanAction) Clone() goap.Action {
	return NewGeneratePlanAction(a.ctx, a.plannerPrompt)
}

// PlanCollection matches the structure in main.go
type PlanCollection struct {
	Plans []Plan `json:"plans"`
}

type Plan struct {
	Name       string         `json:"name"`
	SystemType string         `json:"type"`
	Rationale  string         `json:"rationale"`
	Definition PlanDefinition `json:"definition"`
}

type PlanDefinition struct {
	Inputs   []InOut `json:"inputs"`
	Outputs  []InOut `json:"outputs"`
	Behavior string  `json:"behavior"`
}

type InOut struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ImplementCodeAction implements code for a single plan element with quality gates.
// Subactions: Generate code, Review code
// Complexity: Very High (multiple LLM calls, code generation + review)
type ImplementCodeAction struct {
	*goap.BaseAction
	ctx             *ActionContext
	implementPrompt string
	planIndex       int
}

func NewImplementCodeAction(ctx *ActionContext, implementPrompt string, planIndex int) *ImplementCodeAction {
	return &ImplementCodeAction{
		BaseAction: goap.NewBaseAction(
			fmt.Sprintf("ImplementCode[%d]", planIndex),
			fmt.Sprintf("Implement code for plan element %d with quality gates", planIndex),
			goap.WorldState{
				"plan_generated": true,
			},
			goap.WorldState{
				fmt.Sprintf("code_implemented_%d", planIndex): true,
			},
			15.0, // Very high complexity: code generation with review
		),
		ctx:             ctx,
		implementPrompt: implementPrompt,
		planIndex:       planIndex,
	}
}

func (a *ImplementCodeAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("action '%s' cannot execute: preconditions not met", a.Name())
	}

	planData := current.Get("plan_data").(PlanCollection)
	if a.planIndex >= len(planData.Plans) {
		return fmt.Errorf("plan index %d out of range", a.planIndex)
	}

	plan := planData.Plans[a.planIndex]
	log.Info("Implementing code for plan", "name", plan.Name, "index", a.planIndex)

	planJSON, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	var implementation ImplementedPlan
	_, err = a.ctx.Run.AnswerAndVerify(
		&llm.AnswerMeParams{
			LLM:     a.ctx.LLM,
			Jobname: a.ctx.Jobname,
			AgentId: a.ctx.AgentID,
			Query:   a.implementPrompt + "\n" + string(planJSON),
		},
		&implementation,
	)

	if err != nil {
		return fmt.Errorf("failed to implement code: %w", err)
	}

	// Store implementation in world state
	current.Set(fmt.Sprintf("code_implemented_%d", a.planIndex), true)
	current.Set(fmt.Sprintf("code_data_%d", a.planIndex), implementation)

	log.Info("Code implemented successfully", "numFiles", len(implementation.Code))
	return nil
}

func (a *ImplementCodeAction) Clone() goap.Action {
	return NewImplementCodeAction(a.ctx, a.implementPrompt, a.planIndex)
}

// ImplementedPlan matches the structure in main.go
type ImplementedPlan struct {
	Environment    string           `json:"environment"`
	CodingLanguage string           `json:"coding_language"`
	Code           []CodeDefinition `json:"code"`
}

type CodeDefinition struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// WriteCodeAction writes generated code to disk.
// Complexity: Low (simple file writes, no LLM calls)
type WriteCodeAction struct {
	*goap.BaseAction
	ctx       *ActionContext
	planIndex int
	runID     string
}

func NewWriteCodeAction(ctx *ActionContext, planIndex int, runID string) *WriteCodeAction {
	return &WriteCodeAction{
		BaseAction: goap.NewBaseAction(
			fmt.Sprintf("WriteCode[%d]", planIndex),
			fmt.Sprintf("Write generated code for plan %d to disk", planIndex),
			goap.WorldState{
				fmt.Sprintf("code_implemented_%d", planIndex): true,
			},
			goap.WorldState{
				fmt.Sprintf("code_written_%d", planIndex): true,
			},
			2.0, // Low-medium complexity: file I/O operations
		),
		ctx:       ctx,
		planIndex: planIndex,
		runID:     runID,
	}
}

func (a *WriteCodeAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("action '%s' cannot execute: preconditions not met", a.Name())
	}

	implementation := current.Get(fmt.Sprintf("code_data_%d", a.planIndex)).(ImplementedPlan)
	outputDir := path.Join(a.ctx.OutputPath, a.runID)

	log.Info("Writing code to disk", "outputDir", outputDir, "numFiles", len(implementation.Code))

	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, code := range implementation.Code {
		filePath := path.Join(outputDir, code.Filename)
		err := os.WriteFile(filePath, []byte(code.Content), 0644)
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", code.Filename, err)
		}
		log.Info("Code written", "file", code.Filename)
	}

	current.Set(fmt.Sprintf("code_written_%d", a.planIndex), true)
	return nil
}

func (a *WriteCodeAction) Clone() goap.Action {
	return NewWriteCodeAction(a.ctx, a.planIndex, a.runID)
}

// WritePlanAction writes the final plan to disk.
// Complexity: Low (simple file write)
type WritePlanAction struct {
	*goap.BaseAction
	ctx   *ActionContext
	runID string
}

func NewWritePlanAction(ctx *ActionContext, runID string) *WritePlanAction {
	return &WritePlanAction{
		BaseAction: goap.NewBaseAction(
			"WritePlan",
			"Write the final plan to disk",
			goap.WorldState{
				"plan_generated": true,
			},
			goap.WorldState{
				"plan_written": true,
			},
			1.0, // Low complexity: simple file write
		),
		ctx:   ctx,
		runID: runID,
	}
}

func (a *WritePlanAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("action '%s' cannot execute: preconditions not met", a.Name())
	}

	planData := current.Get("plan_data").(PlanCollection)
	planJSON, err := json.MarshalIndent(planData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	outputPath := path.Join(a.ctx.OutputPath, a.runID, "plan.txt")
	err = os.WriteFile(outputPath, planJSON, 0644)
	if err != nil {
		return fmt.Errorf("failed to write plan: %w", err)
	}

	current.Set("plan_written", true)
	log.Info("Plan written to disk", "path", outputPath)
	return nil
}

func (a *WritePlanAction) Clone() goap.Action {
	return NewWritePlanAction(a.ctx, a.runID)
}
