package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPersistAdd(t *testing.T) {
	// Create temporary bare repository
	bareRepoDir := t.TempDir()

	// Initialize bare repository
	cmd := exec.Command("git", "init", "--bare", bareRepoDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create bare repo: %v", err)
	}

	// Configure fetch for worktrees
	cmd = exec.Command("git", "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	cmd.Dir = bareRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure bare repo: %v", err)
	}

	// Create an initial commit in a temporary normal repo
	tempRepoDir := t.TempDir()
	cmd = exec.Command("git", "init", tempRepoDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create temp repo: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(tempRepoDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempRepoDir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempRepoDir
	cmd.Run()

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tempRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add files: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Push to bare repo
	cmd = exec.Command("git", "remote", "add", "origin", bareRepoDir)
	cmd.Dir = tempRepoDir
	cmd.Run()

	cmd = exec.Command("git", "push", "origin", "master")
	cmd.Dir = tempRepoDir
	if err := cmd.Run(); err != nil {
		// Try main branch
		cmd = exec.Command("git", "branch", "-M", "main")
		cmd.Dir = tempRepoDir
		cmd.Run()

		cmd = exec.Command("git", "push", "origin", "main")
		cmd.Dir = tempRepoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to push: %v", err)
		}
	}

	// Create worktree
	worktreeDir := filepath.Join(bareRepoDir, "workspace")
	cmd = exec.Command("git", "worktree", "add", worktreeDir, "main")
	cmd.Dir = bareRepoDir
	cmd.Env = append(os.Environ(), "GIT_DIR="+bareRepoDir)
	if err := cmd.Run(); err != nil {
		// Try master branch
		cmd = exec.Command("git", "worktree", "add", worktreeDir, "master")
		cmd.Dir = bareRepoDir
		cmd.Env = append(os.Environ(), "GIT_DIR="+bareRepoDir)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to create worktree: %v", err)
		}
	}

	t.Run("persist file successfully", func(t *testing.T) {
		// Create a test file in worktree
		testFile := filepath.Join(worktreeDir, ".env")
		content := "SECRET_KEY=test123"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Change to worktree directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		// Run persist add
		rootCmd.SetArgs([]string{"persist", "add", ".env"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Failed to persist file: %v", err)
		}

		// Verify file exists in shared/
		sharedFile := filepath.Join(bareRepoDir, "shared", ".env")
		if _, err := os.Stat(sharedFile); os.IsNotExist(err) {
			t.Errorf("Persisted file does not exist in shared/")
		}

		// Verify content matches
		sharedContent, err := os.ReadFile(sharedFile)
		if err != nil {
			t.Fatalf("Failed to read shared file: %v", err)
		}
		if string(sharedContent) != content {
			t.Errorf("Content mismatch: got %q, want %q", string(sharedContent), content)
		}
	})

	t.Run("persist nested file preserves structure", func(t *testing.T) {
		// Create nested directory structure
		configDir := filepath.Join(worktreeDir, "src", "config")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("Failed to create config dir: %v", err)
		}

		configFile := filepath.Join(configDir, "database.json")
		content := `{"host": "localhost"}`
		if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		// Change to worktree directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		// Run persist add
		rootCmd.SetArgs([]string{"persist", "add", "src/config/database.json"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Failed to persist nested file: %v", err)
		}

		// Verify file exists in shared/ with correct structure
		sharedFile := filepath.Join(bareRepoDir, "shared", "src", "config", "database.json")
		if _, err := os.Stat(sharedFile); os.IsNotExist(err) {
			t.Errorf("Persisted file does not exist in shared/ with correct path structure")
		}

		// Verify content
		sharedContent, err := os.ReadFile(sharedFile)
		if err != nil {
			t.Fatalf("Failed to read shared file: %v", err)
		}
		if string(sharedContent) != content {
			t.Errorf("Content mismatch: got %q, want %q", string(sharedContent), content)
		}
	})

	t.Run("persist directory recursively", func(t *testing.T) {
		// Create directory with multiple files
		libDir := filepath.Join(worktreeDir, "lib")
		if err := os.MkdirAll(libDir, 0755); err != nil {
			t.Fatalf("Failed to create lib dir: %v", err)
		}

		// Create multiple files
		files := map[string]string{
			"lib/util.js":   "export function util() {}",
			"lib/helper.js": "export function helper() {}",
		}

		for path, content := range files {
			fullPath := filepath.Join(worktreeDir, path)
			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create file %s: %v", path, err)
			}
		}

		// Change to worktree directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		// Run persist add for directory
		rootCmd.SetArgs([]string{"persist", "add", "lib"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Failed to persist directory: %v", err)
		}

		// Verify all files exist in shared/
		for path := range files {
			sharedFile := filepath.Join(bareRepoDir, "shared", path)
			if _, err := os.Stat(sharedFile); os.IsNotExist(err) {
				t.Errorf("Persisted file does not exist: %s", sharedFile)
			}
		}
	})

	t.Run("error when file already exists", func(t *testing.T) {
		// Create and persist a file
		testFile := filepath.Join(worktreeDir, "duplicate.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		rootCmd.SetArgs([]string{"persist", "add", "duplicate.txt"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Failed first persist: %v", err)
		}

		// Try to persist again
		rootCmd.SetArgs([]string{"persist", "add", "duplicate.txt"})
		err := rootCmd.Execute()
		if err == nil {
			t.Error("Expected error when persisting duplicate file, got none")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("Expected 'already exists' error, got: %v", err)
		}
	})

	t.Run("error when file does not exist", func(t *testing.T) {
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		rootCmd.SetArgs([]string{"persist", "add", "nonexistent.txt"})
		err := rootCmd.Execute()
		if err == nil {
			t.Error("Expected error when persisting nonexistent file, got none")
		}
		if !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("Expected 'does not exist' error, got: %v", err)
		}
	})
}

