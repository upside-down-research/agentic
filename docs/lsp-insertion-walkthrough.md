# LSP-Based Code Insertion: Go and Rust

## How the Reasoning Agent Adds Code to the Middle of a Function

This document walks through how `GoLSPAction` and `RustLSPAction` (the 1st class citizens) add new code in the middle of a function using Language Server Protocol.

---

## The Problem

Adding code in the middle of a function is tricky:
- **Naive text insertion**: Breaks syntax, indentation, formatting
- **Line-based edits**: Don't understand scope, context, or semantics
- **AST manipulation**: Language-specific, requires parsing

**LSP Solution**: Language servers understand the syntax tree and provide semantic-aware edits.

---

## Go Example: Adding Error Handling

### Original Go Function

```go
func ProcessUser(id string) *User {
    user := db.FindUser(id)
    return user
}
```

### Goal: Add error handling after line 2

We want:
```go
func ProcessUser(id string) (*User, error) {
    user, err := db.FindUser(id)
    if err != nil {
        return nil, fmt.Errorf("failed to find user: %w", err)
    }
    return user, nil
}
```

### How GoLSPAction Works

#### Step 1: Reasoning Agent Creates Action

```go
// In the reasoning agent's planning phase
edit := LSPEdit{
    Type: "textEdit",
    Params: map[string]interface{}{
        "range": map[string]interface{}{
            "start": map[string]int{"line": 2, "character": 29}, // After db.FindUser(id)
            "end":   map[string]int{"line": 2, "character": 29}, // Same position = insertion
        },
        "newText": ", err := db.FindUser(id)\n    if err != nil {\n        return nil, fmt.Errorf(\"failed to find user: %w\", err)\n    }",
    },
}

action := NewGoLSPAction("user_service.go", []LSPEdit{edit})
```

#### Step 2: Execute via gopls

```go
func (a *GoLSPAction) Execute(ctx context.Context, current goap.WorldState) error {
    // 1. Start gopls LSP server (if not running)
    server := startGoplsServer(ctx)

    // 2. Open document with gopls
    server.DidOpen(&lsp.DidOpenTextDocumentParams{
        TextDocument: lsp.TextDocumentItem{
            URI:  "file:///path/to/user_service.go",
            LanguageID: "go",
            Version: 1,
            Text: readFileContent(a.filePath),
        },
    })

    // 3. Request semantic analysis
    // gopls builds AST and validates syntax
    diagnostics := server.GetDiagnostics("file:///path/to/user_service.go")

    // 4. Apply text edit via LSP
    workspaceEdit := &lsp.WorkspaceEdit{
        Changes: map[string][]lsp.TextEdit{
            "file:///path/to/user_service.go": {
                {
                    Range: lsp.Range{
                        Start: lsp.Position{Line: 2, Character: 29},
                        End:   lsp.Position{Line: 2, Character: 29},
                    },
                    NewText: ", err",
                },
                {
                    Range: lsp.Range{
                        Start: lsp.Position{Line: 3, Character: 0},
                        End:   lsp.Position{Line: 3, Character: 0},
                    },
                    NewText: "    if err != nil {\n        return nil, fmt.Errorf(\"failed to find user: %w\", err)\n    }\n",
                },
            },
        },
    }

    // 5. gopls validates the edit maintains valid AST
    // 6. Apply the workspace edit
    ApplyWorkspaceEdit(workspaceEdit)

    // 7. Request gopls to format the result
    server.Formatting(&lsp.DocumentFormattingParams{
        TextDocument: lsp.TextDocumentIdentifier{
            URI: "file:///path/to/user_service.go",
        },
    })

    // 8. Request gopls to organize imports (adds fmt if needed)
    server.CodeAction(&lsp.CodeActionParams{
        TextDocument: lsp.TextDocumentIdentifier{
            URI: "file:///path/to/user_service.go",
        },
        Context: lsp.CodeActionContext{
            Only: []lsp.CodeActionKind{"source.organizeImports"},
        },
    })

    return nil
}
```

#### Step 3: What gopls Does Internally

1. **Parses AST**: Understands function structure, scope, variable names
2. **Type checks**: Knows `db.FindUser(id)` returns `(*User, error)`
3. **Validates edit**: Ensures new code is syntactically valid
4. **Adjusts indentation**: Matches surrounding code style
5. **Updates imports**: Adds `"fmt"` if not present
6. **Preserves semantics**: Doesn't break variable scopes or control flow

