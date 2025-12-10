# Agentic - AI Software Engineer

## Project Overview

Agentic is an experimental system for generating code according to specifications using Large Language Models (LLMs). It functions as an "AI Software Engineer" that can plan, implement, and review code based on natural language requirements.

**Current Status**: v0.1.0 prototype (development paused due to code quality limitations)

**License**: AGPL-3.0 (free to use, but derivatives must be shared)

## What It Does

Agentic follows a structured workflow:

1. **Plan**: Analyzes software requirements and decomposes them into logical units (functions, classes, modules)
2. **Review**: Self-reviews the plan for correctness before proceeding
3. **Implement**: Generates code for each planned component
4. **Verify**: Reviews generated code against specifications

**Current Limitations** (as noted in readme):
- Does NOT compile code or run tests automatically
- Does NOT handle diffs/patches (generates complete new functions)
- Code quality was insufficient for continued development at v0.1.0
- Next step would require per-module review and patch process

## Architecture

### Core Components

#### 1. Main Orchestrator (`cmd/main.go`)
- Entry point and workflow controller
- Implements the plan-review-implement cycle
- Manages run history and output persistence
- Uses retry logic with LLM self-review

**Key Types**:
```go
// Represents decomposed software specifications
type Plan struct {
    Name       string         `json:"name"`
    SystemType string         `json:"type"`
    Rationale  string         `json:"rationale"`
    Definition PlanDefinition `json:"definition"`
}

// Generated code output
type ImplementedPlan struct {
    Environment    string           `json:"environment"`
    CodingLanguage string           `json:"coding_language"`
    Code           []CodeDefinition `json:"code"`
}
```

#### 2. LLM Abstraction Layer (`internal/llm/`)

**Interface**:
```go
type Server interface {
    Completion(data *Query) (string, error)
    Model() string
}
```

**Implementations**:
- **OpenAI** (`openai.go`): GPT-3.5-turbo, GPT-4-turbo support
  - Uses JSON response format mode
  - 120s timeout
  - Recommended as "best and cheapest" option

- **Claude** (`claude.go`): Claude 3 models (Haiku, Opus)
  - Custom JSON enforcement via system prompt
  - 120s timeout
  - Uses Anthropic Messages API (v2023-06-01)

- **AI00** (`ai00.go`): Local RWKV model server
  - Supports localhost deployment
  - Skips TLS verification for local use
  - OpenAI-compatible API

**Common Features**:
- Middleware pattern for timing/observability
- Standardized query structure
- Error handling and retry logic

#### 3. Prompting System (`cmd/prompts/`)

Three core prompt templates:

1. **`planner.prompt`**: Decomposes requirements into structured JSON plans
   - Specifies inputs, outputs, behavior for each component
   - Enforces Go struct compatibility

2. **`plan-review.prompt`**: Simple yes/no validation template
   - Compares output against requirements
   - Returns JSON with answer and reasoning

3. **`implement.prompt`**: Code generation with strict constraints
   - Go-only implementation
   - Complete implementations required (no stubs)
   - Structured JSON output format

#### 4. Observability (`internal/o11y/`)

Integrated monitoring via:
- **Prometheus**: Metrics push gateway integration
  - LLM call counters (by model, agent ID, job name)
  - Duration gauges for performance tracking

- **InfluxDB**: Time-series data recording
  - Hardcoded token/org (development setup)
  - Records detailed metrics with tags

#### 5. Answer and Verify Loop

The `Run.AnswerAndVerify()` method implements a retry-until-correct pattern:

```go
for {
    answer = LLM.Query(prompt)
    review = LLM.Query(reviewPrompt + answer)

    if review.Answer == "yes" {
        return answer
    }

    // Augment prompt with failure reason and retry
    prompt = prompt + "Previous attempt failed: " + review.Reason
}
```

This creates a self-correcting loop where the LLM reviews its own output.

## Project Structure

