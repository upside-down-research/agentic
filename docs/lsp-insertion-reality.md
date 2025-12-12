# LSP Code Insertion: The Reality

## The Truth About How LSP Actually Works

### What I Said Before (Idealized)
> "LSP semantically inserts code with full awareness!"

### What Actually Happens (Reality)
1. **Insert text first** (still text manipulation, but smarter positioning)
2. **LSP analyzes the result** (finds errors, missing imports, bad formatting)
3. **LSP suggests fixes** (via diagnostics and code actions)
4. **Apply LSP's suggested fixes** (imports, formatting, refactorings)
5. **Repeat until clean**

---

## The Real Flow: Insert → Analyze → Fix

### Example: Adding Error Handling to Go Function

#### Original Code
```go
// file: user.go
package main

func ProcessUser(id string) *User {
    user := db.FindUser(id)
    return user
}
```

#### Goal: Add error handling after line 2

---

### Step 1: TEXT INSERTION (Manual/Planned)

We **don't** ask LSP to insert for us. We calculate the position and insert text:

```go
func (a *GoLSPAction) Execute(ctx context.Context, current goap.WorldState) error {
    // Read the file
    content, _ := os.ReadFile(a.filePath)
    lines := strings.Split(string(content), "\n")

    // STEP 1: INSERT TEXT MANUALLY
    // We know we want to insert after line 1 (user := db.FindUser(id))
    insertionPoint := 2  // After line 1 (0-indexed line 2)

    newLines := []string{
        "    if err != nil {",
        "        return nil, fmt.Errorf(\"failed to find user: %w\", err)",
        "    }",
    }

    // Splice in new lines
    lines = append(lines[:insertionPoint], append(newLines, lines[insertionPoint:]...)...)

    // Write back
    result := strings.Join(lines, "\n")
    os.WriteFile(a.filePath, []byte(result), 0644)

    // NOW the file looks like this:
    // package main
    //
    // func ProcessUser(id string) *User {
    //     user := db.FindUser(id)
    //     if err != nil {                              ← INSERTED
    //         return nil, fmt.Errorf("failed...")      ← INSERTED
    //     }                                            ← INSERTED
    //     return user
    // }

    // But it has PROBLEMS:
    // ❌ Line 1 still says "user :=" not "user, err :="
    // ❌ Missing "fmt" import
    // ❌ Variable "err" doesn't exist yet
    // ❌ May have wrong indentation
```

**At this point:** The file has been modified with text insertion, but it's **broken**.

---

### Step 2: LSP ANALYZES (Diagnostics)

Now we open the file with LSP and let it analyze:

```go
    // STEP 2: START LSP AND GET DIAGNOSTICS

    // Start gopls server
    lspConn := startLSPServer("gopls")

    // Tell gopls to open/analyze the file
    lspConn.Call("textDocument/didOpen", map[string]interface{}{
        "textDocument": map[string]interface{}{
            "uri":        "file:///path/to/user.go",
            "languageId": "go",
            "version":    1,
            "text":       result,  // The modified content
        },
    })

    // gopls analyzes and sends diagnostics
    diagnostics := lspConn.WaitForDiagnostics()

    // diagnostics contains:
    // [
    //   {
    //     "range": {"start": {"line": 2, "character": 7}, "end": {...}},
    //     "severity": 1,  // Error
    //     "message": "undeclared name: err",
    //     "source": "compiler"
    //   },
    //   {
    //     "range": {"start": {"line": 3, "character": 23}, "end": {...}},
    //     "severity": 1,  // Error
    //     "message": "undeclared name: fmt",
    //     "source": "compiler"
    //   }
    // ]
```

