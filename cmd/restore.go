package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	restoreLink  bool
	restoreTo    string
	restoreForce bool
	restoreAll   bool
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore <file|dir>",
	Short: "Restore persisted files to the current worktree",
	Long: `Copy or link a persisted file/directory from shared storage to the current worktree.
By default, files are copied. Use --link to create a symlink instead.

The file is restored to its original relative path unless --to is specified.

Examples:
  wtm restore .env                    # Copy .env to current worktree
  wtm restore node_modules --link     # Symlink node_modules (saves space)
  wtm restore config.json --to custom/path/config.json
  wtm restore .env --force            # Overwrite if exists
  wtm restore --all                   # Restore all persisted files`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate arguments
		if restoreAll && len(args) > 0 {
			return fmt.Errorf("cannot specify file with --all flag")
		}
		if !restoreAll && len(args) == 0 {
			return fmt.Errorf("must specify a file/directory or use --all flag")
		}

		bareRepoRoot, worktreeRoot, err := findRepoRoots()
		if err != nil {
			return err
		}

		sharedDir := filepath.Join(bareRepoRoot, "shared")

		// Check if shared directory exists
		if _, err := os.Stat(sharedDir); os.IsNotExist(err) {
			return fmt.Errorf("no persisted files found. Use 'wtm persist add <file>' to persist files first")
		}

		if restoreAll {
			return restoreAllFiles(sharedDir, worktreeRoot)
		}

		targetPath := args[0]
		return restoreFile(targetPath, sharedDir, worktreeRoot)
	},
}

func restoreFile(targetPath, sharedDir, worktreeRoot string) error {
	sourcePath := filepath.Join(sharedDir, targetPath)

	// Check if file exists in shared
	sourceInfo, err := os.Stat(sourcePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("file not found in shared storage: %s\nUse 'wtm persist list' to see available files", targetPath)
	}
	if err != nil {
		return fmt.Errorf("error accessing shared file: %w", err)
	}

	// Determine destination path
	var destPath string
	if restoreTo != "" {
		destPath = restoreTo
		if !filepath.IsAbs(destPath) {
			destPath = filepath.Join(worktreeRoot, destPath)
		}
	} else {
		// Restore to same relative path
		destPath = filepath.Join(worktreeRoot, targetPath)
	}

	// Check if destination exists
	if _, err := os.Stat(destPath); err == nil && !restoreForce {
		return fmt.Errorf("file already exists: %s\nUse --force to overwrite", destPath)
	}

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("error creating parent directories: %w", err)
	}

	action := "Copying"
	if restoreLink {
		action = "Linking"
	}

	relDestPath, _ := filepath.Rel(worktreeRoot, destPath)
	fmt.Printf("%s shared/%s to %s...\n", action, targetPath, relDestPath)

	if restoreLink {
		// Remove existing file if --force
		if restoreForce {
			os.RemoveAll(destPath)
		}

		// Create symlink (use relative path for portability)
		relLink, err := filepath.Rel(filepath.Dir(destPath), sourcePath)
		if err != nil {
			return fmt.Errorf("error calculating relative link path: %w", err)
		}

		if err := os.Symlink(relLink, destPath); err != nil {
			return fmt.Errorf("error creating symlink: %w", err)
		}
	} else {
		// Remove existing file if --force
		if restoreForce {
			os.RemoveAll(destPath)
		}

		// Copy file or directory
		if sourceInfo.IsDir() {
			if err := copyDir(sourcePath, destPath); err != nil {
				return fmt.Errorf("error copying directory: %w", err)
			}
		} else {
			if err := copyFile(sourcePath, destPath); err != nil {
				return fmt.Errorf("error copying file: %w", err)
			}
		}
	}

	fmt.Printf("Successfully restored to %s\n", relDestPath)
	return nil
}

func restoreAllFiles(sharedDir, worktreeRoot string) error {
	fmt.Println("Restoring all persisted files...")

	count := 0
	errors := []string{}

	// Walk through shared directory
	err := filepath.Walk(sharedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the shared directory itself
		if path == sharedDir {
			return nil
		}

		// Get relative path from shared directory
		relPath, err := filepath.Rel(sharedDir, path)
		if err != nil {
			return err
		}

		// Only process top-level entries (files and direct subdirectories)
		// This prevents processing nested items twice
		if filepath.Dir(relPath) != "." {
			// Skip nested entries; they'll be handled by their parent directory copy
			return nil
		}

		// Try to restore this file/directory
		fmt.Printf("\nRestoring %s...\n", relPath)
		if err := restoreFile(relPath, sharedDir, worktreeRoot); err != nil {
			errMsg := fmt.Sprintf("  ❌ %s: %v", relPath, err)
			fmt.Println(errMsg)
			errors = append(errors, errMsg)
		} else {
			fmt.Printf("  ✓ %s\n", relPath)
			count++
		}

		// If it's a directory, skip walking into it
		if info.IsDir() {
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking shared directory: %w", err)
	}

	fmt.Printf("\nRestored %d file(s)\n", count)

	if len(errors) > 0 {
		fmt.Printf("\nErrors encountered:\n")
		for _, errMsg := range errors {
			fmt.Println(errMsg)
		}
		return fmt.Errorf("some files failed to restore")
	}

	return nil
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().BoolVar(&restoreLink, "link", false, "Create symlink instead of copying")
	restoreCmd.Flags().StringVar(&restoreTo, "to", "", "Restore to a different path")
	restoreCmd.Flags().BoolVar(&restoreForce, "force", false, "Overwrite if file exists")
	restoreCmd.Flags().BoolVar(&restoreAll, "all", false, "Restore all persisted files")
}
