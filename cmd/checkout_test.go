package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple branch name",
			input:    "master",
			expected: "master",
		},
		{
			name:     "branch with slash",
			input:    "feature/new-feature",
			expected: "feature-new-feature",
		},
		{
			name:     "branch with multiple slashes",
			input:    "bugfix/critical/issue-123",
			expected: "bugfix-critical-issue-123",
		},
		{
			name:     "commit hash",
			input:    "7fd1a60",
			expected: "7fd1a60",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the sanitization logic from checkout.go
			commitish := tt.input
			dirName := ""
			for _, char := range commitish {
				if char == '/' {
					dirName += "-"
				} else {
					dirName += string(char)
				}
			}

			if dirName != tt.expected {
				t.Errorf("sanitize(%q) = %q, want %q", tt.input, dirName, tt.expected)
			}
		})
	}
}

func TestCheckoutCmd_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary bare repository
	tempDir := t.TempDir()
	bareRepoPath := filepath.Join(tempDir, "test-bare-repo")

	// Initialize bare repository
	initCmd := exec.Command("git", "init", "--bare", bareRepoPath)
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}

	// Create an initial commit
	tempClone := filepath.Join(tempDir, "temp-clone")
	cloneCmd := exec.Command("git", "clone", bareRepoPath, tempClone)
	if err := cloneCmd.Run(); err != nil {
		t.Fatalf("Failed to clone bare repo: %v", err)
	}

	configUserCmd := exec.Command("git", "config", "user.email", "test@example.com")
	configUserCmd.Dir = tempClone
	_ = configUserCmd.Run()

	configNameCmd := exec.Command("git", "config", "user.name", "Test User")
	configNameCmd.Dir = tempClone
	_ = configNameCmd.Run()

	readmeFile := filepath.Join(tempClone, "README.md")
	if err := os.WriteFile(readmeFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}

	addCmd := exec.Command("git", "add", "README.md")
	addCmd.Dir = tempClone
	_ = addCmd.Run()

	commitCmd := exec.Command("git", "commit", "-m", "Initial commit")
	commitCmd.Dir = tempClone
	_ = commitCmd.Run()

	pushCmd := exec.Command("git", "push", "origin", "master")
	pushCmd.Dir = tempClone
	if err := pushCmd.Run(); err != nil {
		pushCmd = exec.Command("git", "push", "origin", "main")
		pushCmd.Dir = tempClone
		_ = pushCmd.Run()
	}

	// Create feature-restore branch
	createBranchCmd := exec.Command("git", "checkout", "-b", "feature-restore")
	createBranchCmd.Dir = tempClone
	_ = createBranchCmd.Run()

	pushFeatureCmd := exec.Command("git", "push", "origin", "feature-restore")
	pushFeatureCmd.Dir = tempClone
	if err := pushFeatureCmd.Run(); err != nil {
		t.Fatalf("Failed to push feature-restore branch: %v", err)
	}

	// Create a setup worktree to run persist add
	setupWorktreePath := filepath.Join(bareRepoPath, "setup-worktree")
	setupWorktreeCmd := exec.Command("git", "worktree", "add", setupWorktreePath, "master")
	setupWorktreeCmd.Dir = bareRepoPath
	if err := setupWorktreeCmd.Run(); err != nil {
		// Try main if master fails
		setupWorktreeCmd = exec.Command("git", "worktree", "add", setupWorktreePath, "main")
		setupWorktreeCmd.Dir = bareRepoPath
		if err := setupWorktreeCmd.Run(); err != nil {
			t.Fatalf("Failed to create setup worktree: %v", err)
		}
	}

	// Create a file to persist
	configFile := filepath.Join(setupWorktreePath, "config.json")
	if err := os.WriteFile(configFile, []byte(`{"key": "value"}`), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Run persist add from the setup worktree
	func() {
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		if err := os.Chdir(setupWorktreePath); err != nil {
			t.Fatalf("Failed to change to setup worktree: %v", err)
		}

		// We need to use the persistAddCmd
		// Since it's in the same package, we can access it
		err := persistAddCmd.RunE(persistAddCmd, []string{"config.json"})
		if err != nil {
			t.Fatalf("persist add failed: %v", err)
		}
	}()

	// Test checkout with --restore
	t.Run("checkout with restore", func(t *testing.T) {
		// Change to bare repo root
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		if err := os.Chdir(bareRepoPath); err != nil {
			t.Fatalf("Failed to change to bare repo: %v", err)
		}

		// Checkout a new branch
		branchName := "feature-restore"

		// Reset flags for this test run
		checkoutCmd.Flags().Set("restore", "true")
		defer checkoutCmd.Flags().Set("restore", "false")

		err := checkoutCmd.RunE(checkoutCmd, []string{branchName})
		if err != nil {
			t.Fatalf("checkout command failed: %v", err)
		}

		// Verify worktree created
		worktreePath := filepath.Join(bareRepoPath, "tree", branchName)
		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			t.Errorf("worktree directory was not created")
		}

		// Verify shared file restored
		restoredFile := filepath.Join(worktreePath, "config.json")
		if _, err := os.Stat(restoredFile); os.IsNotExist(err) {
			t.Errorf("shared file was not restored")
		}
	})
}
