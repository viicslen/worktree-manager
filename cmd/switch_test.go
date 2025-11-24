package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectLocation(t *testing.T) {
	// Create a temporary bare repository
	tempDir := t.TempDir()
	bareRepoPath := filepath.Join(tempDir, "test-bare-repo")

	// Initialize bare repository
	initCmd := exec.Command("git", "init", "--bare", bareRepoPath)
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}

	// Create an initial commit in the bare repo (needed to have branches)
	// We'll do this by cloning it, making a commit, and pushing
	tempClone := filepath.Join(tempDir, "temp-clone")
	cloneCmd := exec.Command("git", "clone", bareRepoPath, tempClone)
	if err := cloneCmd.Run(); err != nil {
		t.Fatalf("Failed to clone bare repo: %v", err)
	}

	// Configure git in the temp clone
	configUserCmd := exec.Command("git", "config", "user.email", "test@example.com")
	configUserCmd.Dir = tempClone
	_ = configUserCmd.Run()

	configNameCmd := exec.Command("git", "config", "user.name", "Test User")
	configNameCmd.Dir = tempClone
	_ = configNameCmd.Run()

	// Create an initial commit
	readmeFile := filepath.Join(tempClone, "README.md")
	if err := os.WriteFile(readmeFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}

	addCmd := exec.Command("git", "add", "README.md")
	addCmd.Dir = tempClone
	if err := addCmd.Run(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	commitCmd := exec.Command("git", "commit", "-m", "Initial commit")
	commitCmd.Dir = tempClone
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	pushCmd := exec.Command("git", "push", "origin", "master")
	pushCmd.Dir = tempClone
	if err := pushCmd.Run(); err != nil {
		// Try with main branch
		pushCmd = exec.Command("git", "push", "origin", "main")
		pushCmd.Dir = tempClone
		if err := pushCmd.Run(); err != nil {
			t.Fatalf("Failed to push: %v", err)
		}
	}

	// Test 1: Detect location from bare repo root
	t.Run("detect from bare repo root", func(t *testing.T) {
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		if err := os.Chdir(bareRepoPath); err != nil {
			t.Fatalf("Failed to change to bare repo: %v", err)
		}

		bareRoot, location, err := detectLocation()
		if err != nil {
			t.Fatalf("detectLocation failed: %v", err)
		}

		if location != "bare-root" {
			t.Errorf("Expected location 'bare-root', got %q", location)
		}

		if bareRoot != bareRepoPath {
			t.Errorf("Expected bareRepoRoot %q, got %q", bareRepoPath, bareRoot)
		}
	})

	// Test 2: Detect location from workspace
	t.Run("detect from workspace", func(t *testing.T) {
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		// Create a workspace worktree
		workspacePath := filepath.Join(bareRepoPath, "workspace")
		addWorktreeCmd := exec.Command("git", "worktree", "add", workspacePath, "master")
		addWorktreeCmd.Dir = bareRepoPath
		if err := addWorktreeCmd.Run(); err != nil {
			// Try with main branch
			addWorktreeCmd = exec.Command("git", "worktree", "add", workspacePath, "main")
			addWorktreeCmd.Dir = bareRepoPath
			if err := addWorktreeCmd.Run(); err != nil {
				t.Fatalf("Failed to add workspace worktree: %v", err)
			}
		}

		if err := os.Chdir(workspacePath); err != nil {
			t.Fatalf("Failed to change to workspace: %v", err)
		}

		bareRoot, location, err := detectLocation()
		if err != nil {
			t.Fatalf("detectLocation failed: %v", err)
		}

		if location != "workspace" {
			t.Errorf("Expected location 'workspace', got %q", location)
		}

		if bareRoot != bareRepoPath {
			t.Errorf("Expected bareRepoRoot %q, got %q", bareRepoPath, bareRoot)
		}
	})

	// Test 3: Error when in tree/ subdirectory
	t.Run("error from tree subdirectory", func(t *testing.T) {
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		// Create a tree/branch worktree (use HEAD to create detached worktree)
		treeDir := filepath.Join(bareRepoPath, "tree")
		os.MkdirAll(treeDir, 0755)
		treeBranchPath := filepath.Join(treeDir, "test-branch")

		addWorktreeCmd := exec.Command("git", "worktree", "add", "--detach", treeBranchPath, "HEAD")
		addWorktreeCmd.Dir = bareRepoPath
		if err := addWorktreeCmd.Run(); err != nil {
			t.Fatalf("Failed to add tree worktree: %v", err)
		}

		if err := os.Chdir(treeBranchPath); err != nil {
			t.Fatalf("Failed to change to tree branch: %v", err)
		}

		_, _, err := detectLocation()
		if err == nil {
			t.Error("Expected error when running from tree/ subdirectory, got nil")
		}

		if !strings.Contains(err.Error(), "cannot run switch from inside tree/") {
			t.Errorf("Expected error about tree/ directory, got: %v", err)
		}
	})
}

