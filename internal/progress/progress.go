package progress

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Indicator provides progress tracking for long operations
type Indicator struct {
	enabled bool
	mu      sync.Mutex
	phase   string
	step    string
	start   time.Time
}

// NewIndicator creates a new progress indicator
func NewIndicator(enabled bool) *Indicator {
	return &Indicator{
		enabled: enabled,
		start:   time.Now(),
	}
}

// Phase sets the current phase
func (p *Indicator) Phase(name string) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.phase = name
	fmt.Printf("\n📋 %s\n", name)
}

// Step sets the current step within a phase
func (p *Indicator) Step(name string) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.step = name
	fmt.Printf("  ├─ %s\n", name)
}

// SubStep shows a sub-step
func (p *Indicator) SubStep(name string) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Printf("  │  ├─ %s\n", name)
}

// Success marks a step as successful
func (p *Indicator) Success(name string) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Printf("  └─ ✓ %s\n", name)
}

// Error shows an error
func (p *Indicator) Error(name string, err error) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Printf("  └─ ✗ %s: %v\n", name, err)
}

// Info shows informational message
func (p *Indicator) Info(msg string) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Printf("  │  %s\n", msg)
}

// LLMCall shows LLM call information
func (p *Indicator) LLMCall(model string, attempt, maxAttempts int, promptTokens int) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Printf("  │  ├─ %s (attempt %d/%d, %s tokens)\n",
		model, attempt, maxAttempts, formatNumber(promptTokens))
}

// LLMResponse shows LLM response information
func (p *Indicator) LLMResponse(responseTokens int, costUSD float64) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Printf("  │  └─ Response: %s tokens ($%.4f)\n",
		formatNumber(responseTokens), costUSD)
}

// Review shows review attempt
func (p *Indicator) Review(attempt int) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Printf("  ├─ Reviewing (attempt %d)...\n", attempt)
}

// Elapsed returns time since start
func (p *Indicator) Elapsed() time.Duration {
	return time.Since(p.start)
}

// Summary prints final summary
func (p *Indicator) Summary(success bool, details string) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	symbol := "✓"
	if !success {
		symbol = "✗"
	}

	elapsed := time.Since(p.start)
	fmt.Printf("\n%s Complete in %s\n", symbol, formatDuration(elapsed))
	if details != "" {
		fmt.Printf("  %s\n", details)
	}
}

func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	// Add commas
	var parts []string
	for i := len(s); i > 0; i -= 3 {
		start := i - 3
		if start < 0 {
			start = 0
		}
		parts = append([]string{s[start:i]}, parts...)
	}
	return strings.Join(parts, ",")
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}
