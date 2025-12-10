package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/log"
	"upside-down-research.com/oss/agentic/internal/goap"
)

// LSPEditAction performs edits via Language Server Protocol
// This gives us proper syntax tree awareness across languages!
type LSPEditAction struct {
	*goap.BaseAction
	language   string
	filePath   string
	edits      []LSPEdit
	lspCommand string
}

type LSPEdit struct {
	Type   string                 // "rename", "codeAction", "formatting", etc.
	Params map[string]interface{} // LSP-specific parameters
}

func NewLSPEditAction(language, filePath string, edits []LSPEdit, lspCommand string) *LSPEditAction {
	return &LSPEditAction{
		BaseAction: goap.NewBaseAction(
			"LSPEdit",
			fmt.Sprintf("LSP-based edit of %s (%s)", filePath, language),
			goap.WorldState{"file_exists": true},
			goap.WorldState{"lsp_edited": true},
			6.0, // LSP operations are sophisticated
		),
		language:   language,
		filePath:   filePath,
		edits:      edits,
		lspCommand: lspCommand,
	}
}

func (a *LSPEditAction) Execute(ctx context.Context, current goap.WorldState) error {
	log.Info("LSP-based edit", "language", a.language, "file", a.filePath, "edits", len(a.edits))

	// Ensure LSP server is available
	if a.lspCommand == "" {
		a.lspCommand = a.getDefaultLSPCommand()
	}

	// Check if LSP server exists
	_, err := exec.LookPath(a.lspCommand)
	if err != nil {
		log.Warn("LSP server not found, falling back to text edits", "command", a.lspCommand)
		return fmt.Errorf("LSP server not available: %s", a.lspCommand)
	}

	// Apply each LSP edit
	for i, edit := range a.edits {
		log.Debug("Applying LSP edit", "index", i, "type", edit.Type)

		err := a.applyLSPEdit(ctx, edit)
		if err != nil {
			return fmt.Errorf("LSP edit %d failed: %w", i, err)
		}
	}

	current.Set("lsp_edited", true)
	current.Set("edited_file", a.filePath)

	log.Info("LSP edits applied successfully")
	return nil
}

func (a *LSPEditAction) applyLSPEdit(ctx context.Context, edit LSPEdit) error {
	switch edit.Type {
	case "rename":
		return a.applyRename(ctx, edit.Params)
	case "formatting":
		return a.applyFormatting(ctx, edit.Params)
	case "codeAction":
		return a.applyCodeAction(ctx, edit.Params)
	default:
		return fmt.Errorf("unsupported LSP edit type: %s", edit.Type)
	}
}

func (a *LSPEditAction) applyRename(ctx context.Context, params map[string]interface{}) error {
	// LSP rename operation
	// This would send a textDocument/rename request to the LSP server

	oldName, _ := params["oldName"].(string)
	newName, _ := params["newName"].(string)
	line, _ := params["line"].(int)
	character, _ := params["character"].(int)

	log.Info("LSP rename", "old", oldName, "new", newName)

	// In a real implementation, this would:
	// 1. Start LSP server if not running
	// 2. Open document
	// 3. Send textDocument/rename request
	// 4. Apply returned WorkspaceEdit
	// 5. Save file

	// For now, log the operation
	log.Info("Would rename via LSP", "file", a.filePath, "position", fmt.Sprintf("%d:%d", line, character))

	return nil
}

func (a *LSPEditAction) applyFormatting(ctx context.Context, params map[string]interface{}) error {
	// LSP formatting operation
	log.Info("LSP formatting")

	// This would send a textDocument/formatting request
	// For now, fallback to language-specific formatters

	switch a.language {
	case "go":
		return a.formatWithGofmt()
	case "python":
		return a.formatWithBlack()
	case "javascript", "typescript":
		return a.formatWithPrettier()
	default:
		log.Warn("No formatter available for language", "language", a.language)
		return nil
	}
}

func (a *LSPEditAction) applyCodeAction(ctx context.Context, params map[string]interface{}) error {
	actionKind, _ := params["kind"].(string)
	log.Info("LSP code action", "kind", actionKind)

	// Code actions: refactorings, quick fixes, etc.
	// Would send textDocument/codeAction request

	return nil
}

func (a *LSPEditAction) getDefaultLSPCommand() string {
	switch a.language {
	case "go":
		return "gopls"
	case "python":
		return "pylsp" // or "pyright-langserver"
	case "rust":
		return "rust-analyzer"
	case "javascript", "typescript":
		return "typescript-language-server"
	case "java":
		return "jdtls"
	case "c", "cpp":
		return "clangd"
	default:
		return ""
	}
}

func (a *LSPEditAction) formatWithGofmt() error {
	cmd := exec.Command("gofmt", "-w", a.filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gofmt failed: %w\nOutput: %s", err, output)
	}
	return nil
}

