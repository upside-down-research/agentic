package main

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/google/uuid"
	"path"
	"sync"

	"github.com/charmbracelet/log"
	"os"
	"strings"

	"upside-down-research.com/oss/agentic/internal/llm"
)

//go:embed prompts/planner.prompt
var planner string

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

//go:embed prompts/plan-review.prompt
var planReview string

// The AcceptableResponse is what the reviewer call should parse to.
type AcceptableResponse struct {
	Answer string `json:"answer"`
	Reason string `json:"reason"`
}

//go:embed prompts/implement.prompt
var implement string

type CodeDefinition struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

func (cd *CodeDefinition) WriteFile(superiorPath string) error {
	dst := path.Join(superiorPath, cd.Filename)
	return os.WriteFile(dst, []byte(cd.Content), 0644)
}

type ImplementedPlan struct {
	Environment    string           `json:"environment"`
	CodingLanguage string           `json:"coding_language"`
	Code           []CodeDefinition `json:"code"`
}

var CLI struct {
	LLMType    string  `name:"llm" help:"LLM type to use." enum:"openai,ai00,claude" default:"openai"`
	Output     string  `name:"output" help:"Output directory for details." type:"path"`
	TicketPath string  `arg:"" name:"ticket" help:"TicketPath to read." type:"path"`
	Model      *string `name:"model" help:"Model to use; leave blank for agentic pick"`
}

func StringPrompt(label string) string {
	var s string
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprint(os.Stderr, label+" ")
		s, _ = r.ReadString('\n')
		if s != "" {
			break
		}
	}
	return strings.TrimSpace(s)
}

type RunRecord struct {
	ID     int      `json:"id"`
	Query  string   `json:"query"`
	Answer string   `json:"answer"`
	Takes  []string `json:"analysis"`
}

// A Run is a top level structure describing the history of the program invocation.
type Run struct {
	RunID string `json:"run_id"`
	// The OutputPath is the directory that holds the runoutput.
	OutputPath string `json:"output_path"`
	// RunRecords are stored of the LLM runs: the results.
	// The index into the run is the sequential number of the query.
	//
	RunRecords map[int]RunRecord `json:"run_records"`
	latestRun  int
	sync.Mutex
}

func NewRun(runID string, outputPath string) *Run {
	return &Run{
		RunID:      runID,
		OutputPath: outputPath,
		RunRecords: make(map[int]RunRecord),
		latestRun:  0,
	}
}

func (run *Run) AppendRecord(query string, answer string, takes []string) {
	id := run.latestRun
	log.Info("Appending record to run", "id", id, "number of takes", len(takes))
	run.RunRecords[id] = RunRecord{
		ID:     id,
		Query:  query,
		Answer: answer,
		Takes:  takes,
	}
	run.latestRun = run.latestRun + 1
}

// WriteData should be used as a defer after a Run is created.
func (run *Run) WriteData() {
	log.Info("Writing data to disk")
	/// Create a directory RunID under OutputPath, and write the RunRecords to a file
	//make directory
	err := os.MkdirAll(run.OutputPath+"/"+run.RunID, os.ModePerm)
	if err != nil {
		log.Error("Failed to create directory: ", err)
		return
	}
	for _, runRecord := range run.RunRecords {
		runDirectory := run.OutputPath + "/" + run.RunID + "/" + fmt.Sprintf("%d", runRecord.ID)
		err = os.MkdirAll(runDirectory, os.ModePerm)
		if err != nil {
			log.Error("Failed to write runRecord record: ", err)
			return
		}
		// open file and write to
		queryPath := runDirectory + "/query.txt"
		err := os.WriteFile(queryPath, []byte(runRecord.Query), os.ModePerm)
		if err != nil {
			log.Error("Failed to write query: ", err)
		}
		answerPath := runDirectory + "/answer.txt"
		err = os.WriteFile(answerPath, []byte(runRecord.Answer), os.ModePerm)
		if err != nil {
			log.Error("Failed to write answer: ", err)
		}
		analysisPath := runDirectory + "/analysis/"
		err = os.MkdirAll(analysisPath, os.ModePerm)
		if err != nil {
			log.Error("Failed to create analysis directory: ", err)
			continue
		}
		for idx, take := range runRecord.Takes {
			err = os.WriteFile(fmt.Sprintf("%s/%d", analysisPath, idx), []byte(take), os.ModePerm)
			if err != nil {
				log.Error("Failed to write analysis: ", err)
			}
		}
	}
}

