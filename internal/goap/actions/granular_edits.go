package actions

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"upside-down-research.com/oss/agentic/internal/goap"
)

// === WHOLESALE EDITS ===

// WholesaleFileReplaceAction replaces entire file content
type WholesaleFileReplaceAction struct {
	*goap.BaseAction
	filePath   string
	newContent string
}

func NewWholesaleFileReplaceAction(filePath, newContent string) *WholesaleFileReplaceAction {
	return &WholesaleFileReplaceAction{
		BaseAction: goap.NewBaseAction(
			"WholesaleReplace",
			fmt.Sprintf("Replace entire file: %s", filePath),
			goap.WorldState{},
			goap.WorldState{"file_replaced": true},
			2.0,
		),
		filePath:   filePath,
		newContent: newContent,
	}
}

func (a *WholesaleFileReplaceAction) Execute(ctx context.Context, current goap.WorldState) error {
	log.Info("Wholesale file replacement", "file", a.filePath)

	err := os.WriteFile(a.filePath, []byte(a.newContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to replace file: %w", err)
	}

	current.Set("file_replaced", true)
	current.Set("replaced_file", a.filePath)

	log.Info("File replaced successfully")
	return nil
}

func (a *WholesaleFileReplaceAction) Clone() goap.Action {
	return NewWholesaleFileReplaceAction(a.filePath, a.newContent)
}

// === PARTIAL EDITS (Block-based) ===

// PartialBlockEditAction edits a block/section of a file
type PartialBlockEditAction struct {
	*goap.BaseAction
	filePath    string
	startMarker string
	endMarker   string
	newContent  string
}

func NewPartialBlockEditAction(filePath, startMarker, endMarker, newContent string) *PartialBlockEditAction {
	return &PartialBlockEditAction{
		BaseAction: goap.NewBaseAction(
			"PartialBlockEdit",
			fmt.Sprintf("Edit block in %s between markers", filePath),
			goap.WorldState{"file_exists": true},
			goap.WorldState{"block_edited": true},
			3.0,
		),
		filePath:    filePath,
		startMarker: startMarker,
		endMarker:   endMarker,
		newContent:  newContent,
	}
}

func (a *PartialBlockEditAction) Execute(ctx context.Context, current goap.WorldState) error {
	log.Info("Partial block edit", "file", a.filePath, "start", a.startMarker, "end", a.endMarker)

	content, err := os.ReadFile(a.filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	text := string(content)

	// Find block boundaries
	startIdx := strings.Index(text, a.startMarker)
	if startIdx == -1 {
		return fmt.Errorf("start marker not found: %s", a.startMarker)
	}

	endIdx := strings.Index(text[startIdx:], a.endMarker)
	if endIdx == -1 {
		return fmt.Errorf("end marker not found: %s", a.endMarker)
	}
	endIdx += startIdx + len(a.endMarker)

	// Replace block
	result := text[:startIdx+len(a.startMarker)] + "\n" + a.newContent + "\n" + text[endIdx:]

	err = os.WriteFile(a.filePath, []byte(result), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	current.Set("block_edited", true)
	current.Set("edited_file", a.filePath)

	log.Info("Block edited successfully")
	return nil
}

func (a *PartialBlockEditAction) Clone() goap.Action {
	return NewPartialBlockEditAction(a.filePath, a.startMarker, a.endMarker, a.newContent)
}

// === LINE-BASED EDITS ===

// LineBasedEditAction edits specific lines in a file
type LineBasedEditAction struct {
	*goap.BaseAction
	filePath string
	edits    []LineEdit
}

type LineEdit struct {
	LineNumber  int    // 1-indexed line number
	Operation   string // "replace", "insert_after", "insert_before", "delete"
	NewContent  string // Content for insert/replace operations
}

func NewLineBasedEditAction(filePath string, edits []LineEdit) *LineBasedEditAction {
	return &LineBasedEditAction{
		BaseAction: goap.NewBaseAction(
			"LineBasedEdit",
			fmt.Sprintf("Edit %d lines in %s", len(edits), filePath),
			goap.WorldState{"file_exists": true},
			goap.WorldState{"lines_edited": true},
			4.0,
		),
		filePath: filePath,
		edits:    edits,
	}
}

func (a *LineBasedEditAction) Execute(ctx context.Context, current goap.WorldState) error {
	log.Info("Line-based edit", "file", a.filePath, "edits", len(a.edits))

	file, err := os.Open(a.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read all lines
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Apply edits (sort by line number descending to avoid offset issues)
	// For simplicity, we'll process in order given
	for _, edit := range a.edits {
		lineIdx := edit.LineNumber - 1 // Convert to 0-indexed

		if lineIdx < 0 || lineIdx > len(lines) {
			return fmt.Errorf("invalid line number: %d", edit.LineNumber)
		}

		switch edit.Operation {
		case "replace":
			if lineIdx < len(lines) {
				lines[lineIdx] = edit.NewContent
			}

		case "insert_after":
			if lineIdx < len(lines) {
				lines = append(lines[:lineIdx+1], append([]string{edit.NewContent}, lines[lineIdx+1:]...)...)
			} else {
				lines = append(lines, edit.NewContent)
			}

		case "insert_before":
			lines = append(lines[:lineIdx], append([]string{edit.NewContent}, lines[lineIdx:]...)...)

		case "delete":
			if lineIdx < len(lines) {
				lines = append(lines[:lineIdx], lines[lineIdx+1:]...)
			}

		default:
			return fmt.Errorf("unknown operation: %s", edit.Operation)
		}

		log.Debug("Applied line edit", "line", edit.LineNumber, "op", edit.Operation)
	}

	// Write back
	result := strings.Join(lines, "\n") + "\n"
	err = os.WriteFile(a.filePath, []byte(result), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	current.Set("lines_edited", true)
	current.Set("edited_file", a.filePath)

	log.Info("Lines edited successfully", "total", len(lines))
	return nil
}

func (a *LineBasedEditAction) Clone() goap.Action {
	return NewLineBasedEditAction(a.filePath, a.edits)
}

// === CHARACTER-BASED EDITS (Precise) ===

// CharacterBasedEditAction performs precise character-level edits
type CharacterBasedEditAction struct {
	*goap.BaseAction
	filePath string
	edits    []CharEdit
}

type CharEdit struct {
	Offset    int    // Character offset in file
	Length    int    // Number of characters to replace (0 for insert)
	NewText   string // New text to insert
}

func NewCharacterBasedEditAction(filePath string, edits []CharEdit) *CharacterBasedEditAction {
	return &CharacterBasedEditAction{
		BaseAction: goap.NewBaseAction(
			"CharacterBasedEdit",
			fmt.Sprintf("Character-level edit of %s (%d edits)", filePath, len(edits)),
			goap.WorldState{"file_exists": true},
			goap.WorldState{"chars_edited": true},
			5.0, // Most precise, highest complexity
		),
		filePath: filePath,
		edits:    edits,
	}
}

func (a *CharacterBasedEditAction) Execute(ctx context.Context, current goap.WorldState) error {
	log.Info("Character-based edit", "file", a.filePath, "edits", len(a.edits))

	content, err := os.ReadFile(a.filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	text := string(content)

	// Sort edits by offset (descending) to avoid offset corruption
	sortedEdits := make([]CharEdit, len(a.edits))
	copy(sortedEdits, a.edits)

	// Simple bubble sort for descending offset
	for i := 0; i < len(sortedEdits)-1; i++ {
		for j := 0; j < len(sortedEdits)-i-1; j++ {
			if sortedEdits[j].Offset < sortedEdits[j+1].Offset {
				sortedEdits[j], sortedEdits[j+1] = sortedEdits[j+1], sortedEdits[j]
			}
		}
	}

	// Apply edits from end to beginning
	for _, edit := range sortedEdits {
		if edit.Offset < 0 || edit.Offset > len(text) {
			return fmt.Errorf("invalid offset: %d", edit.Offset)
		}

		if edit.Offset+edit.Length > len(text) {
			return fmt.Errorf("edit extends beyond file: offset=%d, length=%d, filesize=%d",
				edit.Offset, edit.Length, len(text))
		}

		// Apply edit
		text = text[:edit.Offset] + edit.NewText + text[edit.Offset+edit.Length:]

		log.Debug("Applied char edit", "offset", edit.Offset, "length", edit.Length, "newLen", len(edit.NewText))
	}

	// Write back
	err = os.WriteFile(a.filePath, []byte(text), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	current.Set("chars_edited", true)
	current.Set("edited_file", a.filePath)

	log.Info("Character-level edits applied successfully")
	return nil
}

func (a *CharacterBasedEditAction) Clone() goap.Action {
	return NewCharacterBasedEditAction(a.filePath, a.edits)
}

// === RANGE-BASED EDITS (Line:Column to Line:Column) ===

// RangeEditAction edits a specific range in the file
type RangeEditAction struct {
	*goap.BaseAction
	filePath string
	start    Position
	end      Position
	newText  string
}

type Position struct {
	Line   int // 1-indexed
	Column int // 1-indexed
}

func NewRangeEditAction(filePath string, start, end Position, newText string) *RangeEditAction {
	return &RangeEditAction{
		BaseAction: goap.NewBaseAction(
			"RangeEdit",
			fmt.Sprintf("Edit range in %s (%d:%d to %d:%d)", filePath, start.Line, start.Column, end.Line, end.Column),
			goap.WorldState{"file_exists": true},
			goap.WorldState{"range_edited": true},
			4.0,
		),
		filePath: filePath,
		start:    start,
		end:      end,
		newText:  newText,
	}
}

func (a *RangeEditAction) Execute(ctx context.Context, current goap.WorldState) error {
	log.Info("Range-based edit", "file", a.filePath,
		"start", fmt.Sprintf("%d:%d", a.start.Line, a.start.Column),
		"end", fmt.Sprintf("%d:%d", a.end.Line, a.end.Column))

	file, err := os.Open(a.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read lines
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Validate positions
	if a.start.Line < 1 || a.start.Line > len(lines) {
		return fmt.Errorf("invalid start line: %d", a.start.Line)
	}
	if a.end.Line < 1 || a.end.Line > len(lines) {
		return fmt.Errorf("invalid end line: %d", a.end.Line)
	}

	startLine := a.start.Line - 1 // Convert to 0-indexed
	endLine := a.end.Line - 1
	startCol := a.start.Column - 1
	endCol := a.end.Column - 1

	// Handle single line case
	if startLine == endLine {
		line := lines[startLine]
		if startCol < 0 || startCol > len(line) || endCol < 0 || endCol > len(line) {
			return fmt.Errorf("invalid column range")
		}

		lines[startLine] = line[:startCol] + a.newText + line[endCol:]
	} else {
		// Multi-line case
		startLineContent := lines[startLine][:startCol]
		endLineContent := lines[endLine][endCol:]

		// Replace with new content
		newLines := []string{startLineContent + a.newText + endLineContent}
		lines = append(lines[:startLine], append(newLines, lines[endLine+1:]...)...)
	}

	// Write back
	result := strings.Join(lines, "\n") + "\n"
	err = os.WriteFile(a.filePath, []byte(result), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	current.Set("range_edited", true)
	current.Set("edited_file", a.filePath)

	log.Info("Range edited successfully")
	return nil
}

func (a *RangeEditAction) Clone() goap.Action {
	return NewRangeEditAction(a.filePath, a.start, a.end, a.newText)
}