```
agentic/
├── cmd/
│   ├── main.go              # Main orchestrator
│   ├── main_test.go         # Tests (mostly stubs)
│   └── prompts/             # LLM prompt templates
│       ├── planner.prompt
│       ├── plan-review.prompt
│       └── implement.prompt
├── internal/
│   ├── llm/                 # LLM provider implementations
│   │   ├── llm.go          # Common interface and types
│   │   ├── openai.go       # OpenAI integration
│   │   ├── claude.go       # Anthropic Claude integration
│   │   ├── ai00.go         # Local RWKV integration
│   │   └── vertexai.go     # (Stub, not implemented)
│   ├── o11y/               # Observability (Prometheus, InfluxDB)
│   │   └── lib.go
│   └── utils.go            # Utilities
├── examples/               # Sample input tickets
│   ├── andon.in           # Web app specification
│   ├── r-tree.in          # Data structure request
│   └── rpm-host.in        # System utility request
├── .github/workflows/     # CI/CD
│   └── go.yml            # Build and test automation
├── grafana/              # Monitoring dashboards
├── go.mod                # Dependencies
├── makefile              # Build automation
├── docker-compose.yaml   # Monitoring stack
└── readme.md            # User documentation
```

## Usage

### Prerequisites

Set up API keys for your chosen LLM provider:

```bash
# For OpenAI (recommended)
export OPENAI_API_KEY="your-key-here"

# For Claude
export CLAUDE_API_KEY="your-key-here"

# For AI00 (local RWKV)
# No key needed, but requires setup (see readme.md)
```

### Building

```bash
make
# Produces: output/agentic
```

### Running

```bash
./output/agentic --llm=openai --model=gpt-4-turbo examples/andon.in --output planning
```

**CLI Options**:
- `--llm`: Provider selection (`openai`, `claude`, `ai00`)
- `--model`: Specific model (optional, has sensible defaults)
- `--output`: Directory for run artifacts
- Positional arg: Input ticket/specification file

### Output Structure

```
<output-dir>/<run-uuid>/
├── plan.txt                 # Final approved plan
├── 0/                      # First LLM interaction
│   ├── query.txt
│   ├── answer.txt
│   └── analysis/
│       ├── 0               # Review attempts
│       └── 1
├── 1/                      # Second interaction
│   └── ...
└── <generated-files>       # Actual code output
```

## Code Review & Quality Assessment

### Strengths

1. **Clean Architecture**
   - Well-defined separation of concerns
   - Interface-based LLM abstraction enables easy provider swapping
   - Middleware pattern for cross-cutting concerns

2. **Robust Retry Logic**
   - Self-reviewing mechanism catches LLM errors
   - Augmented prompts on retry provide learning feedback

3. **Good Observability**
   - Comprehensive metrics tracking
   - Full history preservation for debugging

4. **Structured Prompting**
   - JSON-enforced responses enable reliable parsing
   - Clear specifications with Go struct examples

### Issues & Recommendations

#### Critical Issues

1. **Hardcoded Secrets** (`internal/o11y/lib.go:121`)
   - InfluxDB token is committed to source
   - **Fix**: Use environment variables or config files

2. **Incomplete Error Handling**
   - Several errors are logged but execution continues
   - Example: `openai.go:118` - empty response returns `nil` instead of error
   - **Fix**: Return errors consistently, use proper error wrapping

3. **Deprecated API Usage**
   - `ioutil.ReadAll()` is deprecated (Go 1.16+)
   - **Fix**: Use `io.ReadAll()` instead

4. **TLS Security** (`ai00.go:98`)
   - `InsecureSkipVerify: true` is dangerous if not localhost
   - **Fix**: Add hostname verification or document localhost-only usage

#### Design Issues

1. **Infinite Retry Loop**
   - `AnswerAndVerify()` has no max retry limit
   - Could run indefinitely on persistent failures
   - **Fix**: Add configurable max attempts with exponential backoff

2. **Global State** (`o11y/lib.go`)
   - Global `pusher` and `mm` variables
   - Makes testing difficult
   - **Fix**: Use dependency injection

3. **Race Condition**
   - `Run.latestRun` is incremented without atomic operation
   - Though mutex protects `RunRecords`, `latestRun` access in `AppendRecord` is not fully protected
   - **Fix**: Use `atomic.AddInt32()` or protect with existing mutex

