# GOAP - Goal-Oriented Action Planning System

## Overview

The GOAP (Goal-Oriented Action Planning) system is a hierarchical planning framework designed for AI agents. Unlike traditional game AI GOAP systems, this implementation focuses on **deliberative planning** for LLM-based agents, enabling them to decompose high-level goals into concrete, executable action sequences.

## Key Concepts

### Planning vs. Vibe Coding

The Agentic GOAP is a **"planning-focused Claude"** rather than a pair programming assistant. Instead of improvising solutions on the fly, it:

1. Takes an initial high-level goal
2. Uses LLM reasoning to generate refinements and subnodes
3. Recursively decomposes goals into a tree of increasingly specific subgoals
4. Executes the plan with minimal, focused context per node

This approach enables more thoughtful, structured problem-solving compared to reactive coding.

### Core Components

#### 1. WorldState

Represents the current state of the system as key-value pairs:

```go
ws := goap.NewWorldState()
ws.Set("ticket_read", true)
ws.Set("plan_generated", false)
ws.Set("tests_passing", false)
```

#### 2. Goal

Represents a desired state to achieve:

```go
goal := goap.NewGoal(
    "ImplementFeature",
    "Implement the user authentication feature",
    goap.WorldState{
        "feature_implemented": true,
        "tests_passing": true,
        "documentation_written": true,
    },
    100.0, // Priority
)
```

#### 3. Action

Represents an operation that changes the world state:

```go
action := goap.NewSimpleAction(
    "WriteTests",
    "Write unit tests for the feature",
    goap.WorldState{"feature_implemented": true}, // Preconditions
    goap.WorldState{"tests_written": true},       // Effects
    5.0, // Cost (complexity)
    func(ctx context.Context, ws goap.WorldState) error {
        // Execute the action (e.g., prompt LLM, run tools)
        return nil
    },
)
```

**Cost as Complexity**: The `cost` parameter represents the complexity of the action:
- Low cost (1-3): Simple operations (file reads, basic prompts)
- Medium cost (4-7): Moderate operations (code analysis, simple generation)
- High cost (8-10): Complex operations (full code generation with review)
- Very high cost (11+): Multi-step operations with quality gates

#### 4. Hierarchical Planning

The system uses recursive goal refinement:

```go
// Create hierarchical planner
planner := goap.NewPlanner(actions)
refiner := goap.NewLLMGoalRefiner(llmServer, jobname, agentID)
hierarchicalPlanner := goap.NewHierarchicalPlanner(planner, refiner, 10)

// Plan recursively
plan, err := hierarchicalPlanner.PlanHierarchical(ctx, currentState, goal)
```

The planner:
- Checks if a goal is **atomic** (can be achieved directly by actions)
- If not atomic, asks the LLM to **refine** it into subgoals
- Recursively plans for each subgoal
- Builds a tree of goals from abstract to concrete

#### 5. Graph Persistence

Plans are persisted to disk as graph databases:

```go
// Build graph from hierarchical plan
graph := goap.BuildGraphFromPlan(plan, agentID)

// Save to disk
persistence := goap.NewGraphPersistence(outputPath)
err := persistence.SaveGraph(graph, runID)
```

**Why Persist?**
- Enables minimal context loading per node during execution
- Keeps LLM context windows small and focused
- Allows inspection and debugging of the plan
- Supports resumption and incremental execution

#### 6. Minimal Context Execution

The executor loads only necessary context per node:

```go
executor := goap.NewGraphExecutor(persistence, runID)
executor.RegisterActions(availableActions)

// Executes with minimal context per node
err := executor.Execute(ctx, initialState)
```

Each node loads:
- Its own goal and state
- Immediate parent (for context)
- Direct children (for planning ahead)
- Siblings (for coordination)
- Path from root (for overall context)

This keeps LLM prompts focused on the specific task at hand.

## Architecture

### Hierarchical Planning Flow

```
1. High-Level Goal
   "Implement user authentication"

2. LLM Refinement
   ├─ Design auth system architecture
   ├─ Implement core auth logic
   ├─ Add tests
   └─ Write documentation

3. Further Refinement
   Design auth system architecture
   ├─ Review existing patterns
   ├─ Design data models
   └─ Design API endpoints

   Implement core auth logic
   ├─ Implement user registration
   │  ├─ [Atomic] Write registration function
   │  └─ [Atomic] Add input validation
   ├─ Implement login
   └─ Implement token generation

   ... (continues recursively)
```