func (run *Run) AnswerAndVerify(params *llm.AnswerMeParams, finalOutput any) (string, error) {
	answer := ""
	var err error
	for {
		answer, err = func() (string, error) {
			var takes = []string{}

			// we update this to correct it if need be.
			query := params.Query

			if err != nil {
				query += "\nThe last time this question was asked, the following error was encountered: " + err.Error() +
					"\nPlease try again, incorporating the fresh information. Remember to use JSON. {"
			}

			defer run.AppendRecord(query, answer, takes)

			answer, err = llm.AnswerMe(params)
			if err != nil {
				return "", err
			}
			// is it any good?
			resp := AcceptableResponse{}

			for {
				log.Info("Reviewing the answer given...")
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
				log.Info("Attempting to unmarshal JSON response...")
				resp = AcceptableResponse{}
				err = json.Unmarshal([]byte(r), &resp)
				if err != nil {
					log.Infof("Not an acceptable response: %v.", r)
					log.Errorf("failed to unmarshal json: %v", err)
					log.Info("Retrying analysis")
				} else {
					break
				}
			}
			log.Info("Result of analysis", "ANSWER", resp.Answer)
			if strings.ToLower(resp.Answer) == "no" {
				log.Info("Restarting, analysis says incorrect:", "reason", resp.Reason)
				query = query + `This was an attempt at an answer: ` + answer +
					"But, according to " + resp.Reason + ", it is incorrect. Please try again, incorporating the fresh information."
				return "", fmt.Errorf("answer incorrect")
			} else {
				log.Info("Analysis says correct: ", "reason", resp.Reason)
				//
				err = json.Unmarshal([]byte(answer), finalOutput)
				if err != nil {
					log.Error("Failed to unmarshal final output: ", "error", err, "body", answer)
					return "", err
				}
			}
			return answer, nil
		}()
		if err != nil {
			log.Error("Failed to answer and verify (retrying): ", "Error", err)
			continue
		} else {
			break
		}
	}
	return answer, nil
}

func main() {
	log.SetLevel(log.DebugLevel)
	_ = kong.Parse(&CLI)

	var s llm.Server
	if CLI.LLMType == "ai00" {
		s = llm.AI00Server{
			Host: "https://localhost:65530",
		}
	} else if CLI.LLMType == "openai" {
		key, found := os.LookupEnv("OPENAI_API_KEY")
		if !found {
			log.Fatal("OPENAI_API_KEY not found")
		}

		if CLI.Model == nil {
			//s = llm.NewOpenAI(key, "gpt-4-turbo")
			s = llm.NewOpenAI(key, "gpt-3.5-turbo")
		} else {
			s = llm.NewOpenAI(key, *CLI.Model)
		}
	} else if CLI.LLMType == "claude" {
		key, found := os.LookupEnv("CLAUDE_API_KEY")
		if !found {
			log.Fatal("CLAUDE_API_KEY not found")
		}
		s = llm.NewClaude(key, "claude-3-haiku-20240307") // opus is EXPENSIVE.
	} else {
		log.Fatal("Unknown LLM type")
	}

	bytes, err := os.ReadFile(CLI.TicketPath)
	if err != nil {
		log.Fatal(err)
	}
	ticket := string(bytes)

	u, err := uuid.NewUUID()
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Starting run, agent id: %v ", u)
	run := NewRun(u.String(), CLI.Output)
	defer run.WriteData()

	query := planner + "\n" + ticket

	fmt.Printf("Initial request:\n\n%s\n", query)
	fmt.Println("--------------------------------------------------------------------------")
	plans := PlanCollection{}
	_, err = run.AnswerAndVerify(
		&llm.AnswerMeParams{
			LLM:     s,
			Jobname: CLI.TicketPath,
			AgentId: u.String(),
			Query:   query},
		&plans)
	if err != nil {
		log.Error("Failed to build plan: ", err)
		return
	}

	// Given the plans above has passed the acceptance gate.
	// we implement the plan
	log.Info("Implementing the plan...", "planSteps", len(plans.Plans))
	for _, plan := range plans.Plans {
		log.Info("Plan element", "name", plan.Name)
	}
	for _, plan := range plans.Plans {
		log.Info("Implementing plan: ", "name", plan.Name)
		b, err := json.Marshal(plan)
		if err != nil {
			log.Error("Failed to marshal plan %v: ", err)
			continue
		}
		candidate := ImplementedPlan{}
		_, err = run.AnswerAndVerify(
			&llm.AnswerMeParams{
				LLM:     s,
				Jobname: CLI.TicketPath,
				AgentId: u.String(),
				Query:   implement + "\n" + string(b)},
			&candidate)
		if err != nil {
			log.Error("Failed to implement plan: ", err)
			continue
		}
		dir := path.Join(run.OutputPath, run.RunID)
		err = os.MkdirAll(dir, os.ModePerm)
		for _, code := range candidate.Code {
			err = code.WriteFile(dir)
			if err != nil {
				log.Error("Failed to write code: ", err)
				continue
			}
			log.Info("Code written to disk: ", "filename", code.Filename)
		}
	}
	err = os.WriteFile(path.Join(run.OutputPath, run.RunID, "plan.txt"), plans.PrettyPrint(), 0644)
	if err != nil {
		return
	}
	log.Info("See results in directory", "dir", path.Join(run.OutputPath, run.RunID))
}
