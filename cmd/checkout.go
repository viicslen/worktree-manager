package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

// checkoutCmd represents the checkout command
var checkoutCmd = &cobra.Command{
	Use:   "checkout <commitish>",
	Short: "Checkout a branch or commit to a worktree in tree/<commitish>",
	Long: `Create a new git worktree for the specified branch or commit.
The worktree will be created in the 'tree' directory with the name of the commitish.

Example:
  wtm checkout main          # Creates tree/main
  wtm checkout feature/new   # Creates tree/feature-new
  wtm checkout abc123        # Creates tree/abc123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		commitish := args[0]

		// Get the current directory (should be the bare repo)
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting current directory: %w", err)
		}

		// Check if we're in a bare repository
		checkBare := exec.Command("git", "rev-parse", "--is-bare-repository")
		output, err := checkBare.Output()
		if err != nil || string(output) != "true\n" {
			return fmt.Errorf("not in a bare repository. Please run this command from a bare repository")
		}

		// Sanitize the commitish for use as a directory name
		// Replace slashes with dashes to handle branch names like "feature/new"
		dirName := filepath.Base(commitish)
		if commitish != dirName {
			// If commitish contains slashes, replace them with dashes
			dirName = ""
			for _, char := range commitish {
				if char == '/' {
					dirName += "-"
				} else {
					dirName += string(char)
				}
			}
		}

		// Create the tree directory if it doesn't exist
		treeDir := filepath.Join(cwd, "tree")
		if err := os.MkdirAll(treeDir, 0755); err != nil {
			return fmt.Errorf("error creating tree directory: %w", err)
		}

		// Full path for the new worktree
		worktreePath := filepath.Join(treeDir, dirName)

		// Check if worktree already exists
		if _, err := os.Stat(worktreePath); err == nil {
			return fmt.Errorf("worktree already exists at %s", worktreePath)
		}

		fmt.Printf("Creating worktree for '%s' at %s...\n", commitish, worktreePath)

		// Create the worktree
		addWorktree := exec.Command("git", "worktree", "add", worktreePath, commitish)
		addWorktree.Stdout = os.Stdout
		addWorktree.Stderr = os.Stderr
		if err := addWorktree.Run(); err != nil {
			return fmt.Errorf("error creating worktree: %w", err)
		}

		fmt.Printf("Successfully created worktree at %s\n", worktreePath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkoutCmd)
}
