package goap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"upside-down-research.com/oss/agentic/internal/llm"
)

// LLMGoalRefiner uses an LLM to decompose high-level goals into subgoals.
// This is the key component that makes the Agentic GOAP system intelligent -
// it uses the LLM's reasoning capabilities to plan hierarchically.
type LLMGoalRefiner struct {
	llm         llm.Server
	jobname     string
	agentID     string
	atomicTypes map[string]bool // Types of goals that are atomic (cannot be refined)
}

// NewLLMGoalRefiner creates a new LLM-based goal refiner.
func NewLLMGoalRefiner(llmServer llm.Server, jobname, agentID string) *LLMGoalRefiner {
	return &LLMGoalRefiner{
		llm:     llmServer,
		jobname: jobname,
		agentID: agentID,
		atomicTypes: map[string]bool{
			"read_file":       true,
			"write_file":      true,
			"run_command":     true,
			"llm_prompt":      true,
			"simple_action":   true,
		},
	}
}

// AddAtomicType registers a goal type as atomic (cannot be refined further).
func (r *LLMGoalRefiner) AddAtomicType(goalType string) {
	r.atomicTypes[goalType] = true
}

// IsAtomic determines if a goal is atomic based on naming conventions or metadata.
func (r *LLMGoalRefiner) IsAtomic(goal *Goal, current WorldState) bool {
	// Check if the goal name contains markers for atomic goals
	goalName := strings.ToLower(goal.Name())

	// Goals that directly manipulate single state variables are usually atomic
	if len(goal.DesiredState()) == 1 {
		return true
	}

	// Check if goal has an "atomic" marker in its name
	if strings.Contains(goalName, "[atomic]") {
		return true
	}

	// Check if the desired state only requires simple changes
	distance := goal.Distance(current)
	if distance <= 1 {
		return true
	}

	return false
}

// Refine uses the LLM to decompose a goal into subgoals.
func (r *LLMGoalRefiner) Refine(ctx context.Context, goal *Goal, current WorldState) ([]*Goal, error) {
	log.Info("Refining goal with LLM", "goal", goal.Name())

	prompt := r.buildRefinementPrompt(goal, current)

	// Query the LLM
	response, err := llm.AnswerMe(&llm.AnswerMeParams{
		LLM:     r.llm,
		Jobname: r.jobname,
		AgentId: r.agentID,
		Query:   prompt,
	})

	if err != nil {
		return nil, fmt.Errorf("LLM query failed: %w", err)
	}

	// Parse the response
	var refinement GoalRefinement
	err = json.Unmarshal([]byte(response), &refinement)
	if err != nil {
		log.Error("Failed to parse LLM response", "error", err, "response", response)
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Convert the refinement to Goal objects
	subgoals := make([]*Goal, 0, len(refinement.Subgoals))
	for i, subgoalSpec := range refinement.Subgoals {
		desiredState := NewWorldState()
		for key, value := range subgoalSpec.DesiredState {
			desiredState.Set(key, value)
		}

		subgoal := NewGoal(
			subgoalSpec.Name,
			subgoalSpec.Description,
			desiredState,
			float64(len(refinement.Subgoals)-i), // Earlier subgoals have higher priority
		)

		subgoals = append(subgoals, subgoal)
	}

	log.Info("Goal refined successfully", "goal", goal.Name(), "numSubgoals", len(subgoals))
	return subgoals, nil
}

func (r *LLMGoalRefiner) buildRefinementPrompt(goal *Goal, current WorldState) string {
	return fmt.Sprintf(`You are a goal-oriented planning agent. Your task is to decompose a high-level goal into a sequence of subgoals.

Current World State:
%s

Goal to Achieve:
Name: %s
Description: %s
Desired State: %s

Instructions:
1. Analyze the current state and the goal
2. Break down the goal into a logical sequence of subgoals
3. Each subgoal should be simpler and more concrete than the parent goal
4. Subgoals should be ordered such that achieving them in sequence accomplishes the parent goal
5. Consider dependencies between subgoals (earlier subgoals may be prerequisites for later ones)

Respond with a JSON object in this format:
{
  "rationale": "Explanation of your decomposition strategy",
  "subgoals": [
    {
      "name": "Subgoal1Name",
      "description": "What this subgoal accomplishes",
      "desired_state": {
        "key1": "value1",
        "key2": "value2"
      }
    },
    {
      "name": "Subgoal2Name",
      "description": "What this subgoal accomplishes",
      "desired_state": {
        "key3": "value3"
      }
    }
  ]
}

Important:
- The subgoals should be ordered sequentially
- Each subgoal's desired_state should represent a meaningful intermediate state
- Make subgoals concrete and achievable
- Aim for 2-5 subgoals (avoid over-decomposition)

Return ONLY valid JSON, starting with '{' and ending with '}'.`,
		current.String(),
		goal.Name(),
		goal.Description(),
		goal.DesiredState().String(),
	)
}

// GoalRefinement represents the LLM's response when refining a goal.
type GoalRefinement struct {
	Rationale string        `json:"rationale"`
	Subgoals  []SubgoalSpec `json:"subgoals"`
}

// SubgoalSpec represents a subgoal specification from the LLM.
type SubgoalSpec struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	DesiredState map[string]interface{} `json:"desired_state"`
}
