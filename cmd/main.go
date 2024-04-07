package main

import (
	"os"
	"upside-down-research.com/oss/agentic/internal/llm"
)

func main() {
	query := os.Args[1]
	s := llm.Server{
		Host: "https://localhost:65530",
	}
	q := llm.NewChatQuery(
		llm.Names{User: "user", Assistant: "assistant"},
		[]llm.Messages{
			{Role: "user", Content: query},
		})
	s.Completion(q)

}
