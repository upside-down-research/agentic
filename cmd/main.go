package main

import (
	"bufio"
	"fmt"
	"github.com/alecthomas/kong"
	"log"
	"os"
	"strings"
	"upside-down-research.com/oss/agentic/internal/llm"
)

var CLI struct {
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

func main() {
	_ = kong.Parse(&CLI)
	s := llm.AI00Server{
		Host: "https://localhost:65530",
	}

	bytes, err := os.ReadFile(CLI.Path)
	if err != nil {
		log.Fatal(err)
	}
	query := string(bytes)

	messages := []llm.Messages{
		{
			Role: "user", Content: query,
		},
	}
	q := llm.NewChatQuery(
		llm.Names{User: "user",
			Assistant: "assistant"},
		messages,
	)
	sb := strings.Builder{}
	for {
		answer := s.Completion(q)
		fmt.Println(answer)
		promptResult := StringPrompt("enter to continue, r to reject, e to exit")
		if promptResult == "e" {
			break
		} else if promptResult == "r" {
			// Just move back up and retry
			continue
		} else {
			sb.WriteString(answer)
			q.Messages = append(q.Messages, llm.Messages{
				Role:    "assistant",
				Content: answer,
			})
			q.Messages = append(q.Messages, llm.Messages{
				Role:    "user",
				Content: "Continue as per initial instructions. As a reminder, the initial instructions were: " + query + "., and the results to date are: " + sb.String() + ". Please use the initial instructions and complete the task",
			})
		}
	}

	if CLI.Output != nil {
		err := os.WriteFile(*CLI.Output, []byte(sb.String()), 644)
		if err != nil {
			log.Fatal(err)
		}
	}

}