func TestGetCurrentBranch(t *testing.T) {
	// Create a temporary repository
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	// Initialize repository
	initCmd := exec.Command("git", "init", repoPath)
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	// Configure git
	configUserCmd := exec.Command("git", "config", "user.email", "test@example.com")
	configUserCmd.Dir = repoPath
	_ = configUserCmd.Run()

	configNameCmd := exec.Command("git", "config", "user.name", "Test User")
	configNameCmd.Dir = repoPath
	_ = configNameCmd.Run()

	// Create an initial commit
	readmeFile := filepath.Join(repoPath, "README.md")
	if err := os.WriteFile(readmeFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}

	addCmd := exec.Command("git", "add", "README.md")
	addCmd.Dir = repoPath
	if err := addCmd.Run(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	commitCmd := exec.Command("git", "commit", "-m", "Initial commit")
	commitCmd.Dir = repoPath
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Get the current branch
	branch, err := getCurrentBranch(repoPath)
	if err != nil {
		t.Fatalf("getCurrentBranch failed: %v", err)
	}

	// Should be either master or main depending on git version
	if branch != "master" && branch != "main" {
		t.Errorf("Expected branch 'master' or 'main', got %q", branch)
	}
}

func TestSwitchCmd_Integration(t *testing.T) {
	// This is a full integration test that simulates the switch workflow
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

	// Create initial commit (same as in TestDetectLocation)
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

	// Create a second branch
	checkoutCmd := exec.Command("git", "checkout", "-b", "feature-branch")
	checkoutCmd.Dir = tempClone
	_ = checkoutCmd.Run()

	featureFile := filepath.Join(tempClone, "feature.txt")
	os.WriteFile(featureFile, []byte("feature\n"), 0644)
	addCmd = exec.Command("git", "add", "feature.txt")
	addCmd.Dir = tempClone
	_ = addCmd.Run()

	commitCmd = exec.Command("git", "commit", "-m", "Add feature")
	commitCmd.Dir = tempClone
	_ = commitCmd.Run()

	pushCmd = exec.Command("git", "push", "origin", "feature-branch")
	pushCmd.Dir = tempClone
	_ = pushCmd.Run()

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Test: Switch to master from bare repo (creates workspace)
	t.Run("switch creates workspace", func(t *testing.T) {
		if err := os.Chdir(bareRepoPath); err != nil {
			t.Fatalf("Failed to change to bare repo: %v", err)
		}

		// Use master or main depending on what exists
		branchName := "master"
		checkCmd := exec.Command("git", "branch", "-r")
		checkCmd.Dir = bareRepoPath
		if output, _ := checkCmd.Output(); !strings.Contains(string(output), "master") {
			branchName = "main"
		}

		err := switchCmd.RunE(switchCmd, []string{branchName})
		if err != nil {
			t.Fatalf("switch command failed: %v", err)
		}

		workspacePath := filepath.Join(bareRepoPath, "workspace")
		if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
			t.Error("workspace directory was not created")
		}

		// Verify workspace is on the correct branch
		branch, _ := getCurrentBranch(workspacePath)
		if branch != branchName {
			t.Errorf("Expected workspace on %s, got %s", branchName, branch)
		}
	})

	// Test: Switch to feature-branch (moves workspace to tree/, creates new workspace)
	t.Run("switch moves workspace and creates new", func(t *testing.T) {
		if err := os.Chdir(bareRepoPath); err != nil {
			t.Fatalf("Failed to change to bare repo: %v", err)
		}

		err := switchCmd.RunE(switchCmd, []string{"feature-branch"})
		if err != nil {
			t.Fatalf("switch command failed: %v", err)
		}

		workspacePath := filepath.Join(bareRepoPath, "workspace")
		branch, _ := getCurrentBranch(workspacePath)
		if branch != "feature-branch" {
			t.Errorf("Expected workspace on feature-branch, got %s", branch)
		}

		// Check that old branch was moved to tree/
		branchName := "master"
		checkCmd := exec.Command("git", "branch", "-r")
		checkCmd.Dir = bareRepoPath
		if output, _ := checkCmd.Output(); !strings.Contains(string(output), "master") {
			branchName = "main"
		}

		treePath := filepath.Join(bareRepoPath, "tree", branchName)
		if _, err := os.Stat(treePath); os.IsNotExist(err) {
			t.Errorf("tree/%s directory was not created", branchName)
		}
	})
}