4. **Test Coverage**
   - Most tests are empty stubs (marked `// TODO: Add test cases`)
   - Only 2 actual test cases exist
   - **Fix**: Implement comprehensive test suite, especially for LLM mocking

#### Code Quality Issues

1. **Inconsistent Logging**
   - Mix of `log.Error()`, `log.Fatal()`, `log.Info()`
   - Some errors use `fmt.Println()`
   - **Fix**: Standardize on structured logging library (already using charmbracelet/log)

2. **Magic Numbers**
   - Hardcoded values: timeouts (120s), max tokens (1000, 4096), temperatures
   - **Fix**: Extract to constants or config

3. **Commented Code**
   - Several commented-out alternatives (e.g., `main.go:284-285`, `main.go:295-296`)
   - **Fix**: Remove or explain why preserved

4. **Missing Documentation**
   - No package-level documentation
   - Exported types lack doc comments (Go convention)
   - **Fix**: Add godoc comments

#### Architectural Limitations (Noted by Authors)

1. **No Compilation Step**
   - Generated code is not validated
   - Syntax errors would only be found manually

2. **No Test Execution**
   - Cannot verify generated code actually works
   - Critical for production readiness

3. **No Diff/Patch Support**
   - Always generates complete new code
   - Cannot incrementally modify existing codebases
   - Authors correctly identified this as blocker for v0.2.0

## Development Workflow

### Adding a New LLM Provider

1. Create `internal/llm/<provider>.go`
2. Implement the `Server` interface:
   ```go
   type YourLLM struct {
       Key    string
       _model string
   }

   func (llm YourLLM) Model() string { return llm._model }

   func (llm YourLLM) Completion(data *Query) (string, error) {
       TimedCompletion := TimeWrapper(llm.Model())
       return TimedCompletion(data, llm._completion)
   }

   func (llm YourLLM) _completion(data *Query) (string, error) {
       // Your implementation here
   }
   ```
3. Add CLI option in `cmd/main.go`
4. Handle JSON response formatting

### Modifying Prompts

Edit files in `cmd/prompts/`:
- Use `%s` for string interpolation points
- Ensure JSON format is clearly specified
- Provide Go struct examples for complex outputs
- Test with multiple LLM providers (behavior varies)

### Running Tests

```bash
make && go test -v ./...
```

Note: Most tests are stubs and will be skipped.

## Monitoring Setup

Included docker-compose stack provides:
- Prometheus (port 9090)
- Pushgateway (port 9091)
- Grafana (configured datasources)
- InfluxDB support

```bash
docker-compose up -d
```

## Dependencies

Key libraries:
- `github.com/alecthomas/kong`: CLI argument parsing
- `github.com/charmbracelet/log`: Structured logging
- `github.com/prometheus/client_golang`: Metrics
- `github.com/influxdata/influxdb-client-go/v2`: Time-series DB
- `github.com/google/uuid`: Run ID generation

## Future Directions (Author Notes)

From readme.md, the next phase would require:
1. **Diff/Patch Integration**: Apply incremental changes instead of full rewrites
2. **Compilation Step**: Validate syntax before accepting code
3. **Test Execution**: Run tests and retry on failures
4. **Specification Verification**: Ensure code meets requirements programmatically

The project was paused at v0.1.0 because generated code quality was insufficient without these features.

## Contributing

Per AGPL-3.0 license:
- Contributions welcome
- Derivative works must also be open-sourced
- Service usage requires sharing source code

## For AI Assistants (Claude, etc.)

When working with this codebase:

1. **Don't modify prompt files** without deep consideration - they're carefully tuned for JSON output
2. **Test with actual LLMs** - behavior varies significantly between providers
3. **Preserve run history** - the output structure is designed for debugging LLM behavior
4. **Consider token costs** - the retry loop can accumulate significant API usage
5. **Remember AGPL implications** - document any significant changes
6. **Fix security issues first** - especially the hardcoded InfluxDB token
7. **Focus on the core problem** - making LLMs reliably generate production-quality code with diffs/patches, not full rewrites

This is a research prototype exploring LLM code generation patterns. The architecture is sound, but the fundamental challenge (LLM code quality and incremental updates) remains unsolved.
