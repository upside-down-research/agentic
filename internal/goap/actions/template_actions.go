package actions

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"upside-down-research.com/oss/agentic/internal/goap"
	"upside-down-research.com/oss/agentic/internal/goap/templates"
)

// TemplateBasedLLMAction uses templates to guide LLM generation.
// This enforces structure: LLM generates, GOFAI reasons.
type TemplateBasedLLMAction struct {
	*goap.BaseAction
	ctx          *ActionContext
	template     *templates.Template
	templateData interface{}
	resultKey    string
}

func NewTemplateBasedLLMAction(
	name string,
	ctx *ActionContext,
	template *templates.Template,
	templateData interface{},
	resultKey string,
	preconditions goap.WorldState,
) *TemplateBasedLLMAction {
	return &TemplateBasedLLMAction{
		BaseAction: goap.NewBaseAction(
			name,
			fmt.Sprintf("Generate using template: %s", template.Name()),
			preconditions,
			goap.WorldState{resultKey: true},
			10.0, // Template-guided LLM generation
		),
		ctx:          ctx,
		template:     template,
		templateData: templateData,
		resultKey:    resultKey,
	}
}

func (a *TemplateBasedLLMAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for TemplateBasedLLMAction")
	}

	log.Info("Generating with template", "template", a.template.Name(), "resultKey", a.resultKey)

	// Render the template to create structured prompt
	prompt, err := a.template.RenderWithExamples(a.templateData)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// In real implementation, would call LLM here with the structured prompt
	// The LLM fills in the template, GOFAI validates and uses the result
	log.Info("LLM would fill template here (simplified in implementation)")

	current.Set(a.resultKey, true)
	current.Set(a.resultKey+"_prompt", prompt)

	return nil
}

func (a *TemplateBasedLLMAction) Clone() goap.Action {
	return NewTemplateBasedLLMAction(
		a.Name(),
		a.ctx,
		a.template,
		a.templateData,
		a.resultKey,
		a.Preconditions().Clone(),
	)
}

// Language-Specific Template Generation Actions

// GenerateGoStructAction generates Go struct templates.
type GenerateGoStructAction struct {
	*goap.BaseAction
	ctx        *ActionContext
	structName string
	fields     []FieldSpec
}

type FieldSpec struct {
	Name string
	Type string
	Tags string
}

func NewGenerateGoStructAction(ctx *ActionContext, structName string, fields []FieldSpec) *GenerateGoStructAction {
	return &GenerateGoStructAction{
		BaseAction: goap.NewBaseAction(
			"GenerateGoStruct",
			fmt.Sprintf("Generate Go struct: %s", structName),
			goap.WorldState{"project_initialized": true},
			goap.WorldState{"go_struct_generated": true},
			3.0,
		),
		ctx:        ctx,
		structName: structName,
		fields:     fields,
	}
}

func (a *GenerateGoStructAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for GenerateGoStruct")
	}

	log.Info("Generating Go struct template", "name", a.structName)

	// Generate struct template
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("type %s struct {\n", a.structName))

	for _, field := range a.fields {
		sb.WriteString(fmt.Sprintf("\t%s %s", field.Name, field.Type))
		if field.Tags != "" {
			sb.WriteString(fmt.Sprintf(" `%s`", field.Tags))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("}\n")

	current.Set("go_struct_generated", true)
	current.Set("go_struct_code", sb.String())

	log.Info("Go struct template generated")
	return nil
}

func (a *GenerateGoStructAction) Clone() goap.Action {
	return NewGenerateGoStructAction(a.ctx, a.structName, a.fields)
}

// GeneratePythonClassAction generates Python class templates.
type GeneratePythonClassAction struct {
	*goap.BaseAction
	ctx       *ActionContext
	className string
	methods   []string
	baseClass string
}

func NewGeneratePythonClassAction(ctx *ActionContext, className string, methods []string, baseClass string) *GeneratePythonClassAction {
	return &GeneratePythonClassAction{
		BaseAction: goap.NewBaseAction(
			"GeneratePythonClass",
			fmt.Sprintf("Generate Python class: %s", className),
			goap.WorldState{"project_initialized": true},
			goap.WorldState{"python_class_generated": true},
			3.0,
		),
		ctx:       ctx,
		className: className,
		methods:   methods,
		baseClass: baseClass,
	}
}

func (a *GeneratePythonClassAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for GeneratePythonClass")
	}

	log.Info("Generating Python class template", "name", a.className)

	var sb strings.Builder

	// Class declaration
	if a.baseClass != "" {
		sb.WriteString(fmt.Sprintf("class %s(%s):\n", a.className, a.baseClass))
	} else {
		sb.WriteString(fmt.Sprintf("class %s:\n", a.className))
	}

	sb.WriteString(fmt.Sprintf("    \"\"\"TODO: Add docstring for %s\"\"\"\n\n", a.className))

	// Constructor
	sb.WriteString("    def __init__(self):\n")
	sb.WriteString("        \"\"\"Initialize the class\"\"\"\n")
	sb.WriteString("        pass\n\n")

	// Methods
	for _, method := range a.methods {
		sb.WriteString(fmt.Sprintf("    def %s(self):\n", method))
		sb.WriteString(fmt.Sprintf("        \"\"\"TODO: Implement %s\"\"\"\n", method))
		sb.WriteString("        pass\n\n")
	}

	current.Set("python_class_generated", true)
	current.Set("python_class_code", sb.String())

	log.Info("Python class template generated")
	return nil
}

func (a *GeneratePythonClassAction) Clone() goap.Action {
	return NewGeneratePythonClassAction(a.ctx, a.className, a.methods, a.baseClass)
}