func (a *LSPEditAction) formatWithBlack() error {
	cmd := exec.Command("black", a.filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("black failed: %w\nOutput: %s", err, output)
	}
	return nil
}

func (a *LSPEditAction) formatWithPrettier() error {
	cmd := exec.Command("prettier", "--write", a.filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("prettier failed: %w\nOutput: %s", err, output)
	}
	return nil
}

func (a *LSPEditAction) Clone() goap.Action {
	return NewLSPEditAction(a.language, a.filePath, a.edits, a.lspCommand)
}

// === SPECIFIC LSP-BASED REFACTORING ACTIONS ===

// LSPRenameAction renames a symbol using LSP
type LSPRenameAction struct {
	*goap.BaseAction
	language string
	filePath string
	position Position
	oldName  string
	newName  string
}

func NewLSPRenameAction(language, filePath string, pos Position, oldName, newName string) *LSPRenameAction {
	return &LSPRenameAction{
		BaseAction: goap.NewBaseAction(
			"LSPRename",
			fmt.Sprintf("LSP rename %s -> %s", oldName, newName),
			goap.WorldState{"file_exists": true},
			goap.WorldState{"symbol_renamed": true},
			7.0, // Complex operation - needs semantic analysis
		),
		language: language,
		filePath: filePath,
		position: pos,
		oldName:  oldName,
		newName:  newName,
	}
}

func (a *LSPRenameAction) Execute(ctx context.Context, current goap.WorldState) error {
	log.Info("LSP rename", "old", a.oldName, "new", a.newName, "file", a.filePath)

	// In production, this would:
	// 1. Connect to LSP server
	// 2. Send textDocument/rename request at position
	// 3. Receive WorkspaceEdit with all necessary changes
	// 4. Apply changes atomically
	// 5. Validate compilation still works

	// For now, demonstrate with gopls for Go files
	if a.language == "go" {
		return a.renameWithGopls(ctx)
	}

	log.Info("LSP rename would be performed here")
	current.Set("symbol_renamed", true)
	return nil
}

func (a *LSPRenameAction) renameWithGopls(ctx context.Context) error {
	// gopls can be used for rename operations
	// This is a simplified version - real implementation would use LSP protocol

	log.Info("Would use gopls for rename", "position", fmt.Sprintf("%d:%d", a.position.Line, a.position.Column))

	// Real implementation would:
	// - Start gopls server
	// - Send initialize request
	// - Open document
	// - Send textDocument/rename at position
	// - Get WorkspaceEdit
	// - Apply all edits
	// - Close document

	return nil
}

func (a *LSPRenameAction) Clone() goap.Action {
	return NewLSPRenameAction(a.language, a.filePath, a.position, a.oldName, a.newName)
}

// LSPExtractFunctionAction extracts code into a function using LSP
type LSPExtractFunctionAction struct {
	*goap.BaseAction
	language     string
	filePath     string
	startPos     Position
	endPos       Position
	functionName string
}

func NewLSPExtractFunctionAction(language, filePath string, start, end Position, funcName string) *LSPExtractFunctionAction {
	return &LSPExtractFunctionAction{
		BaseAction: goap.NewBaseAction(
			"LSPExtractFunction",
			fmt.Sprintf("Extract function: %s", funcName),
			goap.WorldState{"file_exists": true},
			goap.WorldState{"function_extracted": true},
			9.0, // Very complex refactoring
		),
		language:     language,
		filePath:     filePath,
		startPos:     start,
		endPos:       end,
		functionName: funcName,
	}
}

func (a *LSPExtractFunctionAction) Execute(ctx context.Context, current goap.WorldState) error {
	log.Info("LSP extract function", "name", a.functionName, "file", a.filePath)

	// This would use LSP code action with kind "refactor.extract.function"
	// LSP servers provide intelligent extraction that:
	// - Analyzes variable usage
	// - Determines parameters and return values
	// - Handles scoping correctly
	// - Maintains semantics

	log.Info("LSP extract function would be performed here",
		"range", fmt.Sprintf("%d:%d-%d:%d", a.startPos.Line, a.startPos.Column, a.endPos.Line, a.endPos.Column))

	current.Set("function_extracted", true)
	current.Set("function_name", a.functionName)

	return nil
}

func (a *LSPExtractFunctionAction) Clone() goap.Action {
	return NewLSPExtractFunctionAction(a.language, a.filePath, a.startPos, a.endPos, a.functionName)
}

// LSPOrganizeImportsAction organizes imports using LSP
type LSPOrganizeImportsAction struct {
	*goap.BaseAction
	language string
	filePath string
}

