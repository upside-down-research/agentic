package actions

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/log"
	"upside-down-research.com/oss/agentic/internal/goap"
)

// GitStatusAction checks the current git status
type GitStatusAction struct {
	*goap.BaseAction
	workDir string
}

func NewGitStatusAction(workDir string) *GitStatusAction {
	return &GitStatusAction{
		BaseAction: goap.NewBaseAction(
			"GitStatus",
			"Check git repository status",
			goap.WorldState{},
			goap.WorldState{"git_status_checked": true},
			1.0, // Low complexity
		),
		workDir: workDir,
	}
}

func (a *GitStatusAction) Execute(ctx context.Context, current goap.WorldState) error {
	log.Info("Checking git status", "workDir", a.workDir)

	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = a.workDir

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}

	current.Set("git_status_checked", true)
	current.Set("git_status_output", string(output))
	current.Set("git_has_changes", len(output) > 0)

	log.Info("Git status complete", "hasChanges", len(output) > 0)
	return nil
}

func (a *GitStatusAction) Clone() goap.Action {
	return NewGitStatusAction(a.workDir)
}

// GitAddAction stages files for commit
type GitAddAction struct {
	*goap.BaseAction
	workDir string
	paths   []string
}

func NewGitAddAction(workDir string, paths []string) *GitAddAction {
	return &GitAddAction{
		BaseAction: goap.NewBaseAction(
			"GitAdd",
			fmt.Sprintf("Stage files for commit: %v", paths),
			goap.WorldState{"git_status_checked": true},
			goap.WorldState{"files_staged": true},
			2.0,
		),
		workDir: workDir,
		paths:   paths,
	}
}

func (a *GitAddAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for GitAdd")
	}

	log.Info("Staging files", "paths", a.paths)

	args := append([]string{"add"}, a.paths...)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = a.workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add failed: %w\nOutput: %s", err, output)
	}

	current.Set("files_staged", true)
	log.Info("Files staged successfully")
	return nil
}

func (a *GitAddAction) Clone() goap.Action {
	return NewGitAddAction(a.workDir, a.paths)
}

// GitCommitAction creates a commit
type GitCommitAction struct {
	*goap.BaseAction
	workDir string
	message string
}

func NewGitCommitAction(workDir, message string) *GitCommitAction {
	return &GitCommitAction{
		BaseAction: goap.NewBaseAction(
			"GitCommit",
			"Create git commit",
			goap.WorldState{"files_staged": true},
			goap.WorldState{"changes_committed": true},
			3.0,
		),
		workDir: workDir,
		message: message,
	}
}

func (a *GitCommitAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for GitCommit")
	}

	log.Info("Creating commit", "message", a.message)

	cmd := exec.CommandContext(ctx, "git", "commit", "-m", a.message)
	cmd.Dir = a.workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed: %w\nOutput: %s", err, output)
	}

	current.Set("changes_committed", true)
	current.Set("commit_hash", extractCommitHash(string(output)))

	log.Info("Commit created successfully")
	return nil
}

func (a *GitCommitAction) Clone() goap.Action {
	return NewGitCommitAction(a.workDir, a.message)
}

// GitPushAction pushes commits to remote
type GitPushAction struct {
	*goap.BaseAction
	workDir string
	branch  string
}

func NewGitPushAction(workDir, branch string) *GitPushAction {
	return &GitPushAction{
		BaseAction: goap.NewBaseAction(
			"GitPush",
			fmt.Sprintf("Push to branch: %s", branch),
			goap.WorldState{"changes_committed": true},
			goap.WorldState{"changes_pushed": true},
			5.0, // Medium complexity (network operation)
		),
		workDir: workDir,
		branch:  branch,
	}
}

func (a *GitPushAction) Execute(ctx context.Context, current goap.WorldState) error {
	if !a.CanExecute(current) {
		return fmt.Errorf("preconditions not met for GitPush")
	}

	log.Info("Pushing to remote", "branch", a.branch)

	cmd := exec.CommandContext(ctx, "git", "push", "-u", "origin", a.branch)
	cmd.Dir = a.workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push failed: %w\nOutput: %s", err, output)
	}

	current.Set("changes_pushed", true)
	log.Info("Push successful", "branch", a.branch)
	return nil
}

func (a *GitPushAction) Clone() goap.Action {
	return NewGitPushAction(a.workDir, a.branch)
}

// GitBranchAction creates a new branch
type GitBranchAction struct {
	*goap.BaseAction
	workDir    string
	branchName string
}

func NewGitBranchAction(workDir, branchName string) *GitBranchAction {
	return &GitBranchAction{
		BaseAction: goap.NewBaseAction(
			"GitBranch",
			fmt.Sprintf("Create and checkout branch: %s", branchName),
			goap.WorldState{},
			goap.WorldState{"branch_created": true, "current_branch": branchName},
			2.0,
		),
		workDir:    workDir,
		branchName: branchName,
	}
}

func (a *GitBranchAction) Execute(ctx context.Context, current goap.WorldState) error {
	log.Info("Creating branch", "name", a.branchName)

	// Create and checkout branch
	cmd := exec.CommandContext(ctx, "git", "checkout", "-b", a.branchName)
	cmd.Dir = a.workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Branch might already exist, try to checkout
		cmd = exec.CommandContext(ctx, "git", "checkout", a.branchName)
		cmd.Dir = a.workDir
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git branch failed: %w\nOutput: %s", err, output)
		}
	}

	current.Set("branch_created", true)
	current.Set("current_branch", a.branchName)

	log.Info("Branch ready", "name", a.branchName)
	return nil
}

func (a *GitBranchAction) Clone() goap.Action {
	return NewGitBranchAction(a.workDir, a.branchName)
}

// Helper function to extract commit hash from git commit output
func extractCommitHash(output string) string {
	// Look for pattern like "[branch abcd123]"
	parts := strings.Fields(output)
	for i, part := range parts {
		if strings.HasPrefix(part, "[") && i+1 < len(parts) {
			hash := strings.TrimSuffix(parts[i+1], "]")
			return hash
		}
	}
	return "unknown"
}
