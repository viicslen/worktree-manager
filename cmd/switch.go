package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// switchCmd represents the switch command
var switchCmd = &cobra.Command{
	Use:   "switch <branch|commit>",
	Short: "Switch the active workspace to a different branch or commit",
	Long: `Switch the active workspace to a different branch or commit.

The workspace directory is where your IDE should be open. This command allows you
to switch between branches without closing your IDE or changing directories.

The current workspace will be moved to tree/<current-branch> and the target
branch will be moved from tree/<target> to workspace (or created if it doesn't exist).

Can be run from:
  - The bare repository root
  - The workspace directory

Cannot be run from:
  - Inside tree/<branch> subdirectories

Example:
  wtm switch develop        # Switch to develop branch
  wtm switch feature/new    # Switch to feature/new branch
  wtm switch abc123         # Switch to commit abc123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetCommitish := args[0]

		// Detect our current location and find bare repo root
		bareRepoRoot, currentLocation, err := detectLocation()
		if err != nil {
			return err
		}

		// Change to bare repo root to ensure all git operations work correctly
		// This is especially important when running from within workspace
		if err := os.Chdir(bareRepoRoot); err != nil {
			return fmt.Errorf("error changing to bare repo directory: %w", err)
		}

		workspacePath := filepath.Join(bareRepoRoot, "workspace")
		treeDir := filepath.Join(bareRepoRoot, "tree")

		// Get current workspace info if it exists
		var currentBranch string
		workspaceExists := false
		if _, err := os.Stat(workspacePath); err == nil {
			workspaceExists = true
			currentBranch, err = getCurrentBranch(workspacePath)
			if err != nil {
				return fmt.Errorf("error getting current branch from workspace: %w", err)
			}
		}

		// Create tree directory if it doesn't exist (needed before any moves)
		if err := os.MkdirAll(treeDir, 0755); err != nil {
			return fmt.Errorf("error creating tree directory: %w", err)
		}

		// If workspace exists, we need to move it to tree/<current-branch>
		if workspaceExists {
			if currentBranch == "" {
				return fmt.Errorf("workspace exists but is in detached HEAD state")
			}

			// Sanitize branch name for directory
			sanitizedCurrent := sanitizeBranchName(currentBranch)
			currentTargetPath := filepath.Join(treeDir, sanitizedCurrent)

			// Check if target already exists (shouldn't happen with git constraints)
			if _, err := os.Stat(currentTargetPath); err == nil {
				return fmt.Errorf("tree/%s already exists, this shouldn't happen", sanitizedCurrent)
			}

			fmt.Printf("Moving current workspace (%s) to tree/%s...\n", currentBranch, sanitizedCurrent)

			// Move workspace to tree/<current-branch>
			if err := moveWorktree(workspacePath, currentTargetPath, bareRepoRoot); err != nil {
				return fmt.Errorf("error moving workspace to tree: %w", err)
			}
		}

		// Now handle the target branch/commit
		sanitizedTarget := sanitizeBranchName(targetCommitish)
		targetTreePath := filepath.Join(treeDir, sanitizedTarget)

		// Check if target exists in tree/
		if _, err := os.Stat(targetTreePath); err == nil {
			// Move from tree to workspace
			fmt.Printf("Moving tree/%s to workspace...\n", sanitizedTarget)
			if err := moveWorktree(targetTreePath, workspacePath, bareRepoRoot); err != nil {
				return fmt.Errorf("error moving tree/%s to workspace: %w", sanitizedTarget, err)
			}
		} else {
			// Create new worktree at workspace
			fmt.Printf("Creating new worktree for '%s' at workspace...\n", targetCommitish)
			if err := createWorktree(workspacePath, targetCommitish, bareRepoRoot); err != nil {
				return fmt.Errorf("error creating worktree: %w", err)
			}
		}

		fmt.Printf("Successfully switched to %s\n", targetCommitish)
		if currentLocation == "workspace" {
			fmt.Println("Note: You may need to reload files in your IDE to see the changes")
		}

		return nil
	},
}

// detectLocation determines where the command is being run from and finds the bare repo root
func detectLocation() (bareRepoRoot string, location string, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("error getting current directory: %w", err)
	}

	// Check if we're in a git worktree
	gitDirCmd := exec.Command("git", "rev-parse", "--git-dir")
	gitDirOutput, err := gitDirCmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("not in a git repository")
	}

	gitDir := strings.TrimSpace(string(gitDirOutput))

	// Check if current directory is a bare repo
	isBareCmd := exec.Command("git", "rev-parse", "--is-bare-repository")
	isBareOutput, _ := isBareCmd.Output()
	isBare := strings.TrimSpace(string(isBareOutput)) == "true"

	if isBare {
		// We're in the bare repo root
		return cwd, "bare-root", nil
	}

	// We're in a worktree, find the bare repo
	// Git dir in a worktree points to either:
	// - /path/to/bare/.git/worktrees/<name> (non-bare repo with worktrees)
	// - /path/to/bare/worktrees/<name> (bare repo with worktrees)
	if strings.Contains(gitDir, "worktrees") {
		// Parse the git dir to find bare repo
		// Use LastIndex to handle cases where "worktrees" appears in the path
		idx := strings.LastIndex(gitDir, "/worktrees/")
		if idx == -1 {
			idx = strings.LastIndex(gitDir, "\\worktrees\\") // Windows path
		}
		if idx == -1 {
			return "", "", fmt.Errorf("unexpected git directory format")
		}
		bareRepoRoot = gitDir[:idx]

		// Check if we're inside tree/ subdirectory
		relPath, err := filepath.Rel(bareRepoRoot, cwd)
		if err != nil {
			return "", "", fmt.Errorf("error determining relative path: %w", err)
		}

		if strings.HasPrefix(relPath, "tree/") || strings.HasPrefix(relPath, "tree\\") {
			return "", "", fmt.Errorf("cannot run switch from inside tree/<branch> directory. Please run from bare repo root or workspace")
		}

		// Check if we're in workspace
		if strings.HasPrefix(relPath, "workspace") {
			return bareRepoRoot, "workspace", nil
		}

		return bareRepoRoot, "worktree", nil
	}

	return "", "", fmt.Errorf("not in a bare repository or valid worktree")
}

// getCurrentBranch returns the current branch name from a worktree
func getCurrentBranch(worktreePath string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = worktreePath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// sanitizeBranchName converts branch names to valid directory names
func sanitizeBranchName(name string) string {
	result := ""
	for _, char := range name {
		if char == '/' {
			result += "-"
		} else {
			result += string(char)
		}
	}
	return result
}

// moveWorktree moves a worktree from source to destination
func moveWorktree(source, destination, bareRepoRoot string) error {
	// Try using git worktree move first
	moveCmd := exec.Command("git", "worktree", "move", source, destination)
	moveCmd.Dir = bareRepoRoot
	moveCmd.Env = append(os.Environ(), "GIT_DIR="+bareRepoRoot)
	output, err := moveCmd.CombinedOutput()

	if err != nil {
		// If git worktree move fails, try filesystem move + repair
		fmt.Printf("git worktree move failed (error: %v, output: %s), trying filesystem move...\n", err, string(output))
		if err := os.Rename(source, destination); err != nil {
			return fmt.Errorf("failed to move directory: %w", err)
		}

		// Repair git worktree metadata
		repairCmd := exec.Command("git", "worktree", "repair")
		repairCmd.Dir = bareRepoRoot
		repairCmd.Env = append(os.Environ(), "GIT_DIR="+bareRepoRoot)
		if err := repairCmd.Run(); err != nil {
			return fmt.Errorf("moved directory but failed to repair git metadata: %w", err)
		}
	}

	return nil
}

// createWorktree creates a new worktree at the specified path
func createWorktree(path, commitish, bareRepoRoot string) error {
	addCmd := exec.Command("git", "worktree", "add", path, commitish)
	addCmd.Dir = bareRepoRoot
	addCmd.Env = append(os.Environ(), "GIT_DIR="+bareRepoRoot)
	addCmd.Stdout = os.Stdout
	addCmd.Stderr = os.Stderr
	return addCmd.Run()
}

func init() {
	rootCmd.AddCommand(switchCmd)
}
