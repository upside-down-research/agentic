package commands

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"upside-down-research.com/oss/agentic/internal/config"
	"upside-down-research.com/oss/agentic/internal/estimation"
	"upside-down-research.com/oss/agentic/internal/llm"
	"upside-down-research.com/oss/agentic/internal/progress"
	"upside-down-research.com/oss/agentic/internal/validation"
)

//go:embed prompts/planner.prompt
var planner string

//go:embed prompts/plan-review.prompt
var planReview string

//go:embed prompts/implement.prompt
var implement string

type InOut struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type PlanDefinition struct {
	Inputs   []InOut `json:"inputs"`
	Outputs  []InOut `json:"outputs"`
	Behavior string  `json:"behavior"`
}

type Plan struct {
	Name       string         `json:"name"`
	SystemType string         `json:"type"`
	Rationale  string         `json:"rationale"`
	Definition PlanDefinition `json:"definition"`
}

type PlanCollection struct {
	Plans []Plan `json:"plans"`
}

func (c PlanCollection) PrettyPrint() []byte {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		log.Error("Failed to marshal plan collection: ", err)
		return nil
	}
	return b
}

type AcceptableResponse struct {
	Answer string `json:"answer"`
	Reason string `json:"reason"`
}

type CodeDefinition struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

func (cd *CodeDefinition) WriteFile(superiorPath string) error {
	dst := path.Join(superiorPath, cd.Filename)
	// Create directory if needed
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, []byte(cd.Content), 0644)
}

type ImplementedPlan struct {
	Environment    string           `json:"environment"`
	CodingLanguage string           `json:"coding_language"`
	Code           []CodeDefinition `json:"code"`
}

// GenerateCommand generates code from a specification
type GenerateCommand struct {
	SpecFile string  `arg:"" name:"spec" help:"Specification file" type:"path"`
	Config   string  `name:"config" help:"Configuration file path" type:"path"`
	Output   string  `name:"output" help:"Output directory" type:"path" default:"./output"`
	Model    *string `name:"model" help:"Override model from config"`
	DryRun   bool    `name:"dry-run" help:"Validate and estimate without executing"`
	Resume   string  `name:"resume" help:"Resume from a previous run ID"`
	NoPrompt bool    `name:"yes" short:"y" help:"Skip confirmation prompts"`
}

