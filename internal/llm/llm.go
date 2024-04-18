package llm

import (
	"github.com/charmbracelet/log"
	"time"
	"upside-down-research.com/oss/agentic/internal/o11y"
)

type Query struct {
	Model            string     `json:"model,omitempty"`
	Messages         []Messages `json:"messages"`
	MaxTokens        int        `json:"max_tokens"`
	Temperature      int        `json:"temperature"`
	TopP             float64    `json:"top_p,omitempty"`
	PresencePenalty  float64    `json:"presence_penalty"`
	FrequencyPenalty float64    `json:"frequency_penalty"`
	PenaltyDecay     float64    `json:"penalty_decay"`
	Stop             []string   `json:"stop"`
	Stream           bool       `json:"stream"`
	Names            Names      `json:"names"`
}

func NewChatQuery(n Names, m []Messages) *Query {
	r := &Query{
		Messages:         m,
		MaxTokens:        1000,
		TopP:             0.5,
		Temperature:      1,
		PresencePenalty:  0.3,
		FrequencyPenalty: 0.3,
		PenaltyDecay:     0.9982686325973925,
		Stop:             []string{"↵User:", "User:", "\n\n"},
		Stream:           false,
		Names:            n,
	}
	return r
}

type Middleware = func(query *Query) (string, error)

func TimeWrapper(model string) func(query *Query, next Middleware) (string, error) {
	return func(query *Query, next Middleware) (string, error) {
		now := time.Now()
		s, err := next(query)
		defer func() {
			end := time.Now()
			o11y.WriteData("llm_duration", map[string]string{"model": query.Model}, float32(end.Sub(now).Milliseconds())/1000)
			log.Printf("%v Completion took %v", model, end.Sub(now))
		}()
		return s, err
	}
}

type Server interface {
	Completion(data *Query) (string, error)
	Model() string
}

type Messages struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type Names struct {
	User      string `json:"user"`
	Assistant string `json:"assistant"`
}

func AnswerMe(l Server, query string) (string, error) {
	messages := []Messages{
		{
			Role:    "user",
			Content: query,
		},
	}
	q := NewChatQuery(
		Names{User: "user",
			Assistant: "assistant"},
		messages,
	)
	return l.Completion(q)
}
