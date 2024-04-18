package main

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/alecthomas/kong"

	"github.com/charmbracelet/log"
	"os"
	"strings"

	"upside-down-research.com/oss/agentic/internal/llm"
)


//go:embed prompts/planner.prompt
var planner string

// Planner prompt should yield a json that looks like this.
type Plan struct {
	Name       string `json:"name"`
	SystemType string `json:"type"`
	Rationale  string `json:"rationale"`
	Definition struct {
		Inputs []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"inputs"`
		Outputs []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"outputs"`
		Behavior string `json:"behavior"`
	} `json:"definition"`
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

var CLI struct {
	LLMType    string `name:"llm" help:"LLM type to use." enum:"openai,ai00,claude" default:"openai"`
	Output     string `name:"output" help:"Output path." type:"path"`
	TicketPath string `arg:"" name:"ticket" help:"TicketPath to read." type:"path"`
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

func AnswerAndVerify(s llm.Server, query string) (string, error) {
	for {
		answer, err := llm.AnswerMe(s, query)
		if err != nil {
			return "", err
		}
		// is it any good?
		var resp AcceptableResponse
		for {
			log.Info("Reviewing the answer given...")
			r, err := llm.AnswerMe(s, fmt.Sprintf(planReview, answer, query))
			if err != nil {
				return "", err
			}

			log.Info("Attempting to unmarshal JSON response...")
			resp = AcceptableResponse{}
			err = json.Unmarshal([]byte(r), &resp)
			if err != nil {
				log.Info("Not an acceptable response: ", resp)
				log.Errorf("failed to unmarshal json: %v", err)
				log.Info("Retrying analysis")
			} else {
				break
			}
		}
		fmt.Println("ANSWER: ", resp.Answer)
		if strings.ToLower(resp.Answer) == "no" {
			log.Info("Restarting, analysis says incorrect", resp.Reason)
			query = query + `
This was an attempt at an answer: ` + answer +
				"But, according to " + resp.Reason + ", it is incorrect. Please try again, incorporating the fresh information."
			continue
		} else {
			log.Info("Analysis says correct: ", "reason", resp.Reason)
			return answer, nil
		}
	}
}

func main() {
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
		s = llm.NewOpenAI(key, "gpt-4-turbo")
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
	query := string(bytes)
	initialQuery := query

	fmt.Printf("Initial request:\n\n%s\n", initialQuery)
	fmt.Println("--------------------------------------------------------------------------")

	planString, err := AnswerAndVerify(s, query)

	plans := []Plan{}
	err = json.Unmarshal([]byte(planString), &plans)
	if err != nil {
		log.Fatal(err)
	}

	// Given the plans above has passed the acceptance gate.
	// we implement the plan
	log.Info("Implementing the plan...")
	for _, plan := range plans {
		log.Info("Implementing plan: ", plan.Name)
		AnswerAndVerify(s, fmt.Sprintf(planner, plan.Name))

		candidatePlan, err := llm.AnswerMe(s, fmt.Sprintf(implement, plan.Name))
		if err != nil {
			log.Fatal(err)
		}
		log.Info("Candidate plan: ", candidatePlan)
		llm
		// check if the implementation is correct
		// if not, restart
		// if correct, continue
	}

}