func TestPersistList(t *testing.T) {
	// Create temporary bare repository with worktree
	bareRepoDir := t.TempDir()

	cmd := exec.Command("git", "init", "--bare", bareRepoDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create bare repo: %v", err)
	}

	// Create temp repo for initial commit
	tempRepoDir := t.TempDir()
	cmd = exec.Command("git", "init", tempRepoDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create temp repo: %v", err)
	}

	testFile := filepath.Join(tempRepoDir, "README.md")
	os.WriteFile(testFile, []byte("# Test"), 0644)

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempRepoDir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempRepoDir
	cmd.Run()

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tempRepoDir
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempRepoDir
	cmd.Run()

	cmd = exec.Command("git", "remote", "add", "origin", bareRepoDir)
	cmd.Dir = tempRepoDir
	cmd.Run()

	cmd = exec.Command("git", "push", "origin", "master")
	cmd.Dir = tempRepoDir
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("git", "branch", "-M", "main")
		cmd.Dir = tempRepoDir
		cmd.Run()
		cmd = exec.Command("git", "push", "origin", "main")
		cmd.Dir = tempRepoDir
		cmd.Run()
	}

	worktreeDir := filepath.Join(bareRepoDir, "workspace")
	cmd = exec.Command("git", "worktree", "add", worktreeDir, "main")
	cmd.Dir = bareRepoDir
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("git", "worktree", "add", worktreeDir, "master")
		cmd.Dir = bareRepoDir
		cmd.Run()
	}

	t.Run("list empty shared directory", func(t *testing.T) {
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		// Should not error when shared directory doesn't exist
		rootCmd.SetArgs([]string{"persist", "list"})
		if err := rootCmd.Execute(); err != nil {
			t.Errorf("Unexpected error listing empty shared: %v", err)
		}
	})

	t.Run("list persisted files", func(t *testing.T) {
		// Create shared directory with test files
		sharedDir := filepath.Join(bareRepoDir, "shared")
		os.MkdirAll(sharedDir, 0755)

		// Create test files
		os.WriteFile(filepath.Join(sharedDir, ".env"), []byte("SECRET=123"), 0644)
		os.MkdirAll(filepath.Join(sharedDir, "config"), 0755)
		os.WriteFile(filepath.Join(sharedDir, "config", "app.json"), []byte("{}"), 0644)

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		// List should succeed (we can't easily capture output in tests, but ensure no error)
		rootCmd.SetArgs([]string{"persist", "list"})
		if err := rootCmd.Execute(); err != nil {
			t.Errorf("Failed to list persisted files: %v", err)
		}
	})
}