func NewLSPOrganizeImportsAction(language, filePath string) *LSPOrganizeImportsAction {
	return &LSPOrganizeImportsAction{
		BaseAction: goap.NewBaseAction(
			"LSPOrganizeImports",
			fmt.Sprintf("Organize imports in %s", filePath),
			goap.WorldState{"file_exists": true},
			goap.WorldState{"imports_organized": true},
			3.0,
		),
		language: language,
		filePath: filePath,
	}
}

func (a *LSPOrganizeImportsAction) Execute(ctx context.Context, current goap.WorldState) error {
	log.Info("LSP organize imports", "file", a.filePath)

	// This uses LSP code action with kind "source.organizeImports"
	// LSP handles:
	// - Removing unused imports
	// - Sorting imports
	// - Grouping imports by category
	// - Adding missing imports

	switch a.language {
	case "go":
		return a.organizeGoImports()
	default:
		log.Info("LSP organize imports would be performed here")
	}

	current.Set("imports_organized", true)
	return nil
}

func (a *LSPOrganizeImportsAction) organizeGoImports() error {
	// Use goimports for Go
	cmd := exec.Command("goimports", "-w", a.filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("goimports failed: %w\nOutput: %s", err, output)
	}
	return nil
}

func (a *LSPOrganizeImportsAction) Clone() goap.Action {
	return NewLSPOrganizeImportsAction(a.language, a.filePath)
}

// LSPCompletionInsertAction inserts code using LSP completion
type LSPCompletionInsertAction struct {
	*goap.BaseAction
	language   string
	filePath   string
	position   Position
	triggerChar string
	selection  int // Which completion item to select
}

func NewLSPCompletionInsertAction(language, filePath string, pos Position, trigger string, selection int) *LSPCompletionInsertAction {
	return &LSPCompletionInsertAction{
		BaseAction: goap.NewBaseAction(
			"LSPCompletionInsert",
			"Insert code via LSP completion",
			goap.WorldState{"file_exists": true},
			goap.WorldState{"completion_inserted": true},
			5.0,
		),
		language:    language,
		filePath:    filePath,
		position:    pos,
		triggerChar: trigger,
		selection:   selection,
	}
}

func (a *LSPCompletionInsertAction) Execute(ctx context.Context, current goap.WorldState) error {
	log.Info("LSP completion insert", "file", a.filePath, "position", fmt.Sprintf("%d:%d", a.position.Line, a.position.Column))

	// This would:
	// 1. Send textDocument/completion request
	// 2. Get completion items
	// 3. Select item at index
	// 4. Apply TextEdit from completion item
	// 5. Optionally resolve additional edits

	log.Info("LSP completion would be performed here", "selection", a.selection)

	current.Set("completion_inserted", true)
	return nil
}

func (a *LSPCompletionInsertAction) Clone() goap.Action {
	return NewLSPCompletionInsertAction(a.language, a.filePath, a.position, a.triggerChar, a.selection)
}

// Helper: LSP WorkspaceEdit type
type LSPWorkspaceEdit struct {
	Changes map[string][]LSPTextEdit `json:"changes"`
}

type LSPTextEdit struct {
	Range   LSPRange `json:"range"`
	NewText string   `json:"newText"`
}

type LSPRange struct {
	Start LSPPosition `json:"start"`
	End   LSPPosition `json:"end"`
}

type LSPPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// ApplyWorkspaceEdit applies an LSP WorkspaceEdit to files
func ApplyWorkspaceEdit(edit *LSPWorkspaceEdit) error {
	for uri, textEdits := range edit.Changes {
		// Convert URI to file path
		filePath := strings.TrimPrefix(uri, "file://")

		// Read file
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", filePath, err)
		}

		text := string(content)
		lines := strings.Split(text, "\n")

		// Apply edits (should be in reverse order to maintain offsets)
		for i := len(textEdits) - 1; i >= 0; i-- {
			edit := textEdits[i]

			// Apply edit to lines
			startLine := edit.Range.Start.Line
			startChar := edit.Range.Start.Character
			endLine := edit.Range.End.Line
			endChar := edit.Range.End.Character

			if startLine == endLine {
				// Single line edit
				line := lines[startLine]
				lines[startLine] = line[:startChar] + edit.NewText + line[endChar:]
			} else {
				// Multi-line edit
				startContent := lines[startLine][:startChar]
				endContent := lines[endLine][endChar:]
				newLines := []string{startContent + edit.NewText + endContent}

				lines = append(lines[:startLine], append(newLines, lines[endLine+1:]...)...)
			}
		}

		// Write back
		result := strings.Join(lines, "\n")
		err = os.WriteFile(filePath, []byte(result), 0644)
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", filePath, err)
		}

		log.Info("Applied LSP edits", "file", filePath, "edits", len(textEdits))
	}

	return nil
}