// Run executes the generate command
func (cmd *GenerateCommand) Run() error {
	prog := progress.NewIndicator(true)

	// Load configuration
	cfg, err := config.LoadConfig(cmd.Config)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override output if specified
	if cmd.Output != "" {
		cfg.Output.Directory = cmd.Output
	}

	// Validate configuration
	prog.Phase("Validation")
	configResult := validation.ValidateConfig(cfg)
	if !configResult.IsValid() {
		validation.PrintValidationResult(configResult)
		return fmt.Errorf("configuration validation failed")
	}

	// Validate spec file
	specResult := validation.ValidateSpecFile(cmd.SpecFile)
	if !specResult.IsValid() {
		validation.PrintValidationResult(specResult)
		return fmt.Errorf("specification validation failed")
	}
	prog.Success("Configuration and spec file validated")

	// Read spec file
	specData, err := os.ReadFile(cmd.SpecFile)
	if err != nil {
		return fmt.Errorf("failed to read spec file: %w", err)
	}
	ticket := string(specData)

	// Determine model
	model := ""
	if cmd.Model != nil {
		model = *cmd.Model
	} else if cfg.LLM.Model != nil {
		model = *cfg.LLM.Model
	} else {
		model = getDefaultModel(cfg.LLM.Provider)
	}

	// Cost estimation
	prog.Phase("Cost Estimation")
	est := estimation.EstimateGeneration(model, ticket, 3)
	fmt.Println(estimation.FormatEstimate(est))

	// Check against limits
	if ok, reason := estimation.ShouldProceed(est, cfg.Cost.MaxCostUSD, cfg.Cost.MaxTokens); !ok {
		prog.Error("Cost limit exceeded", fmt.Errorf(reason))
		return fmt.Errorf("cost limit exceeded: %s", reason)
	}
	prog.Success("Estimate within limits")

	// Dry run check
	if cmd.DryRun {
		fmt.Println("\n✓ Dry run complete - all validations passed")
		fmt.Println("  Remove --dry-run to execute")
		return nil
	}

	// Confirmation prompt
	if !cmd.NoPrompt && cfg.Cost.WarnOnCost && est.CostUSD > 1.0 {
		fmt.Printf("\nProceed with estimated cost of $%.2f? [y/N] ", est.CostUSD)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted by user")
			return nil
		}
	}

	// Create LLM server
	server, err := createLLMServer(cfg, model)
	if err != nil {
		return fmt.Errorf("failed to create LLM server: %w", err)
	}

	// Create or resume run
	var runID string
	if cmd.Resume != "" {
		runID = cmd.Resume
		prog.Info(fmt.Sprintf("Resuming run: %s", runID))
	} else {
		u, _ := uuid.NewUUID()
		runID = u.String()
		prog.Info(fmt.Sprintf("Starting new run: %s", runID))
	}

	run := NewRun(runID, cfg.Output.Directory, cfg.Retry.MaxAttempts, prog)
	defer run.WriteData()

	// Planning phase
	prog.Phase("Planning Phase")
	query := planner + "\n" + ticket
	plans := PlanCollection{}

	_, err = run.AnswerAndVerify(
		&llm.AnswerMeParams{
			LLM:     server,
			Jobname: cmd.SpecFile,
			AgentId: runID,
			Query:   query,
		},
		&plans,
		cfg.Retry.MaxAttempts,
	)
	if err != nil {
		prog.Error("Planning failed", err)
		return fmt.Errorf("planning failed: %w", err)
	}
	prog.Success(fmt.Sprintf("Plan approved with %d components", len(plans.Plans)))

	// Implementation phase
	prog.Phase(fmt.Sprintf("Implementation Phase (%d components)", len(plans.Plans)))
	for i, plan := range plans.Plans {
		prog.Step(fmt.Sprintf("Component %d/%d: %s", i+1, len(plans.Plans), plan.Name))

		b, err := json.Marshal(plan)
		if err != nil {
			prog.Error(fmt.Sprintf("Failed to marshal plan %s", plan.Name), err)
			continue
		}

		candidate := ImplementedPlan{}
		_, err = run.AnswerAndVerify(
			&llm.AnswerMeParams{
				LLM:     server,
				Jobname: cmd.SpecFile,
				AgentId: runID,
				Query:   implement + "\n" + string(b),
			},
			&candidate,
			cfg.Retry.MaxAttempts,
		)
		if err != nil {
			prog.Error(fmt.Sprintf("Implementation of %s failed", plan.Name), err)
			continue
		}

		// Write generated code
		dir := path.Join(run.OutputPath, run.RunID)
		for _, code := range candidate.Code {
			if err := code.WriteFile(dir); err != nil {
				prog.Error(fmt.Sprintf("Failed to write %s", code.Filename), err)
				continue
			}
			prog.Info(fmt.Sprintf("✓ Written: %s", code.Filename))
		}
		prog.Success(fmt.Sprintf("Component %s implemented", plan.Name))
	}

	// Write plan file
	planPath := path.Join(run.OutputPath, run.RunID, "plan.txt")
	if err := os.WriteFile(planPath, plans.PrettyPrint(), 0644); err != nil {
		prog.Error("Failed to write plan file", err)
	}

	// Quality gates
	outputDir := path.Join(run.OutputPath, run.RunID)

	// Compilation check
	if cfg.QualityGate.RequireCompilation {
		prog.Phase("Quality Gates: Compilation")
		if err := compileCode(outputDir); err != nil {
			prog.Error("Compilation failed", err)
			return fmt.Errorf("compilation failed: %w", err)
		}
		prog.Success("Code compiles successfully")
	}

	// Test execution
	if cfg.QualityGate.RunTests {
		prog.Phase("Quality Gates: Tests")
		if err := runTests(outputDir); err != nil {
			prog.Error("Tests failed", err)
			return fmt.Errorf("tests failed: %w", err)
		}
		prog.Success("All tests passed")
	}

	prog.Summary(true, fmt.Sprintf("Output directory: %s", outputDir))
	return nil
}

type RunRecord struct {
	ID     int      `json:"id"`
	Query  string   `json:"query"`
	Answer string   `json:"answer"`
	Takes  []string `json:"analysis"`
}

func (runRecord *RunRecord) WriteFile(outputPath, runID string) {
	runDirectory := path.Join(outputPath, runID, fmt.Sprintf("%d", runRecord.ID))
	err := os.MkdirAll(runDirectory, os.ModePerm)
	if err != nil {
		log.Error("Failed to write run record: ", err)
		return
	}

	queryPath := runDirectory + "/query.txt"
	_ = os.WriteFile(queryPath, []byte(runRecord.Query), 0644)

	answerPath := runDirectory + "/answer.txt"
	_ = os.WriteFile(answerPath, []byte(runRecord.Answer), 0644)

	analysisPath := runDirectory + "/analysis/"
	_ = os.MkdirAll(analysisPath, os.ModePerm)

	for idx, take := range runRecord.Takes {
		_ = os.WriteFile(fmt.Sprintf("%s/%d", analysisPath, idx), []byte(take), 0644)
	}
}

type Run struct {
	RunID       string
	OutputPath  string
	RunRecords  map[int]RunRecord
	latestRun   int
	maxAttempts int
	progress    *progress.Indicator
	sync.Mutex
}

func NewRun(runID string, outputPath string, maxAttempts int, prog *progress.Indicator) *Run {
	return &Run{
		RunID:       runID,
		OutputPath:  outputPath,
		RunRecords:  make(map[int]RunRecord),
		latestRun:   0,
		maxAttempts: maxAttempts,
		progress:    prog,
	}
}

