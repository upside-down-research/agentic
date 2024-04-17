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

//go:embed prompts/plan-review.prompt
var planReview string

//go:embed prompts/planner.prompt
var planner string

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

	query = initialQuery
	for {
		answer, err := llm.AnswerMe(s, fmt.Sprintf(planner, query))
		if err != nil {
			log.Fatal(err)
		}
		// is it any good?
		log.Info("Reviewing the answer given...")
		r, err := llm.AnswerMe(s, fmt.Sprintf(planReview, answer, query))
		if err != nil {
			log.Fatal(err)
		}

		type Response struct {
			Answer string `json:"answer"`
			Reason string `json:"reason"`
		}

		log.Info("Attempting to unmarshal JSON response...")
		resp := Response{}
		err = json.Unmarshal([]byte(r), &resp)
		if err != nil {
			log.Info("Response: ", resp)
			log.Fatalf("failed to unmarshal json: %v", err)
		}

		fmt.Println("ANSWER: ", resp.Answer)
		if strings.ToLower(resp.Answer) == "no" {
			log.Info("Restarting, analysis says incorrect", resp.Reason)
			query = initialQuery + `
This was an attempt at an answer: ` + answer +
				"But, according to " + resp.Reason + ", it is incorrect. Please try again."

			continue
		} else {
			log.Info("Analysis says correct: ", "reason", resp.Reason)
			log.Info("See file for output ", "file", CLI.Output)
			err := os.WriteFile(CLI.Output, []byte(answer), 0644)
			if err != nil {
				log.Fatal(err)
			}
			break
		}
	}
}
