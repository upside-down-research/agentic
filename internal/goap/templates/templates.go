package templates

import (
	"bytes"
	"fmt"
	"text/template"
)

// Template represents a structured prompt template for LLM generation.
// Templates guide the LLM to produce consistent, parseable output.
// This embodies the philosophy: LLMs generate, GOFAI reasons.
type Template struct {
	name        string
	description string
	tmpl        *template.Template
	examples    []string
}

// NewTemplate creates a new template.
func NewTemplate(name, description, templateStr string) (*Template, error) {
	tmpl, err := template.New(name).Parse(templateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &Template{
		name:        name,
		description: description,
		tmpl:        tmpl,
		examples:    []string{},
	}, nil
}

// AddExample adds an example output for this template.
func (t *Template) AddExample(example string) {
	t.examples = append(t.examples, example)
}

// Render renders the template with the given data.
func (t *Template) Render(data interface{}) (string, error) {
	var buf bytes.Buffer
	err := t.tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}
	return buf.String(), nil
}

// RenderWithExamples renders the template with examples included.
func (t *Template) RenderWithExamples(data interface{}) (string, error) {
	prompt, err := t.Render(data)
	if err != nil {
		return "", err
	}

	if len(t.examples) > 0 {
		prompt += "\n\n📚 EXAMPLES:\n"
		for i, example := range t.examples {
			prompt += fmt.Sprintf("\nExample %d:\n```json\n%s\n```\n", i+1, example)
		}
	}

	return prompt, nil
}

// Name returns the template name.
func (t *Template) Name() string {
	return t.name
}

// Description returns the template description.
func (t *Template) Description() string {
	return t.description
}

// Pre-defined templates for common LLM generation tasks

// GoalDecompositionTemplate guides LLM to decompose goals.
var GoalDecompositionTemplate = mustTemplate(NewTemplate(
	"GoalDecomposition",
	"Decompose a high-level goal into concrete subgoals",
	`You are a goal decomposition specialist. Break down the following goal into concrete subgoals.

GOAL:
Name: {{.GoalName}}
Description: {{.GoalDescription}}
Current State: {{.CurrentState}}
Desired State: {{.DesiredState}}

INSTRUCTIONS:
1. Analyze the goal and determine what subgoals are needed
2. Order subgoals logically (earlier subgoals enable later ones)
3. Make each subgoal concrete and achievable
4. Ensure subgoals collectively achieve the parent goal
5. Aim for 2-5 subgoals (avoid over-decomposition)

IMPORTANT: Respond ONLY with valid JSON in this exact format:
{
  "rationale": "Brief explanation of decomposition strategy",
  "subgoals": [
    {
      "name": "SubgoalName",
      "description": "What this subgoal accomplishes",
      "desired_state": {
        "key1": "value1",
        "key2": "value2"
      }
    }
  ]
}

Your response (JSON only, no other text):`,
))

// CodeGenerationTemplate guides LLM to generate code.
var CodeGenerationTemplate = mustTemplate(NewTemplate(
	"CodeGeneration",
	"Generate code following a specification",
	`You are a code generation specialist. Generate code according to this specification.

SPECIFICATION:
{{.Specification}}

LANGUAGE: {{.Language}}
STYLE GUIDE: {{.StyleGuide}}

CONSTRAINTS:
- Write complete, working code (no stubs or TODOs)
- Include necessary imports
- Follow language best practices
- Add minimal, clear comments
- Make code testable

IMPORTANT: Respond ONLY with valid JSON in this exact format:
{
  "analysis": "Brief analysis of the specification",
  "files": [
    {
      "path": "path/to/file.ext",
      "content": "... complete file content ..."
    }
  ]
}

Your response (JSON only, no other text):`,
))

// TestGenerationTemplate guides LLM to generate tests.
var TestGenerationTemplate = mustTemplate(NewTemplate(
	"TestGeneration",
	"Generate comprehensive tests for code",
	`You are a test generation specialist. Generate comprehensive tests for this code.

CODE TO TEST:
{{.Code}}

LANGUAGE: {{.Language}}
TEST FRAMEWORK: {{.Framework}}

REQUIREMENTS:
- Test happy paths
- Test edge cases
- Test error conditions
- Aim for high coverage
- Use clear test names
- Include assertions

TARGET COVERAGE: {{.TargetCoverage}}%

IMPORTANT: Respond ONLY with valid JSON in this exact format:
{
  "analysis": "What aspects of the code need testing",
  "tests": [
    {
      "name": "TestName",
      "description": "What this test validates",
      "code": "... complete test code ..."
    }
  ],
  "estimated_coverage": 85.0
}

Your response (JSON only, no other text):`,
))

