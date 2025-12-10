package actions

import (
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"upside-down-research.com/oss/agentic/internal/goap"
)

// FileEditAction performs text-based file edits (fallback for non-AST languages)
type FileEditAction struct {
	*goap.BaseAction
	filePath string
	edits    []TextEdit
}

type TextEdit struct {
	SearchText  string
	ReplaceText string
	All         bool // Replace all occurrences
}

func NewFileEditAction(filePath string, edits []TextEdit) *FileEditAction {
	return &FileEditAction{
		BaseAction: goap.NewBaseAction(
			"FileEdit",
			fmt.Sprintf("Edit file: %s (%d edits)", filePath, len(edits)),
			goap.WorldState{"file_exists": true},
			goap.WorldState{"file_edited": true},
			3.0, // Text edits are simpler than AST
		),
		filePath: filePath,
		edits:    edits,
	}
}

func (a *FileEditAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for FileEdit")
	}

	log.Info("Editing file (text-based)", "file", a.filePath, "edits", len(a.edits))

	// Read file
	content, err := os.ReadFile(a.filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	result := string(content)

	// Apply edits
	for i, edit := range a.edits {
		if edit.All {
			result = strings.ReplaceAll(result, edit.SearchText, edit.ReplaceText)
		} else {
			result = strings.Replace(result, edit.SearchText, edit.ReplaceText, 1)
		}
		log.Debug("Applied edit", "index", i, "all", edit.All)
	}

	// Write back
	err = os.WriteFile(a.filePath, []byte(result), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	current.Set("file_edited", true)
	current.Set("edited_file", a.filePath)

	log.Info("File edited successfully", "file", a.filePath)
	return nil
}

func (a *FileEditAction) Clone() goap.Action {
	return NewFileEditAction(a.filePath, a.edits)
}

// === AST-BASED EDITS: The State of the Art! ===

// GoASTEditAction performs AST-based edits on Go files
// This is MUCH better than text manipulation!
type GoASTEditAction struct {
	*goap.BaseAction
	filePath string
	edits    []ASTEdit
}

type ASTEdit interface {
	Apply(fset *token.FileSet, file *ast.File) error
	Description() string
}

func NewGoASTEditAction(filePath string, edits []ASTEdit) *GoASTEditAction {
	return &GoASTEditAction{
		BaseAction: goap.NewBaseAction(
			"GoASTEdit",
			fmt.Sprintf("AST-based edit of Go file: %s", filePath),
			goap.WorldState{"file_exists": true},
			goap.WorldState{"go_ast_edited": true},
			5.0, // AST editing is more complex but safer
		),
		filePath: filePath,
		edits:    edits,
	}
}

func (a *GoASTEditAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for GoASTEdit")
	}

	log.Info("Editing Go file with AST", "file", a.filePath, "edits", len(a.edits))

	// Parse file into AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, a.filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse Go file: %w", err)
	}

	// Apply AST edits
	for i, edit := range a.edits {
		log.Debug("Applying AST edit", "index", i, "description", edit.Description())
		err := edit.Apply(fset, file)
		if err != nil {
			return fmt.Errorf("AST edit %d failed: %w", i, err)
		}
	}

	// Format and write back
	f, err := os.Create(a.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for writing: %w", err)
	}
	defer f.Close()

	err = format.Node(f, fset, file)
	if err != nil {
		return fmt.Errorf("failed to format AST: %w", err)
	}

	current.Set("go_ast_edited", true)
	current.Set("edited_file", a.filePath)

	log.Info("Go file edited successfully with AST", "file", a.filePath)
	return nil
}

func (a *GoASTEditAction) Clone() goap.Action {
	return NewGoASTEditAction(a.filePath, a.edits)
}

// === CONCRETE AST EDITS FOR GO ===

// RenameIdentifierEdit renames an identifier throughout the file
type RenameIdentifierEdit struct {
	OldName string
	NewName string
}

func (e *RenameIdentifierEdit) Description() string {
	return fmt.Sprintf("Rename %s -> %s", e.OldName, e.NewName)
}

func (e *RenameIdentifierEdit) Apply(fset *token.FileSet, file *ast.File) error {
	// Walk the AST and rename all occurrences
	ast.Inspect(file, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {
			if ident.Name == e.OldName {
				ident.Name = e.NewName
			}
		}
		return true
	})
	return nil
}

// AddImportEdit adds an import to the file
type AddImportEdit struct {
	ImportPath string
	Alias      string
}

func (e *AddImportEdit) Description() string {
	if e.Alias != "" {
		return fmt.Sprintf("Add import: %s %q", e.Alias, e.ImportPath)
	}
	return fmt.Sprintf("Add import: %q", e.ImportPath)
}

func (e *AddImportEdit) Apply(fset *token.FileSet, file *ast.File) error {
	// Check if import already exists
	for _, imp := range file.Imports {
		if imp.Path.Value == fmt.Sprintf("%q", e.ImportPath) {
			return nil // Already exists
		}
	}

	// Create new import spec
	importSpec := &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf("%q", e.ImportPath),
		},
	}

	if e.Alias != "" {
		importSpec.Name = &ast.Ident{Name: e.Alias}
	}

	// Add to first import declaration, or create one
	if len(file.Decls) > 0 {
		if genDecl, ok := file.Decls[0].(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			genDecl.Specs = append(genDecl.Specs, importSpec)
			return nil
		}
	}

	// Create new import declaration
	importDecl := &ast.GenDecl{
		Tok: token.IMPORT,
		Specs: []ast.Spec{importSpec},
	}

	// Insert at beginning
	file.Decls = append([]ast.Decl{importDecl}, file.Decls...)

	return nil
}

