package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// cloneCmd represents the clone command
var cloneCmd = &cobra.Command{
	Use:   "clone <repo-url> [directory]",
	Short: "Clone a repository as a bare repository for worktree usage",
	Long: `Clone a repository as a bare repository and configure the fetch refspec
to fetch all remote branches. This setup is ideal for a worktree-based workflow.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoURL := args[0]
		var dir string
		if len(args) == 2 {
			dir = args[1]
		} else {
			// Infer directory from repo URL
			parts := strings.Split(repoURL, "/")
			lastPart := parts[len(parts)-1]
			dir = strings.TrimSuffix(lastPart, ".git")
		}

		fmt.Printf("Cloning %s into %s...\n", repoURL, dir)

		// 1. git clone --bare --recurse-submodules <repo-url> <directory>
		clone := exec.Command("git", "clone", "--bare", "--recurse-submodules", repoURL, dir)
		clone.Stdout = os.Stdout
		clone.Stderr = os.Stderr
		if err := clone.Run(); err != nil {
			return fmt.Errorf("error cloning repository: %w", err)
		}

		// 2. Change directory to <directory>
		// We can't change the process's working directory permanently for the user,
		// but we can run subsequent commands in that directory.

		// 3. git config --add remote.origin.fetch "+refs/heads/*:refs/remotes/origin/*"
		configCmd := exec.Command("git", "config", "--add", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
		configCmd.Dir = dir
		configCmd.Stdout = os.Stdout
		configCmd.Stderr = os.Stderr
		if err := configCmd.Run(); err != nil {
			return fmt.Errorf("error configuring remote fetch: %w", err)
		}

		fmt.Println("Repository cloned and configured successfully.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cloneCmd)
}