#### Step 4: Result

gopls returns a WorkspaceEdit with:
- Properly indented code
- Correct import additions
- Valid syntax
- Type-checked semantics

---

## Rust Example: Adding Logging

### Original Rust Function

```rust
fn calculate_total(items: &[Item]) -> f64 {
    let mut sum = 0.0;
    for item in items {
        sum += item.price;
    }
    sum
}
```

### Goal: Add logging in the middle of the loop

We want:
```rust
fn calculate_total(items: &[Item]) -> f64 {
    let mut sum = 0.0;
    for item in items {
        log::debug!("Processing item: {:?}", item.name);
        sum += item.price;
    }
    sum
}
```

### How RustLSPAction Works

#### Step 1: Reasoning Agent Creates Action

```go
edit := LSPEdit{
    Type: "textEdit",
    Params: map[string]interface{}{
        "range": map[string]interface{}{
            "start": map[string]int{"line": 3, "character": 27}, // After opening {
            "end":   map[string]int{"line": 3, "character": 27},
        },
        "newText": "\n        log::debug!(\"Processing item: {:?}\", item.name);",
    },
}

action := NewRustLSPAction("calculator.rs", []LSPEdit{edit})
```

#### Step 2: Execute via rust-analyzer

```go
func (a *RustLSPAction) Execute(ctx context.Context, current goap.WorldState) error {
    // 1. Start rust-analyzer LSP server
    server := startRustAnalyzerServer(ctx)

    // 2. Initialize with Cargo.toml location
    server.Initialize(&lsp.InitializeParams{
        RootURI: "file:///path/to/rust/project",
        Capabilities: lsp.ClientCapabilities{
            TextDocument: lsp.TextDocumentClientCapabilities{
                Completion: &lsp.CompletionClientCapabilities{
                    CompletionItem: &lsp.CompletionItemCapabilities{
                        SnippetSupport: true,
                    },
                },
            },
        },
    })

    // 3. Open document
    server.DidOpen(&lsp.DidOpenTextDocumentParams{
        TextDocument: lsp.TextDocumentItem{
            URI:  "file:///path/to/calculator.rs",
            LanguageID: "rust",
            Version: 1,
            Text: readFileContent(a.filePath),
        },
    })

    // 4. Request completion at the insertion point (optional)
    // rust-analyzer provides context-aware suggestions
    completions := server.Completion(&lsp.CompletionParams{
        TextDocument: lsp.TextDocumentIdentifier{
            URI: "file:///path/to/calculator.rs",
        },
        Position: lsp.Position{Line: 3, Character: 27},
        Context: lsp.CompletionContext{
            TriggerKind: lsp.Invoked,
        },
    })
    // rust-analyzer knows: item is in scope, has .name field

    // 5. Apply the edit
    workspaceEdit := &lsp.WorkspaceEdit{
        Changes: map[string][]lsp.TextEdit{
            "file:///path/to/calculator.rs": {
                {
                    Range: lsp.Range{
                        Start: lsp.Position{Line: 3, Character: 27},
                        End:   lsp.Position{Line: 3, Character: 27},
                    },
                    NewText: "\n        log::debug!(\"Processing item: {:?}\", item.name);",
                },
            },
        },
    }

    ApplyWorkspaceEdit(workspaceEdit)

    // 6. rust-analyzer provides diagnostics
    diagnostics := server.GetDiagnostics("file:///path/to/calculator.rs")
    // Might say: "log is not in scope, add 'use log;'"

    // 7. Apply code action to fix
    codeActions := server.CodeAction(&lsp.CodeActionParams{
        TextDocument: lsp.TextDocumentIdentifier{
            URI: "file:///path/to/calculator.rs",
        },
        Range: diagnostics[0].Range,
        Context: lsp.CodeActionContext{
            Diagnostics: diagnostics,
        },
    })
    // rust-analyzer suggests: "Add 'use log;' at top"

    // 8. Apply suggestion
    ApplyWorkspaceEdit(codeActions[0].Edit)

    // 9. Format with rustfmt
    server.Formatting(&lsp.DocumentFormattingParams{
        TextDocument: lsp.TextDocumentIdentifier{
            URI: "file:///path/to/calculator.rs",
        },
    })

    return nil
}
```

#### Step 3: What rust-analyzer Does Internally

