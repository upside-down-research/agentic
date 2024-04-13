package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/alecthomas/kong"
	"log"
	"os"
	"strings"
	"upside-down-research.com/oss/agentic/internal/llm"
)

var CLI struct {
	LLMType string `name:"llm" help:"LLM type to use." enum:"openai,ai00,claude" default:"openai"`
	// optional flag to set the output path: -o
	Output *string `name:"output" help:"Output path." type:"path"`
	// path to initial query
	Path string `arg:"" name:"path" help:"Path to read." type:"path"`
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

func AnswerMe(l llm.LLMServer, query string) (string, error) {
	messages := []llm.Messages{
		{
			Role:    "user",
			Content: query,
		},
	}
	q := llm.NewChatQuery(
		llm.Names{User: "user",
			Assistant: "assistant"},
		messages,
	)
	return l.Completion(q)
}

func main() {
	_ = kong.Parse(&CLI)

	var s llm.LLMServer
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

	bytes, err := os.ReadFile(CLI.Path)
	if err != nil {
		log.Fatal(err)
	}
	query := string(bytes)
	initialQuery := query

	var messages []llm.Messages
	q := llm.NewChatQuery(
		llm.Names{User: "user",
			Assistant: "assistant"},
		messages,
	)

	// instruct the AI to analyze the query and flesh it out to guarantee a better response.
	reviewQuery := `Please analyze the query below and address any gaps. Add whatever is required to make the answer explicit and complete:\n\n` + query
	results, err := AnswerMe(s, reviewQuery)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Final query:\n\n%s\n", results)

	query = results
restart:
	messages = []llm.Messages{
		{
			Role: "user", Content: query,
		},
	}
	q.Messages = messages
	answer, err := s.Completion(q)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Candidate answer:")
	fmt.Println(answer)
	r, _ := AnswerMe(s, fmt.Sprintf(`Does the output below meet the requirements below? 
Output: 

%s

Requirements:

%s

After reviewing the output in relationship to the requirements, please assess the output for compliance with the requirements. Is it complete and correct?

Please formulate the answer in this JSON template:

{
   "answer": "$YES_OR_NO",
   "reason": "$REASONING"
}

`, answer, query))

	type Response struct {
		Answer string `json:"answer"`
		Reason string `json:"reason"`
	}

	resp := Response{}
	err = json.Unmarshal([]byte(r), &resp)
	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Println("ANSWER: ", resp.Answer)
	if strings.ToLower(resp.Answer) == "no" {
		log.Println("Restarting, analysis says incorrect", resp.Reason)
		query = initialQuery + `
This was an attempt at an answer: ` + answer +
			"But, according to " + resp.Reason + ", it is incorrect. Please try again."

		goto restart
	} else {
		log.Println("Analysis says correct: ", resp.Reason)
		fmt.Println(answer)
	}

	if CLI.Output != nil {
		err := os.WriteFile(*CLI.Output, []byte(answer), 0644)
		if err != nil {
			log.Fatal(err)
		}
	}

}
