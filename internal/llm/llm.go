package llm

type LLMQuery struct {
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

func NewChatQuery(n Names, m []Messages) *LLMQuery {
	r := &LLMQuery{
		Messages:         m,
		MaxTokens:        1000,
		TopP:             0.5,
		Temperature:      1,
		PresencePenalty:  0.3,
		FrequencyPenalty: 0.3,
		PenaltyDecay:     0.9982686325973925,
		Stop:             []string{"↵User:", "User:", "\n\n"},
		Stream:           true,
		Names:            n,
	}
	return r
}

type LLMServer interface {
	Completion(data *LLMQuery) (string, error)
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

type VertexAI struct {
	// no idea
}