1. **Parses HIR (High-level IR)**: Understands Rust's type system
2. **Macro expansion**: Validates `log::debug!` macro syntax
3. **Borrow checker**: Ensures `item.name` borrow is valid
4. **Trait resolution**: Knows `item.name` implements `Debug` trait
5. **Import inference**: Suggests adding `use log;`
6. **Formatting**: Applies rustfmt style
7. **Clippy lints**: Runs linters on new code

#### Step 4: Rust-Specific Features

**Macro Expansion** (rust-analyzer exclusive):
```go
// Expand macro to see what code it generates
expandAction := LSPEdit{
    Type: "expand_macro",
    Params: map[string]interface{}{
        "line":      4,
        "character": 8,
    },
}
```

rust-analyzer shows:
```rust
// log::debug! expands to:
if log::log_enabled!(log::Level::Debug) {
    log::log!(log::Level::Debug, "Processing item: {:?}", item.name);
}
```

---

## The LSP Protocol Flow (Detailed)

### Phase 1: Server Initialization

```
Client (GoLSPAction)          Server (gopls/rust-analyzer)
        |                                |
        |------- initialize -----------→|
        |                                | (starts up)
        |←----- capabilities ----------|
        |                                |
        |------- initialized ----------→|
        |                                |
```

### Phase 2: Document Lifecycle

```
Client                        Server
        |                                |
        |--- textDocument/didOpen -----→| (parse file, build AST)
        |                                |
        |←-- textDocument/publishDiagnostics --| (syntax errors, warnings)
        |                                |
```

### Phase 3: Code Insertion

```
Client                        Server
        |                                |
        |-- workspace/applyEdit -------→| (validate edit)
        |                                | (check AST still valid)
        |                                | (update symbol table)
        |←----- edit result -----------|
        |                                |
```

### Phase 4: Semantic Fixups

```
Client                        Server
        |                                |
        |-- textDocument/codeAction ---→| (find missing imports)
        |←----- actions --------------| (suggest fixes)
        |                                |
        |-- workspace/applyEdit -------→| (apply import)
        |←----- edit result -----------|
        |                                |
        |-- textDocument/formatting ---→| (format code)
        |←----- edits -----------------| (indentation, style)
        |                                |
```

---

## Concrete Example: Full Flow

### Scenario: Insert validation check in Go function

**Before:**
```go
// file: handlers/user.go
package handlers

func CreateUser(w http.ResponseWriter, r *http.Request) {
    var user User
    json.NewDecoder(r.Body).Decode(&user)
    db.Save(&user)
    json.NewEncoder(w).Encode(user)
}
```

**After (Goal):**
```go
package handlers

import (
    "encoding/json"
    "net/http"
    "fmt"
)

func CreateUser(w http.ResponseWriter, r *http.Request) {
    var user User
    if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
        http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
        return
    }
    // Validation check (NEWLY INSERTED)
    if user.Email == "" {
        http.Error(w, "Email is required", http.StatusBadRequest)
        return
    }
    if err := db.Save(&user); err != nil {
        http.Error(w, fmt.Sprintf("Save failed: %v", err), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(user)
}
```

### Reasoning Agent Action Sequence

```go
// 1. Create insertion edit
insertValidation := LSPEdit{
    Type: "codeAction",
    Params: map[string]interface{}{
        "kind": "refactor.insert",
        "position": map[string]int{
            "line":      4, // After decode error check
            "character": 5,
        },
        "codeSnippet": `if user.Email == "" {
        http.Error(w, "Email is required", http.StatusBadRequest)
        return
    }`,
    },
}

// 2. Create Go LSP action
action := NewGoLSPAction("handlers/user.go", []LSPEdit{insertValidation})

// 3. Execute (gopls handles everything)
err := action.Execute(ctx, worldState)
```

### What gopls Does (Automatically)

1. **Validates position**: Line 4, after the decode error check
2. **Checks scope**: `user` variable is in scope
3. **Type inference**: `user.Email` is valid (knows User struct)
4. **Import management**: `http` and `fmt` already imported, no change needed
5. **Indentation**: Matches surrounding code (4 spaces)
6. **Control flow**: Validates `return` is appropriate
7. **Formatting**: Applies gofmt style
8. **Returns WorkspaceEdit**: Client applies it

---

## Advantages Over Text Manipulation

