package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// persistCmd represents the persist command
var persistCmd = &cobra.Command{
	Use:   "persist",
	Short: "Manage persisted files across worktrees",
	Long: `Persist files and directories to share them across all worktrees.
Files are stored in the shared/ directory in the bare repository root.

Available subcommands:
  add     - Persist a file or directory
  list    - List all persisted files
  remove  - Remove a persisted file or directory`,
}

var persistAddCmd = &cobra.Command{
	Use:   "add <file|dir>",
	Short: "Persist a file or directory to shared storage",
	Long: `Copy a file or directory from the current worktree to shared storage.
The path structure is preserved, so the file can be restored to the same location.

Example:
  wtm persist add .env
  wtm persist add src/config.json
  wtm persist add node_modules`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetPath := args[0]

		// Find bare repo root and current worktree root
		bareRepoRoot, worktreeRoot, err := findRepoRoots()
		if err != nil {
			return err
		}

		// Get absolute path of target file
		absTargetPath := targetPath
		if !filepath.IsAbs(targetPath) {
			absTargetPath = filepath.Join(worktreeRoot, targetPath)
		}

		// Check if target exists
		if _, err := os.Stat(absTargetPath); os.IsNotExist(err) {
			return fmt.Errorf("file or directory does not exist: %s", targetPath)
		}

		// Get relative path from worktree root to preserve structure
		relPath, err := filepath.Rel(worktreeRoot, absTargetPath)
		if err != nil {
			return fmt.Errorf("error determining relative path: %w", err)
		}

		// Create destination path in shared/
		sharedDir := filepath.Join(bareRepoRoot, "shared")
		destPath := filepath.Join(sharedDir, relPath)

		// Check if already exists in shared
		if _, err := os.Stat(destPath); err == nil {
			return fmt.Errorf("file already exists in shared storage: %s\nUse 'wtm persist remove %s' first if you want to update it", relPath, relPath)
		}

		// Create parent directories in shared/
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("error creating shared directory structure: %w", err)
		}

		fmt.Printf("Persisting %s to shared/%s...\n", targetPath, relPath)

		// Copy file or directory
		if err := copyPath(absTargetPath, destPath); err != nil {
			return fmt.Errorf("error copying to shared storage: %w", err)
		}

		fmt.Printf("Successfully persisted to shared/%s\n", relPath)
		return nil
	},
}

var persistListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all persisted files and directories",
	Long:  `Display all files and directories stored in shared storage.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		bareRepoRoot, _, err := findRepoRoots()
		if err != nil {
			return err
		}

		sharedDir := filepath.Join(bareRepoRoot, "shared")

		// Check if shared directory exists
		if _, err := os.Stat(sharedDir); os.IsNotExist(err) {
			fmt.Println("No persisted files yet. Use 'wtm persist add <file>' to persist files.")
			return nil
		}

		fmt.Println("Persisted files in shared/:")
		fmt.Println()

		// Walk through shared directory
		count := 0
		err = filepath.Walk(sharedDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip the shared directory itself
			if path == sharedDir {
				return nil
			}

			relPath, _ := filepath.Rel(sharedDir, path)

			if info.IsDir() {
				fmt.Printf("  📁 %s/\n", relPath)
			} else {
				size := formatSize(info.Size())
				fmt.Printf("  📄 %s (%s)\n", relPath, size)
			}
			count++

			return nil
		})

		if err != nil {
			return fmt.Errorf("error listing shared files: %w", err)
		}

		if count == 0 {
			fmt.Println("  (empty)")
		}

		return nil
	},
}

var persistRemoveCmd = &cobra.Command{
	Use:   "remove <file|dir>",
	Short: "Remove a persisted file or directory",
	Long: `Remove a file or directory from shared storage.

Example:
  wtm persist remove .env
  wtm persist remove src/config.json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetPath := args[0]

		bareRepoRoot, _, err := findRepoRoots()
		if err != nil {
			return err
		}

		sharedDir := filepath.Join(bareRepoRoot, "shared")
		targetFullPath := filepath.Join(sharedDir, targetPath)

		// Check if file exists
		if _, err := os.Stat(targetFullPath); os.IsNotExist(err) {
			return fmt.Errorf("file not found in shared storage: %s", targetPath)
		}

		fmt.Printf("Removing %s from shared storage...\n", targetPath)

		// Remove file or directory
		if err := os.RemoveAll(targetFullPath); err != nil {
			return fmt.Errorf("error removing from shared storage: %w", err)
		}

		fmt.Printf("Successfully removed shared/%s\n", targetPath)
		return nil
	},
}

// findRepoRoots finds the bare repo root and current worktree root
func findRepoRoots() (bareRepoRoot string, worktreeRoot string, err error) {
	// Check if we're in a git repository
	gitDirCmd := []string{"git", "rev-parse", "--git-dir"}
	gitDir, err := runCommand(gitDirCmd...)
	if err != nil {
		return "", "", fmt.Errorf("not in a git repository")
	}

	gitDir = strings.TrimSpace(gitDir)

	// Check if we're in a bare repo
	isBareCmd := []string{"git", "rev-parse", "--is-bare-repository"}
	isBare, _ := runCommand(isBareCmd...)

	if strings.TrimSpace(isBare) == "true" {
		return "", "", fmt.Errorf("cannot run persist from bare repository. Please run from a worktree")
	}

	// Get worktree root
	worktreeRootCmd := []string{"git", "rev-parse", "--show-toplevel"}
	worktreeRoot, err = runCommand(worktreeRootCmd...)
	if err != nil {
		return "", "", fmt.Errorf("error getting worktree root: %w", err)
	}
	worktreeRoot = strings.TrimSpace(worktreeRoot)

	// Find bare repo root from git dir
	if strings.Contains(gitDir, "worktrees") {
		idx := strings.LastIndex(gitDir, "/worktrees/")
		if idx == -1 {
			idx = strings.LastIndex(gitDir, "\\worktrees\\")
		}
		if idx == -1 {
			return "", "", fmt.Errorf("unexpected git directory format")
		}
		bareRepoRoot = gitDir[:idx]
	} else {
		return "", "", fmt.Errorf("not in a worktree")
	}

	return bareRepoRoot, worktreeRoot, nil
}

// copyPath copies a file or directory from src to dst
func copyPath(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Preserve permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// formatSize formats bytes into human-readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// runCommand runs a command and returns the output
func runCommand(args ...string) (string, error) {
	cmd := args[0]
	cmdArgs := args[1:]

	execCmd := exec.Command(cmd, cmdArgs...)
	output, err := execCmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func init() {
	rootCmd.AddCommand(persistCmd)
	persistCmd.AddCommand(persistAddCmd)
	persistCmd.AddCommand(persistListCmd)
	persistCmd.AddCommand(persistRemoveCmd)
}