// RemoveImportEdit removes an import from the file
type RemoveImportEdit struct {
	ImportPath string
}

func (e *RemoveImportEdit) Description() string {
	return fmt.Sprintf("Remove import: %q", e.ImportPath)
}

func (e *RemoveImportEdit) Apply(fset *token.FileSet, file *ast.File) error {
	targetPath := fmt.Sprintf("%q", e.ImportPath)

	for i, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			newSpecs := []ast.Spec{}
			for _, spec := range genDecl.Specs {
				if importSpec, ok := spec.(*ast.ImportSpec); ok {
					if importSpec.Path.Value != targetPath {
						newSpecs = append(newSpecs, spec)
					}
				}
			}
			genDecl.Specs = newSpecs

			// Remove empty import declaration
			if len(newSpecs) == 0 {
				file.Decls = append(file.Decls[:i], file.Decls[i+1:]...)
			}
			break
		}
	}

	return nil
}

// AddFunctionEdit adds a new function to the file
type AddFunctionEdit struct {
	FunctionName string
	Parameters   []string
	ReturnType   string
	Body         string
}

func (e *AddFunctionEdit) Description() string {
	return fmt.Sprintf("Add function: %s", e.FunctionName)
}

func (e *AddFunctionEdit) Apply(fset *token.FileSet, file *ast.File) error {
	// Parse the function body
	funcCode := fmt.Sprintf("package p\nfunc %s(%s) %s { %s }",
		e.FunctionName,
		strings.Join(e.Parameters, ", "),
		e.ReturnType,
		e.Body,
	)

	funcFile, err := parser.ParseFile(fset, "", funcCode, 0)
	if err != nil {
		return fmt.Errorf("failed to parse function: %w", err)
	}

	// Extract the function declaration
	if len(funcFile.Decls) == 0 {
		return fmt.Errorf("no function declaration found")
	}

	funcDecl, ok := funcFile.Decls[0].(*ast.FuncDecl)
	if !ok {
		return fmt.Errorf("not a function declaration")
	}

	// Add to file
	file.Decls = append(file.Decls, funcDecl)

	return nil
}

// ModifyFunctionBodyEdit modifies the body of an existing function
type ModifyFunctionBodyEdit struct {
	FunctionName string
	NewBody      string
}

func (e *ModifyFunctionBodyEdit) Description() string {
	return fmt.Sprintf("Modify function body: %s", e.FunctionName)
}

func (e *ModifyFunctionBodyEdit) Apply(fset *token.FileSet, file *ast.File) error {
	// Parse the new body
	bodyCode := fmt.Sprintf("package p\nfunc dummy() { %s }", e.NewBody)
	bodyFile, err := parser.ParseFile(fset, "", bodyCode, 0)
	if err != nil {
		return fmt.Errorf("failed to parse new body: %w", err)
	}

	if len(bodyFile.Decls) == 0 {
		return fmt.Errorf("no function found in body")
	}

	dummyFunc, ok := bodyFile.Decls[0].(*ast.FuncDecl)
	if !ok {
		return fmt.Errorf("not a function")
	}

	newBody := dummyFunc.Body

	// Find and modify the target function
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if funcDecl.Name.Name == e.FunctionName {
				funcDecl.Body = newBody
				found = true
				return false
			}
		}
		return true
	})

	if !found {
		return fmt.Errorf("function %s not found", e.FunctionName)
	}

	return nil
}

// AddStructFieldEdit adds a field to a struct
type AddStructFieldEdit struct {
	StructName string
	FieldName  string
	FieldType  string
	Tag        string
}

func (e *AddStructFieldEdit) Description() string {
	return fmt.Sprintf("Add field %s to struct %s", e.FieldName, e.StructName)
}

func (e *AddStructFieldEdit) Apply(fset *token.FileSet, file *ast.File) error {
	found := false

	ast.Inspect(file, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			if typeSpec.Name.Name == e.StructName {
				if structType, ok := typeSpec.Type.(*ast.StructType); ok {
					// Create new field
					field := &ast.Field{
						Names: []*ast.Ident{{Name: e.FieldName}},
						Type:  &ast.Ident{Name: e.FieldType},
					}

					if e.Tag != "" {
						field.Tag = &ast.BasicLit{
							Kind:  token.STRING,
							Value: fmt.Sprintf("`%s`", e.Tag),
						}
					}

					// Add field
					structType.Fields.List = append(structType.Fields.List, field)
					found = true
					return false
				}
			}
		}
		return true
	})

	if !found {
		return fmt.Errorf("struct %s not found", e.StructName)
	}

	return nil
}

// AddCommentEdit adds a comment to a declaration
type AddCommentEdit struct {
	TargetName string
	Comment    string
}

func (e *AddCommentEdit) Description() string {
	return fmt.Sprintf("Add comment to %s", e.TargetName)
}

func (e *AddCommentEdit) Apply(fset *token.FileSet, file *ast.File) error {
	found := false

	ast.Inspect(file, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.FuncDecl:
			if decl.Name.Name == e.TargetName {
				if decl.Doc == nil {
					decl.Doc = &ast.CommentGroup{}
				}
				decl.Doc.List = append(decl.Doc.List, &ast.Comment{
					Text: "// " + e.Comment,
				})
				found = true
				return false
			}
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if typeSpec.Name.Name == e.TargetName {
						if decl.Doc == nil {
							decl.Doc = &ast.CommentGroup{}
						}
						decl.Doc.List = append(decl.Doc.List, &ast.Comment{
							Text: "// " + e.Comment,
						})
						found = true
						return false
					}
				}
			}
		}
		return true
	})

	if !found {
		return fmt.Errorf("target %s not found", e.TargetName)
	}

	return nil
}