func TestPersistRemove(t *testing.T) {
	// Create temporary bare repository with worktree
	bareRepoDir := t.TempDir()

	cmd := exec.Command("git", "init", "--bare", bareRepoDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create bare repo: %v", err)
	}

	tempRepoDir := t.TempDir()
	cmd = exec.Command("git", "init", tempRepoDir)
	cmd.Run()

	testFile := filepath.Join(tempRepoDir, "README.md")
	os.WriteFile(testFile, []byte("# Test"), 0644)

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempRepoDir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempRepoDir
	cmd.Run()

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tempRepoDir
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempRepoDir
	cmd.Run()

	cmd = exec.Command("git", "remote", "add", "origin", bareRepoDir)
	cmd.Dir = tempRepoDir
	cmd.Run()

	cmd = exec.Command("git", "push", "origin", "master")
	cmd.Dir = tempRepoDir
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("git", "branch", "-M", "main")
		cmd.Dir = tempRepoDir
		cmd.Run()
		cmd = exec.Command("git", "push", "origin", "main")
		cmd.Dir = tempRepoDir
		cmd.Run()
	}

	worktreeDir := filepath.Join(bareRepoDir, "workspace")
	cmd = exec.Command("git", "worktree", "add", worktreeDir, "main")
	cmd.Dir = bareRepoDir
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("git", "worktree", "add", worktreeDir, "master")
		cmd.Dir = bareRepoDir
		cmd.Run()
	}

	t.Run("remove persisted file", func(t *testing.T) {
		// Create shared file
		sharedDir := filepath.Join(bareRepoDir, "shared")
		os.MkdirAll(sharedDir, 0755)
		testFile := filepath.Join(sharedDir, "test.txt")
		os.WriteFile(testFile, []byte("content"), 0644)

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		// Remove file
		rootCmd.SetArgs([]string{"persist", "remove", "test.txt"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Failed to remove file: %v", err)
		}

		// Verify file is gone
		if _, err := os.Stat(testFile); !os.IsNotExist(err) {
			t.Error("File still exists after removal")
		}
	})

	t.Run("remove persisted directory", func(t *testing.T) {
		// Create shared directory
		sharedDir := filepath.Join(bareRepoDir, "shared")
		testDir := filepath.Join(sharedDir, "testdir")
		os.MkdirAll(testDir, 0755)
		os.WriteFile(filepath.Join(testDir, "file.txt"), []byte("content"), 0644)

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		// Remove directory
		rootCmd.SetArgs([]string{"persist", "remove", "testdir"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Failed to remove directory: %v", err)
		}

		// Verify directory is gone
		if _, err := os.Stat(testDir); !os.IsNotExist(err) {
			t.Error("Directory still exists after removal")
		}
	})

	t.Run("error when file not found", func(t *testing.T) {
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		rootCmd.SetArgs([]string{"persist", "remove", "nonexistent.txt"})
		err := rootCmd.Execute()
		if err == nil {
			t.Error("Expected error when removing nonexistent file, got none")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})
}

func TestFindRepoRoots(t *testing.T) {
	t.Run("error when not in worktree", func(t *testing.T) {
		// Create a normal (non-worktree) repo
		tempDir := t.TempDir()
		cmd := exec.Command("git", "init", tempDir)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tempDir)

		_, _, err := findRepoRoots()
		if err == nil {
			t.Error("Expected error when not in worktree, got none")
		}
	})

	t.Run("error when in bare repository", func(t *testing.T) {
		// Create bare repo
		bareDir := t.TempDir()
		cmd := exec.Command("git", "init", "--bare", bareDir)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to create bare repo: %v", err)
		}

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(bareDir)

		_, _, err := findRepoRoots()
		if err == nil {
			t.Error("Expected error when in bare repository, got none")
		}
		if !strings.Contains(err.Error(), "bare repository") {
			t.Errorf("Expected 'bare repository' error, got: %v", err)
		}
	})
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{100, "100 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}