### Text-Based Insertion (Naive)
```go
// Read file as string
content := readFile("user.go")

// Find line 4
lines := strings.Split(content, "\n")

// Insert text
newLine := `    if user.Email == "" {`
lines = append(lines[:4], append([]string{newLine}, lines[4:]...)...)

// PROBLEMS:
// ❌ Wrong indentation if tabs vs spaces differ
// ❌ Doesn't know if user.Email field exists
// ❌ Doesn't add imports if needed
// ❌ Breaks if line numbers change
// ❌ No syntax validation
// ❌ Can't handle multi-line insertion easily
```

### LSP-Based Insertion (Semantic)
```go
// Create semantic edit
edit := NewGoLSPAction("user.go", []LSPEdit{{
    Type: "insertAfterStatement",
    Params: map[string]interface{}{
        "statement": "json.NewDecoder",
        "code": `if user.Email == "" { ... }`,
    },
}})

// BENEFITS:
// ✅ gopls validates user.Email exists
// ✅ Correct indentation (gopls knows the style)
// ✅ Auto-adds imports if needed
// ✅ Position relative to AST node, not line number
// ✅ Full syntax validation
// ✅ Handles multi-line perfectly
// ✅ Type-checked
```

---

## Why Go and Rust Are 1st Class Citizens

### gopls Advantages
- **Fast**: Incremental compilation
- **Accurate**: Official Go team implementation
- **Complete**: Supports all Go features (generics, workspaces)
- **Well-integrated**: Used by VSCode, vim, emacs, etc.
- **Code actions**: Auto-fix imports, extract function, inline variable

### rust-analyzer Advantages
- **Macro-aware**: Can expand and validate macros
- **Trait resolution**: Understands complex type system
- **Borrow checker**: Validates lifetime correctness
- **Inlay hints**: Shows inferred types
- **Smart refactorings**: Extract function, inline module, etc.
- **Clippy integration**: Linting built-in

### Both Share
- **Standards-based**: LSP protocol
- **Context-aware**: Know variable scope, types, imports
- **Auto-formatting**: gofmt / rustfmt built-in
- **Diagnostic-driven**: Suggest fixes for errors
- **Refactoring-safe**: Maintain semantic correctness

---

## How the Reasoning Agent Uses This

### Planning Phase (GOFAI)
```go
// Orchestrator decides: "Need to add validation"
goal := goap.NewGoal(
    "AddEmailValidation",
    "Ensure user email is not empty",
    goap.WorldState{
        "validation_added": true,
        "tests_pass": true,
    },
    priority,
)
```

### Execution Phase (LLM + LSP)
```go
// 1. LLM generates the validation code
prompt := "Generate Go code to validate user.Email is not empty"
validationCode := llm.Generate(prompt)

// 2. LSP inserts it semantically
edit := LSPEdit{
    Type: "insertCode",
    Params: map[string]interface{}{
        "afterNode": "DecodeStatement",
        "code": validationCode,
    },
}

action := NewGoLSPAction("user.go", []LSPEdit{edit})
executor.Execute(action)

// 3. gopls/rust-analyzer ensures correctness
// - Validates syntax
// - Checks types
// - Formats code
// - Adds imports
// - Verifies semantics
```

### Verification Phase (GOFAI)
```go
// Orchestrator verifies the edit succeeded
if !fileCompiles("user.go") {
    return errors.New("LSP edit broke compilation")
}

if !testsPass("user_test.go") {
    return errors.New("validation broke tests")
}

// Mark goal complete
worldState.Set("validation_added", true)
```

---

## Summary

**Text insertion is hard because:**
- Syntax is fragile
- Indentation varies
- Imports need updating
- Types must be checked
- Scopes can break

**LSP makes it easy because:**
1. **Language servers understand semantics** (not just text)
2. **They maintain AST** (know structure, not just lines)
3. **They validate edits** (won't break your code)
4. **They auto-fix** (imports, formatting, style)
5. **They're standardized** (same protocol everywhere)

**Go and Rust are 1st class citizens because:**
- `gopls` and `rust-analyzer` are the best-in-class LSP servers
- They're officially supported and actively developed
- They understand complex language features (macros, generics, traits, lifetimes)
- The reasoning agent can trust them for semantic correctness

**The reasoning agent benefits because:**
- GOFAI plans WHAT to insert
- LLM generates the code content
- LSP ensures HOW to insert it correctly
- Together: semantic-aware, automated code editing at scale