// CodeReviewTemplate guides LLM to review code.
var CodeReviewTemplate = mustTemplate(NewTemplate(
	"CodeReview",
	"Review code for quality and correctness",
	`You are a code review specialist. Review the following code.

CODE:
{{.Code}}

SPECIFICATION:
{{.Specification}}

REVIEW CRITERIA:
{{range .Criteria}}
- {{.}}
{{end}}

INSTRUCTIONS:
1. Check if code meets specification
2. Evaluate code quality
3. Identify potential issues
4. Suggest improvements if needed

IMPORTANT: Respond ONLY with valid JSON in this exact format:
{
  "approved": true,
  "quality_score": 8.5,
  "issues": [
    {
      "severity": "warning",
      "location": "file.go:42",
      "description": "Consider error handling here",
      "suggestion": "Add explicit error check"
    }
  ],
  "summary": "Overall assessment of the code"
}

Your response (JSON only, no other text):`,
))

// BugFixTemplate guides LLM to fix bugs.
var BugFixTemplate = mustTemplate(NewTemplate(
	"BugFix",
	"Analyze and fix bugs in code",
	`You are a debugging specialist. Fix the bug in this code.

BUGGY CODE:
{{.Code}}

ERROR MESSAGE:
{{.ErrorMessage}}

TEST FAILURE:
{{.TestFailure}}

INSTRUCTIONS:
1. Analyze the error
2. Identify root cause
3. Propose a fix
4. Explain the fix

IMPORTANT: Respond ONLY with valid JSON in this exact format:
{
  "analysis": "Root cause analysis",
  "fix": {
    "file": "path/to/file.ext",
    "original": "... code to replace ...",
    "fixed": "... fixed code ...",
    "explanation": "Why this fixes the bug"
  }
}

Your response (JSON only, no other text):`,
))

// RefactoringTemplate guides LLM to refactor code.
var RefactoringTemplate = mustTemplate(NewTemplate(
	"Refactoring",
	"Refactor code to improve quality",
	`You are a refactoring specialist. Refactor this code.

CODE:
{{.Code}}

GOALS:
{{range .Goals}}
- {{.}}
{{end}}

CONSTRAINTS:
- Preserve behavior
- Improve readability
- Reduce complexity
- Follow best practices

IMPORTANT: Respond ONLY with valid JSON in this exact format:
{
  "rationale": "Why refactoring is needed",
  "changes": [
    {
      "file": "path/to/file.ext",
      "description": "What this change does",
      "before": "... original code ...",
      "after": "... refactored code ..."
    }
  ],
  "improvements": ["Better naming", "Reduced complexity", "..."]
}

Your response (JSON only, no other text):`,
))

// DocumentationTemplate guides LLM to write docs.
var DocumentationTemplate = mustTemplate(NewTemplate(
	"Documentation",
	"Generate documentation for code",
	`You are a documentation specialist. Document this code.

CODE:
{{.Code}}

DOCUMENTATION TYPE: {{.DocType}}

REQUIREMENTS:
- Explain purpose clearly
- Document parameters and return values
- Include usage examples
- Note any edge cases
- Keep it concise

IMPORTANT: Respond ONLY with valid JSON in this exact format:
{
  "summary": "One-line summary",
  "description": "Detailed description",
  "parameters": [
    {"name": "param1", "type": "string", "description": "What it does"}
  ],
  "returns": "What the function returns",
  "examples": ["Example code showing usage"],
  "notes": ["Any important notes or caveats"]
}

Your response (JSON only, no other text):`,
))

// Helper function for template creation
func mustTemplate(t *Template, err error) *Template {
	if err != nil {
		panic(err)
	}
	return t
}

// TemplateRegistry manages available templates.
type TemplateRegistry struct {
	templates map[string]*Template
}

// NewTemplateRegistry creates a new template registry.
func NewTemplateRegistry() *TemplateRegistry {
	registry := &TemplateRegistry{
		templates: make(map[string]*Template),
	}

	// Register built-in templates
	registry.Register(GoalDecompositionTemplate)
	registry.Register(CodeGenerationTemplate)
	registry.Register(TestGenerationTemplate)
	registry.Register(CodeReviewTemplate)
	registry.Register(BugFixTemplate)
	registry.Register(RefactoringTemplate)
	registry.Register(DocumentationTemplate)

	return registry
}

// Register adds a template to the registry.
func (r *TemplateRegistry) Register(template *Template) {
	r.templates[template.Name()] = template
}

// Get retrieves a template by name.
func (r *TemplateRegistry) Get(name string) (*Template, error) {
	tmpl, exists := r.templates[name]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", name)
	}
	return tmpl, nil
}

// List returns all registered template names.
func (r *TemplateRegistry) List() []string {
	names := make([]string, 0, len(r.templates))
	for name := range r.templates {
		names = append(names, name)
	}
	return names
}