**gopls found the problems:**
- ❌ `err` is undeclared (we inserted code using it, but didn't declare it)
- ❌ `fmt` is undeclared (we used `fmt.Errorf` but didn't import it)

---

### Step 3: LSP SUGGESTS FIXES (Code Actions)

Now we ask gopls: "How do I fix these errors?"

```go
    // STEP 3: REQUEST CODE ACTIONS (FIXES)

    for _, diagnostic := range diagnostics {
        // Ask gopls: "What can I do about this error?"
        codeActions := lspConn.Call("textDocument/codeAction", map[string]interface{}{
            "textDocument": map[string]interface{}{
                "uri": "file:///path/to/user.go",
            },
            "range": diagnostic.Range,
            "context": map[string]interface{}{
                "diagnostics": []interface{}{diagnostic},
            },
        })

        // gopls responds with suggestions
    }

    // For "undeclared name: err", gopls suggests:
    // {
    //   "title": "Change 'user :=' to 'user, err :='",
    //   "kind": "quickfix",
    //   "edit": {
    //     "changes": {
    //       "file:///path/to/user.go": [
    //         {
    //           "range": {
    //             "start": {"line": 1, "character": 10},
    //             "end": {"line": 1, "character": 10}
    //           },
    //           "newText": ", err"
    //         }
    //       ]
    //     }
    //   }
    // }

    // For "undeclared name: fmt", gopls suggests:
    // {
    //   "title": "Add import \"fmt\"",
    //   "kind": "quickfix",
    //   "edit": {
    //     "changes": {
    //       "file:///path/to/user.go": [
    //         {
    //           "range": {
    //             "start": {"line": 0, "character": 13},
    //             "end": {"line": 0, "character": 13}
    //           },
    //           "newText": "\nimport \"fmt\"\n"
    //         }
    //       ]
    //     }
    //   }
    // }
```

**gopls is telling us:** "Here are text edits that will fix your broken code."

---

### Step 4: APPLY LSP'S FIXES

Now we apply the fixes gopls suggested:

```go
    // STEP 4: APPLY THE SUGGESTED FIXES

    for _, codeAction := range codeActions {
        if codeAction.Kind == "quickfix" {
            // Apply the WorkspaceEdit from the code action
            ApplyWorkspaceEdit(codeAction.Edit)
        }
    }

    // After applying fixes, file now looks like:
    // package main
    //
    // import "fmt"                                   ← ADDED BY GOPLS
    //
    // func ProcessUser(id string) *User {
    //     user, err := db.FindUser(id)               ← FIXED BY GOPLS
    //     if err != nil {
    //         return nil, fmt.Errorf("failed to find user: %w", err)
    //     }
    //     return user
    // }
```

---

### Step 5: LSP FORMATTING

Finally, we ask gopls to format the file:

```go
    // STEP 5: FORMAT THE CODE

    formattingEdits := lspConn.Call("textDocument/formatting", map[string]interface{}{
        "textDocument": map[string]interface{}{
            "uri": "file:///path/to/user.go",
        },
        "options": map[string]interface{}{
            "tabSize":      4,
            "insertSpaces": false,  // Use tabs (Go convention)
        },
    })

    // gopls returns formatting edits (fix indentation, spacing, etc.)
    ApplyWorkspaceEdit(formattingEdits)

    // FINAL RESULT:
    // package main
    //
    // import "fmt"
    //
    // func ProcessUser(id string) (*User, error) {    ← Return signature fixed
    //     user, err := db.FindUser(id)
    //     if err != nil {
    //         return nil, fmt.Errorf("failed to find user: %w", err)
    //     }
    //     return user, nil                              ← Return statement fixed
    // }
```

---

## The Real Sequence

```
┌─────────────────────────────────────────────────────────────┐
│ STEP 1: TEXT INSERTION (We do this)                        │
├─────────────────────────────────────────────────────────────┤
│ - Read file                                                 │
│ - Calculate insertion point (line number or after pattern) │
│ - Splice in new text                                        │
│ - Write file                                                │
│ - Result: File is BROKEN (missing imports, vars, etc.)     │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ STEP 2: LSP ANALYSIS (gopls/rust-analyzer does this)       │
├─────────────────────────────────────────────────────────────┤
│ - Open file in LSP server                                   │
│ - LSP parses and builds AST                                 │
│ - LSP type-checks                                           │
│ - LSP sends diagnostics (errors, warnings)                  │
│ - Result: List of problems                                  │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ STEP 3: LSP SUGGESTIONS (gopls/rust-analyzer does this)    │
├─────────────────────────────────────────────────────────────┤
│ - Request code actions for each diagnostic                  │
│ - LSP suggests fixes (add import, declare variable, etc.)   │
│ - Each suggestion includes a WorkspaceEdit                  │
│ - Result: List of text edits to apply                       │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ STEP 4: APPLY FIXES (We do this)                           │
├─────────────────────────────────────────────────────────────┤
│ - For each suggested fix                                    │
│ - Apply the text edits                                      │
│ - Write file                                                │
│ - Result: File is CORRECT but not formatted                │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ STEP 5: FORMATTING (LSP does this)                         │
├─────────────────────────────────────────────────────────────┤
│ - Request formatting edits                                  │
│ - LSP applies gofmt/rustfmt rules                           │
│ - Returns indentation/spacing edits                         │
│ - Apply those edits                                         │
│ - Result: File is CORRECT and FORMATTED                    │
└─────────────────────────────────────────────────────────────┘
```

---

## What LSP Actually Provides

### ❌ LSP Does NOT:
- Insert code for you semantically
- Magically know where to put new statements
- Write code based on intent

### ✅ LSP DOES:
- **Analyze** text you already inserted
- **Diagnose** problems (type errors, missing imports, undefined variables)
- **Suggest fixes** as text edits (add import, change declaration, etc.)
- **Format** code according to language conventions
- **Validate** that edits maintain syntactic/semantic correctness

---

## Concrete Example: The ApplyWorkspaceEdit Function

This is what actually applies LSP's suggestions (from our code):

```go
// From internal/goap/actions/lsp_edits.go:728-778

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

        // Apply edits (in reverse order to maintain offsets)
        for i := len(textEdits) - 1; i >= 0; i-- {
            edit := textEdits[i]

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
    }

    return nil
}
```

**This is still text manipulation!** But it's text manipulation **guided by LSP's semantic understanding**.

---

## Why This Still Beats Naive Text Insertion

### Naive Approach (No LSP)
```go
// 1. Insert code blindly
lines[4] = "    if err != nil { return nil, err }"

// 2. Hope it works
// ❌ No validation
// ❌ Missing import not detected
// ❌ Wrong variable name not caught
// ❌ Bad indentation persists
```

### LSP-Guided Approach (Our Way)
```go
// 1. Insert code (same text manipulation)
lines[4] = "    if err != nil { return nil, err }"

// 2. LSP analyzes
diagnostics := lsp.GetDiagnostics()
// ✅ "err undeclared" detected
// ✅ "err shadowed" detected

// 3. LSP suggests fixes
fixes := lsp.GetCodeActions(diagnostics)
// ✅ "Change user := to user, err :="
// ✅ "Add import fmt"

// 4. Apply fixes
ApplyWorkspaceEdit(fixes)

// 5. LSP formats
formatted := lsp.Format()
ApplyWorkspaceEdit(formatted)

// ✅ Code is correct, validated, formatted
```

---

## The Real Advantage

**LSP doesn't insert code for you.** But it:

1. **Catches errors** you made during text insertion
2. **Suggests fixes** with precise text edits
3. **Validates semantics** (types, scopes, imports)
4. **Enforces style** (gofmt/rustfmt)
5. **Provides confidence** that the result is valid

---

## How the Reasoning Agent Uses This

### Phase 1: GOFAI Plans Position
```go
// Orchestrator determines: "Insert validation after line 4"
position := CalculateInsertionPoint("after", "json.Decode")
```

### Phase 2: LLM Generates Code
```go
// LLM generates the validation logic
code := llm.Generate("Generate email validation check")
// Returns: "if user.Email == \"\" { ... }"
```

### Phase 3: Text Insertion
```go
// Insert the LLM-generated code at the planned position
InsertText(filePath, position, code)
// File is now modified but potentially broken
```

### Phase 4: LSP Validation & Fixing
```go
// Start LSP and analyze
lsp.DidOpen(filePath)
diagnostics := lsp.GetDiagnostics()

// Get suggested fixes
for _, diag := range diagnostics {
    fixes := lsp.GetCodeActions(diag)
    for _, fix := range fixes {
        ApplyWorkspaceEdit(fix.Edit)
    }
}

// Format
formatted := lsp.Format()
ApplyWorkspaceEdit(formatted)

// File is now correct and formatted
```

### Phase 5: GOFAI Verification
```go
// Verify compilation
if !lsp.DiagnosticsClean() {
    return error("LSP validation failed")
}

// Verify tests
if !runTests() {
    return error("Tests failed")
}

// Mark action complete
```

---

## Summary: The Reality

**The truth:**
1. We insert text at calculated positions (still text manipulation)
2. LSP analyzes the result (finds problems)
3. LSP suggests specific text edits to fix problems
4. We apply those edits
5. LSP formats the result
6. We verify it compiles and tests pass

**The advantage:**
- LSP **validates** our insertions
- LSP **fixes** mistakes (missing imports, wrong declarations)
- LSP **formats** to standard style
- LSP **ensures** semantic correctness

**The workflow:**
- **GOFAI** reasons about WHAT to insert and WHERE
- **LLM** generates the code content
- **Text insertion** puts it in the file
- **LSP** validates, suggests fixes, and formats
- **GOFAI** verifies the result

This is **text insertion + LSP validation**, not **LSP semantic insertion**. But it's still far superior to naive text manipulation because LSP catches and fixes our mistakes.