### Quality Gates in Actions

For code generation actions, quality gates are built into the action structure:

```go
// Composite action with quality gates
codeGenAction := goap.NewCompositeAction(
    "GenerateAndReviewCode",
    "Generate code with quality review",
    preconditions,
    effects,
    15.0, // High complexity
    []goap.Action{
        generateCodeSubaction,  // Prompt LLM for code
        reviewCodeSubaction,    // Review generated code
        runTestsSubaction,      // Run tests
        retryIfFailedSubaction, // Retry on failures
    },
)
```

### Graph Database Structure

When persisted, the plan becomes a graph on disk:

```
output/
└── <run-id>/
    └── graph/
        ├── plan_graph.json       # Full graph structure
        └── nodes/
            ├── node_1.json       # Root node context
            ├── node_2.json       # Child node context
            └── ...
```

Each node file contains minimal context:

```json
{
  "node": {
    "id": "node_5",
    "goal_name": "WriteTests",
    "desired_state": {"tests_written": true},
    "action_names": ["PromptForTests", "ReviewTests"],
    "is_atomic": true,
    "depth": 2,
    "status": "pending"
  },
  "parent": { ... },
  "siblings": [ ... ],
  "path_from_root": ["node_1", "node_3", "node_5"]
}
```

## Usage Examples

### Example 1: Simple Linear Plan

```go
// Define available actions
readAction := goap.NewSimpleAction(
    "ReadFile",
    "Read input file",
    goap.WorldState{},
    goap.WorldState{"file_read": true},
    1.0,
    readFileFunc,
)

processAction := goap.NewSimpleAction(
    "ProcessData",
    "Process the data",
    goap.WorldState{"file_read": true},
    goap.WorldState{"data_processed": true},
    5.0,
    processDataFunc,
)

// Create planner
planner := goap.NewPlanner([]goap.Action{readAction, processAction})

// Define goal
goal := goap.NewGoal(
    "ProcessFile",
    "Read and process the file",
    goap.WorldState{"file_read": true, "data_processed": true},
    10.0,
)

// Find plan
current := goap.NewWorldState()
plan := planner.FindPlan(current, goal)

// Execute
for _, action := range plan.Actions {
    err := action.Execute(ctx, current)
    if err != nil {
        log.Fatalf("Action failed: %v", err)
    }
}
```

### Example 2: Hierarchical Planning with LLM

```go
// Set up LLM-based goal refiner
llmRefiner := goap.NewLLMGoalRefiner(llmServer, "my-job", agentID)

// Create hierarchical planner
actionPlanner := goap.NewPlanner(availableActions)
hierarchicalPlanner := goap.NewHierarchicalPlanner(
    actionPlanner,
    llmRefiner,
    10, // Max depth
)

// Define high-level goal
goal := goap.NewGoal(
    "BuildFeature",
    "Build complete feature with tests and docs",
    goap.WorldState{
        "feature_implemented": true,
        "tests_passing": true,
        "documentation_complete": true,
    },
    100.0,
)

// Plan hierarchically (LLM will decompose)
plan, err := hierarchicalPlanner.PlanHierarchical(ctx, currentState, goal)
if err != nil {
    log.Fatalf("Planning failed: %v", err)
}

// Persist to disk
graph := goap.BuildGraphFromPlan(plan, agentID)
persistence := goap.NewGraphPersistence(outputPath)
err = persistence.SaveGraph(graph, runID)

// Execute with minimal context
executor := goap.NewGraphExecutor(persistence, runID)
executor.RegisterActions(availableActions)
err = executor.Execute(ctx, currentState)
```

### Example 3: Custom Goal Refiner