func (run *Run) AppendRecord(query string, answer string, takes []string) {
	id := run.latestRun
	run.RunRecords[id] = RunRecord{
		ID:     id,
		Query:  query,
		Answer: answer,
		Takes:  takes,
	}
	run.latestRun = run.latestRun + 1
	rr := run.RunRecords[id]
	rr.WriteFile(run.OutputPath, run.RunID)
}

func (run *Run) WriteData() {
	err := os.MkdirAll(run.OutputPath+"/"+run.RunID, os.ModePerm)
	if err != nil {
		log.Error("Failed to create directory: ", err)
		return
	}
	for _, runRecord := range run.RunRecords {
		runRecord.WriteFile(run.OutputPath, run.RunID)
	}
}

func (run *Run) AnswerAndVerify(params *llm.AnswerMeParams, finalOutput any, maxAttempts int) (string, error) {
	answer := ""
	var err error
	attempts := 0

	for attempts < maxAttempts {
		attempts++
		answer, err = func() (string, error) {
			var takes = []string{}
			query := params.Query

			if err != nil {
				query += "\nThe last time this question was asked, the following error was encountered: " + err.Error() +
					"\nPlease try again, incorporating the fresh information. Remember to use JSON. {"
			}

			defer run.AppendRecord(query, answer, takes)

			// Show LLM call in progress
			run.progress.SubStep(fmt.Sprintf("LLM call (attempt %d/%d)", attempts, maxAttempts))

			answer, err = llm.AnswerMe(params)
			if err != nil {
				return "", err
			}

			// Review loop (with its own limit)
			resp := AcceptableResponse{}
			reviewAttempts := 0
			maxReviewAttempts := 5

			for reviewAttempts < maxReviewAttempts {
				reviewAttempts++
				run.progress.Review(reviewAttempts)

				p := &llm.AnswerMeParams{
					LLM:     params.LLM,
					Jobname: params.Jobname,
					AgentId: params.AgentId,
					Query:   fmt.Sprintf(planReview, answer, query),
				}
				r, err := llm.AnswerMe(p)
				if err != nil {
					log.Errorf("Failed to review the answer: %v", err)
					continue
				}
				takes = append(takes, r)

				resp = AcceptableResponse{}
				err = json.Unmarshal([]byte(r), &resp)
				if err != nil {
					log.Infof("Review response not valid JSON: %v", r)
					continue
				} else {
					break
				}
			}

			if strings.ToLower(resp.Answer) == "no" {
				run.progress.Info(fmt.Sprintf("Review rejected: %s", resp.Reason))
				query = query + `This was an attempt at an answer: ` + answer +
					"But, according to " + resp.Reason + ", it is incorrect. Please try again, incorporating the fresh information."
				return "", fmt.Errorf("answer incorrect: %s", resp.Reason)
			} else {
				run.progress.Info("Review approved")
				err = json.Unmarshal([]byte(answer), finalOutput)
				if err != nil {
					log.Error("Failed to unmarshal final output: ", "error", err)
					return "", err
				}
			}
			return answer, nil
		}()

		if err != nil {
			if attempts >= maxAttempts {
				return "", fmt.Errorf("max attempts (%d) reached: %w", maxAttempts, err)
			}
			run.progress.Info(fmt.Sprintf("Retry %d/%d: %v", attempts, maxAttempts, err))
			continue
		} else {
			break
		}
	}
	return answer, nil
}

func createLLMServer(cfg *config.Config, model string) (llm.Server, error) {
	switch cfg.LLM.Provider {
	case "ai00":
		return llm.AI00Server{Host: "https://localhost:65530"}, nil

	case "openai":
		key := cfg.LLM.APIKey
		if key == "" {
			key = os.Getenv("OPENAI_API_KEY")
		}
		if key == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY not set")
		}
		return llm.NewOpenAI(key, model), nil

	case "claude":
		key := cfg.LLM.APIKey
		if key == "" {
			key = os.Getenv("CLAUDE_API_KEY")
		}
		if key == "" {
			return nil, fmt.Errorf("CLAUDE_API_KEY not set")
		}
		return llm.NewClaude(key, model), nil

	case "bedrock":
		region := cfg.LLM.AWSRegion
		if region == "" {
			region = "us-east-1"
		}
		return llm.NewBedrock(region, model)

	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", cfg.LLM.Provider)
	}
}

func compileCode(dir string) error {
	// Check if there are any .go files
	matches, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil || len(matches) == 0 {
		return fmt.Errorf("no Go files found in %s", dir)
	}

	// Try to compile
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compilation error:\n%s", string(output))
	}

	return nil
}

func runTests(dir string) error {
	// Check if there are any test files
	matches, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return fmt.Errorf("no test files found in %s", dir)
	}

	// Run tests
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("test failures:\n%s", string(output))
	}

	return nil
}