// GenerateJavaScriptModuleAction generates JavaScript/TypeScript module templates.
type GenerateJavaScriptModuleAction struct {
	*goap.BaseAction
	ctx        *ActionContext
	moduleName string
	exports    []string
	isTypeScript bool
}

func NewGenerateJavaScriptModuleAction(ctx *ActionContext, moduleName string, exports []string, isTypeScript bool) *GenerateJavaScriptModuleAction {
	actionName := "GenerateJavaScriptModule"
	if isTypeScript {
		actionName = "GenerateTypeScriptModule"
	}

	return &GenerateJavaScriptModuleAction{
		BaseAction: goap.NewBaseAction(
			actionName,
			fmt.Sprintf("Generate %s module: %s", map[bool]string{true: "TypeScript", false: "JavaScript"}[isTypeScript], moduleName),
			goap.WorldState{"project_initialized": true},
			goap.WorldState{"js_module_generated": true},
			3.0,
		),
		ctx:        ctx,
		moduleName: moduleName,
		exports:    exports,
		isTypeScript: isTypeScript,
	}
}

func (a *GenerateJavaScriptModuleAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for GenerateJavaScriptModule")
	}

	lang := "JavaScript"
	if a.isTypeScript {
		lang = "TypeScript"
	}

	log.Info(fmt.Sprintf("Generating %s module template", lang), "name", a.moduleName)

	var sb strings.Builder

	// Module documentation
	sb.WriteString(fmt.Sprintf("/**\n * %s module\n * TODO: Add module description\n */\n\n", a.moduleName))

	// Exports
	for _, export := range a.exports {
		if a.isTypeScript {
			sb.WriteString(fmt.Sprintf("export function %s(): void {\n", export))
		} else {
			sb.WriteString(fmt.Sprintf("export function %s() {\n", export))
		}
		sb.WriteString(fmt.Sprintf("  // TODO: Implement %s\n", export))
		sb.WriteString("}\n\n")
	}

	current.Set("js_module_generated", true)
	current.Set("js_module_code", sb.String())

	log.Info(fmt.Sprintf("%s module template generated", lang))
	return nil
}

func (a *GenerateJavaScriptModuleAction) Clone() goap.Action {
	return NewGenerateJavaScriptModuleAction(a.ctx, a.moduleName, a.exports, a.isTypeScript)
}

// GenerateAPIEndpointAction generates REST API endpoint templates.
type GenerateAPIEndpointAction struct {
	*goap.BaseAction
	ctx      *ActionContext
	endpoint string
	method   string
	language string
}

func NewGenerateAPIEndpointAction(ctx *ActionContext, endpoint, method, language string) *GenerateAPIEndpointAction {
	return &GenerateAPIEndpointAction{
		BaseAction: goap.NewBaseAction(
			"GenerateAPIEndpoint",
			fmt.Sprintf("Generate %s %s endpoint in %s", method, endpoint, language),
			goap.WorldState{"project_initialized": true},
			goap.WorldState{"api_endpoint_generated": true},
			5.0, // API endpoints are more complex
		),
		ctx:      ctx,
		endpoint: endpoint,
		method:   method,
		language: language,
	}
}

func (a *GenerateAPIEndpointAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for GenerateAPIEndpoint")
	}

	log.Info("Generating API endpoint template",
		"endpoint", a.endpoint,
		"method", a.method,
		"language", a.language)

	var code string

	switch a.language {
	case "go":
		code = a.generateGoEndpoint()
	case "python":
		code = a.generatePythonEndpoint()
	case "javascript", "typescript":
		code = a.generateJavaScriptEndpoint()
	default:
		return fmt.Errorf("unsupported language: %s", a.language)
	}

	current.Set("api_endpoint_generated", true)
	current.Set("api_endpoint_code", code)

	log.Info("API endpoint template generated")
	return nil
}

func (a *GenerateAPIEndpointAction) generateGoEndpoint() string {
	return fmt.Sprintf(`func Handle%s(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement %s %s handler

	// Validate request
	if r.Method != "%s" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Process request
	// ...

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}
`, strings.Title(strings.ToLower(a.method)), a.method, a.endpoint, a.method)
}

func (a *GenerateAPIEndpointAction) generatePythonEndpoint() string {
	return fmt.Sprintf(`@app.route('%s', methods=['%s'])
def handle_%s():
    """
    Handle %s %s request
    TODO: Add endpoint documentation
    """
    try:
        # Validate request
        # ...

        # Process request
        # ...

        # Return response
        return jsonify({
            'status': 'success'
        }), 200

    except Exception as e:
        return jsonify({
            'status': 'error',
            'message': str(e)
        }), 500
`, a.endpoint, a.method, strings.ToLower(a.method), a.method, a.endpoint)
}

func (a *GenerateAPIEndpointAction) generateJavaScriptEndpoint() string {
	return fmt.Sprintf(`app.%s('%s', async (req, res) => {
  /**
   * Handle %s %s request
   * TODO: Add endpoint documentation
   */
  try {
    // Validate request
    // ...

    // Process request
    // ...

    // Send response
    res.json({
      status: 'success'
    });

  } catch (error) {
    res.status(500).json({
      status: 'error',
      message: error.message
    });
  }
});
`, strings.ToLower(a.method), a.endpoint, a.method, a.endpoint)
}

func (a *GenerateAPIEndpointAction) Clone() goap.Action {
	return NewGenerateAPIEndpointAction(a.ctx, a.endpoint, a.method, a.language)
}