```go
type CustomRefiner struct {
    // Your refinement logic
}

func (r *CustomRefiner) Refine(ctx context.Context, goal *goap.Goal, current goap.WorldState) ([]*goap.Goal, error) {
    // Implement custom decomposition logic
    // Could use LLM, rules, or hybrid approach

    if goal.Name() == "DeployApplication" {
        return []*goap.Goal{
            goap.NewGoal("RunTests", "Ensure tests pass", ...),
            goap.NewGoal("BuildArtifacts", "Build deployment artifacts", ...),
            goap.NewGoal("DeployToStaging", "Deploy to staging environment", ...),
            goap.NewGoal("VerifyDeployment", "Verify deployment succeeded", ...),
        }, nil
    }

    return nil, nil // No refinement (atomic)
}

func (r *CustomRefiner) IsAtomic(goal *goap.Goal, current goap.WorldState) bool {
    // Determine if goal needs refinement
    return len(goal.DesiredState()) == 1
}
```

## Integration with Agentic

The GOAP system integrates with the existing Agentic workflow:

```go
// In cmd/main.go or new command
import "upside-down-research.com/oss/agentic/internal/goap"
import "upside-down-research.com/oss/agentic/internal/goap/actions"

// Create action context
actionCtx := &actions.ActionContext{
    LLM:        llmServer,
    Run:        run,
    Jobname:    CLI.TicketPath,
    AgentID:    agentID,
    OutputPath: CLI.Output,
}

// Build available actions
builder := actions.NewActionBuilder(actionCtx, ticketPath, runID, planner, implement)
availableActions := builder.BuildAllActions()

// Create goal
goal := goap.NewGoal(
    "CompleteTicket",
    "Implement ticket requirements",
    goap.WorldState{
        "ticket_read": true,
        "plan_generated": true,
        "code_implemented": true,
        "tests_passing": true,
    },
    100.0,
)

// Plan and execute
// ... (see examples above)
```

## Best Practices

### 1. Define Actions with Appropriate Complexity

```go
// Good: Reflects actual complexity
goap.NewSimpleAction("GenerateCodeWithReview", "...", ..., ..., 15.0, ...)

// Bad: Understates complexity
goap.NewSimpleAction("GenerateCodeWithReview", "...", ..., ..., 1.0, ...)
```

### 2. Use Quality Gates for Code Generation

```go
// Good: Built-in quality checks
goap.NewCompositeAction(
    "GenerateCode",
    "Generate with review",
    ...,
    []goap.Action{generateAction, reviewAction, testAction},
)

// Bad: No quality gates
goap.NewSimpleAction("GenerateCode", "Just generate", ...)
```

### 3. Keep Atomic Actions Focused

```go
// Good: Single responsibility
"WriteTestsForFunction"
"ReviewCodeQuality"
"RunUnitTests"

// Bad: Too broad
"DoEverything"
```

### 4. Persist Early and Often

```go
// Generate plan
plan, _ := hierarchicalPlanner.PlanHierarchical(ctx, current, goal)

// Persist immediately
graph := goap.BuildGraphFromPlan(plan, agentID)
persistence.SaveGraph(graph, runID)

// Now execute (can resume if interrupted)
executor.Execute(ctx, current)
```

### 5. Monitor Execution Progress

```go
// Before execution
executor.Execute(ctx, current)

// Check status
status, _ := executor.GetGraphStatus()
fmt.Printf("Progress: %d/%d nodes completed\n",
    status.CompletedNodes, status.TotalNodes)

if status.HasFailures() {
    fmt.Printf("Failures: %d nodes failed\n", status.FailedNodes)
}
```

## Testing

The GOAP system has comprehensive test coverage (77%):

```bash
go test ./internal/goap/
go test -cover ./internal/goap/
```

See `*_test.go` files for examples of testing:
- WorldState operations
- Action execution
- Planning algorithms
- Hierarchical refinement
- Graph persistence
- Execution with minimal context

## Future Enhancements

Potential improvements:
- **Parallel execution**: Execute independent subgoals concurrently
- **Plan repair**: Dynamically adjust plan when actions fail
- **Cost estimation**: Use LLM to estimate action complexity
- **Learning**: Improve refinement strategies based on past executions
- **Visualization**: Generate visual representations of plan graphs
- **Streaming execution**: Stream execution progress in real-time

## Related Documentation

- [Main Agentic README](../../readme.md)
- [CLAUDE.md](../../CLAUDE.md) - Project overview
- [Action Definitions](actions/actions.go) - Concrete action implementations
