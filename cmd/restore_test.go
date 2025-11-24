package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRestoreFile(t *testing.T) {
	// Setup: Create bare repo with worktree
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

	t.Run("restore file by copying", func(t *testing.T) {
		// Reset flags
		restoreTo = ""
		restoreLink = false
		restoreForce = false
		restoreAll = false

		// Create shared file
		sharedDir := filepath.Join(bareRepoDir, "shared")
		os.MkdirAll(sharedDir, 0755)
		sharedFile := filepath.Join(sharedDir, ".env")
		content := "SECRET_KEY=abc123"
		os.WriteFile(sharedFile, []byte(content), 0644)

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		// Restore file
		rootCmd.SetArgs([]string{"restore", ".env"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Failed to restore file: %v", err)
		}

		// Verify file exists in worktree
		restoredFile := filepath.Join(worktreeDir, ".env")
		if _, err := os.Stat(restoredFile); os.IsNotExist(err) {
			t.Error("Restored file does not exist")
		}

		// Verify content
		restoredContent, _ := os.ReadFile(restoredFile)
		if string(restoredContent) != content {
			t.Errorf("Content mismatch: got %q, want %q", string(restoredContent), content)
		}

		// Verify it's a regular file, not a symlink
		fileInfo, _ := os.Lstat(restoredFile)
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			t.Error("Expected regular file, got symlink")
		}
	})

	t.Run("restore file with symlink", func(t *testing.T) {
		// Reset flags
		restoreTo = ""
		restoreLink = false
		restoreForce = false
		restoreAll = false

		// Create shared file
		sharedDir := filepath.Join(bareRepoDir, "shared")
		os.MkdirAll(sharedDir, 0755)
		sharedFile := filepath.Join(sharedDir, "config.json")
		content := `{"key": "value"}`
		os.WriteFile(sharedFile, []byte(content), 0644)

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		// Restore with --link flag
		rootCmd.SetArgs([]string{"restore", "config.json", "--link"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Failed to restore file with link: %v", err)
		}

		// Verify symlink exists
		restoredFile := filepath.Join(worktreeDir, "config.json")
		fileInfo, err := os.Lstat(restoredFile)
		if err != nil {
			t.Fatalf("Restored file does not exist: %v", err)
		}

		if fileInfo.Mode()&os.ModeSymlink == 0 {
			t.Error("Expected symlink, got regular file")
		}

		// Verify symlink points to correct location
		linkTarget, err := os.Readlink(restoredFile)
		if err != nil {
			t.Fatalf("Failed to read symlink: %v", err)
		}

		// Should be relative link
		if filepath.IsAbs(linkTarget) {
			t.Errorf("Expected relative symlink, got absolute: %s", linkTarget)
		}

		// Verify content through symlink
		restoredContent, _ := os.ReadFile(restoredFile)
		if string(restoredContent) != content {
			t.Errorf("Content mismatch through symlink: got %q, want %q", string(restoredContent), content)
		}
	})

	t.Run("restore file to custom path", func(t *testing.T) {
		// Reset flags
		restoreTo = ""
		restoreLink = false
		restoreForce = false

		// Create shared file
		sharedDir := filepath.Join(bareRepoDir, "shared")
		sharedFile := filepath.Join(sharedDir, "original.txt")
		content := "test content"
		os.WriteFile(sharedFile, []byte(content), 0644)

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		// Restore to different path
		rootCmd.SetArgs([]string{"restore", "original.txt", "--to", "custom/path/renamed.txt"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Failed to restore to custom path: %v", err)
		}

		// Verify file at custom location
		customFile := filepath.Join(worktreeDir, "custom", "path", "renamed.txt")
		if _, err := os.Stat(customFile); os.IsNotExist(err) {
			t.Error("File not found at custom path")
		}

		// Verify content
		restoredContent, _ := os.ReadFile(customFile)
		if string(restoredContent) != content {
			t.Errorf("Content mismatch: got %q, want %q", string(restoredContent), content)
		}
	})

	t.Run("restore with force flag overwrites existing", func(t *testing.T) {
		// Reset flags
		restoreTo = ""
		restoreLink = false
		restoreForce = false

		// Create shared file
		sharedDir := filepath.Join(bareRepoDir, "shared")
		sharedFile := filepath.Join(sharedDir, "overwrite.txt")
		newContent := "new content"
		os.WriteFile(sharedFile, []byte(newContent), 0644)

		// Create existing file in worktree
		existingFile := filepath.Join(worktreeDir, "overwrite.txt")
		os.WriteFile(existingFile, []byte("old content"), 0644)

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		// Restore with --force
		rootCmd.SetArgs([]string{"restore", "overwrite.txt", "--force"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Failed to restore with force: %v", err)
		}

		// Verify content was overwritten
		restoredContent, _ := os.ReadFile(existingFile)
		if string(restoredContent) != newContent {
			t.Errorf("Content not overwritten: got %q, want %q", string(restoredContent), newContent)
		}
	})

	t.Run("restore directory recursively", func(t *testing.T) {
		// Reset flags
		restoreTo = ""
		restoreLink = false
		restoreForce = false

		// Create shared directory with files
		sharedDir := filepath.Join(bareRepoDir, "shared")
		libDir := filepath.Join(sharedDir, "lib")
		os.MkdirAll(libDir, 0755)
		os.WriteFile(filepath.Join(libDir, "file1.js"), []byte("content1"), 0644)
		os.WriteFile(filepath.Join(libDir, "file2.js"), []byte("content2"), 0644)

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		// Restore directory
		rootCmd.SetArgs([]string{"restore", "lib"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Failed to restore directory: %v", err)
		}

		// Verify all files exist
		file1 := filepath.Join(worktreeDir, "lib", "file1.js")
		file2 := filepath.Join(worktreeDir, "lib", "file2.js")

		if _, err := os.Stat(file1); os.IsNotExist(err) {
			t.Error("file1.js not restored")
		}
		if _, err := os.Stat(file2); os.IsNotExist(err) {
			t.Error("file2.js not restored")
		}
	})

	t.Run("error when file not in shared", func(t *testing.T) {
		// Reset flags
		restoreTo = ""
		restoreLink = false
		restoreForce = false

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		rootCmd.SetArgs([]string{"restore", "nonexistent.txt"})
		err := rootCmd.Execute()
		if err == nil {
			t.Error("Expected error when restoring nonexistent file, got none")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})

	t.Run("error when file exists without force", func(t *testing.T) {
		// Reset flags
		restoreTo = ""
		restoreLink = false
		restoreForce = false

		// Create shared file
		sharedDir := filepath.Join(bareRepoDir, "shared")
		sharedFile := filepath.Join(sharedDir, "exists.txt")
		os.WriteFile(sharedFile, []byte("content"), 0644)

		// Create existing file in worktree
		existingFile := filepath.Join(worktreeDir, "exists.txt")
		os.WriteFile(existingFile, []byte("existing"), 0644)

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		rootCmd.SetArgs([]string{"restore", "exists.txt"})
		err := rootCmd.Execute()
		if err == nil {
			t.Error("Expected error when file exists without force, got none")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("Expected 'already exists' error, got: %v", err)
		}
	})
}

func TestRestoreAll(t *testing.T) {
	// Setup: Create bare repo with worktree
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

	t.Run("restore all files by copying", func(t *testing.T) {
		// Reset flags
		restoreTo = ""
		restoreLink = false
		restoreForce = false
		restoreAll = false

		// Create multiple shared files
		sharedDir := filepath.Join(bareRepoDir, "shared")
		os.MkdirAll(sharedDir, 0755)
		os.WriteFile(filepath.Join(sharedDir, ".env"), []byte("ENV=test"), 0644)
		os.MkdirAll(filepath.Join(sharedDir, "config"), 0755)
		os.WriteFile(filepath.Join(sharedDir, "config", "app.json"), []byte("{}"), 0644)

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		// Restore all
		rootCmd.SetArgs([]string{"restore", "--all"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Failed to restore all: %v", err)
		}

		// Verify files exist
		envFile := filepath.Join(worktreeDir, ".env")
		configFile := filepath.Join(worktreeDir, "config", "app.json")

		if _, err := os.Stat(envFile); os.IsNotExist(err) {
			t.Error(".env not restored")
		}
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			t.Error("config/app.json not restored")
		}
	})

	t.Run("restore all with symlinks", func(t *testing.T) {
		// Reset flags
		restoreTo = ""
		restoreLink = false
		restoreForce = false
		restoreAll = false

		// Clean worktree
		os.RemoveAll(filepath.Join(worktreeDir, ".env"))
		os.RemoveAll(filepath.Join(worktreeDir, "config"))

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		// Restore all with links
		rootCmd.SetArgs([]string{"restore", "--all", "--link"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Failed to restore all with links: %v", err)
		}

		// Verify symlinks
		envFile := filepath.Join(worktreeDir, ".env")
		configDir := filepath.Join(worktreeDir, "config")

		envInfo, _ := os.Lstat(envFile)
		if envInfo.Mode()&os.ModeSymlink == 0 {
			t.Error(".env is not a symlink")
		}

		configInfo, _ := os.Lstat(configDir)
		if configInfo.Mode()&os.ModeSymlink == 0 {
			t.Error("config is not a symlink")
		}
	})

	t.Run("error when no shared directory", func(t *testing.T) {
		// Reset flags
		restoreTo = ""
		restoreLink = false
		restoreForce = false
		restoreAll = false

		// Create new bare repo without shared directory
		newBareRepo := t.TempDir()
		cmd := exec.Command("git", "init", "--bare", newBareRepo)
		cmd.Run()

		newTempRepo := t.TempDir()
		cmd = exec.Command("git", "init", newTempRepo)
		cmd.Run()

		testFile := filepath.Join(newTempRepo, "README.md")
		os.WriteFile(testFile, []byte("# Test"), 0644)

		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		cmd.Dir = newTempRepo
		cmd.Run()

		cmd = exec.Command("git", "config", "user.name", "Test User")
		cmd.Dir = newTempRepo
		cmd.Run()

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = newTempRepo
		cmd.Run()

		cmd = exec.Command("git", "commit", "-m", "Initial commit")
		cmd.Dir = newTempRepo
		cmd.Run()

		cmd = exec.Command("git", "remote", "add", "origin", newBareRepo)
		cmd.Dir = newTempRepo
		cmd.Run()

		cmd = exec.Command("git", "push", "origin", "master")
		cmd.Dir = newTempRepo
		if err := cmd.Run(); err != nil {
			cmd = exec.Command("git", "branch", "-M", "main")
			cmd.Dir = newTempRepo
			cmd.Run()
			cmd = exec.Command("git", "push", "origin", "main")
			cmd.Dir = newTempRepo
			cmd.Run()
		}

		newWorktree := filepath.Join(newBareRepo, "workspace")
		cmd = exec.Command("git", "worktree", "add", newWorktree, "main")
		cmd.Dir = newBareRepo
		if err := cmd.Run(); err != nil {
			cmd = exec.Command("git", "worktree", "add", newWorktree, "master")
			cmd.Dir = newBareRepo
			cmd.Run()
		}

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(newWorktree)

		rootCmd.SetArgs([]string{"restore", "--all"})
		err := rootCmd.Execute()
		if err == nil {
			t.Error("Expected error when shared directory doesn't exist, got none")
		}
		if !strings.Contains(err.Error(), "no persisted files") {
			t.Errorf("Expected 'no persisted files' error, got: %v", err)
		}
	})

	t.Run("error when specifying file with --all", func(t *testing.T) {
		// Reset flags
		restoreTo = ""
		restoreLink = false
		restoreForce = false
		restoreAll = false

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		rootCmd.SetArgs([]string{"restore", ".env", "--all"})
		err := rootCmd.Execute()
		if err == nil {
			t.Error("Expected error when specifying file with --all, got none")
		}
		if !strings.Contains(err.Error(), "cannot specify file with --all") {
			t.Errorf("Expected 'cannot specify file with --all' error, got: %v", err)
		}
	})

	t.Run("error when no file specified and no --all", func(t *testing.T) {
		// Reset flags
		restoreTo = ""
		restoreLink = false
		restoreForce = false
		restoreAll = false

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(worktreeDir)

		rootCmd.SetArgs([]string{"restore"})
		err := rootCmd.Execute()
		if err == nil {
			t.Error("Expected error when no arguments provided, got none")
		}
	})
}

func TestRestoreIntegration(t *testing.T) {
	// Full workflow test: persist from one worktree and restore in another
	bareRepoDir := t.TempDir()

	cmd := exec.Command("git", "init", "--bare", bareRepoDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create bare repo: %v", err)
	}

	// Create temp repo for initial commit
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

	// Create first worktree
	worktree1 := filepath.Join(bareRepoDir, "workspace1")
	cmd = exec.Command("git", "worktree", "add", worktree1, "main")
	cmd.Dir = bareRepoDir
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("git", "worktree", "add", worktree1, "master")
		cmd.Dir = bareRepoDir
		cmd.Run()
	}

	// Persist a file from worktree1
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	sharedFile := filepath.Join(worktree1, "shared.txt")
	sharedContent := "shared across worktrees"
	os.WriteFile(sharedFile, []byte(sharedContent), 0644)

	os.Chdir(worktree1)
	rootCmd.SetArgs([]string{"persist", "add", "shared.txt"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Failed to persist from worktree1: %v", err)
	}

	// Create second worktree
	worktree2 := filepath.Join(bareRepoDir, "workspace2")
	cmd = exec.Command("git", "worktree", "add", worktree2, "HEAD")
	cmd.Dir = bareRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create worktree2: %v", err)
	}

	// Restore in worktree2
	os.Chdir(worktree2)

	// Reset flags
	restoreTo = ""
	restoreLink = false
	restoreForce = false
	restoreAll = false

	rootCmd.SetArgs([]string{"restore", "shared.txt"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Failed to restore in worktree2: %v", err)
	}

	// Verify file exists in worktree2 with correct content
	restoredFile := filepath.Join(worktree2, "shared.txt")
	restoredContent, err := os.ReadFile(restoredFile)
	if err != nil {
		t.Fatalf("Restored file not found in worktree2: %v", err)
	}
	if string(restoredContent) != sharedContent {
		t.Errorf("Content mismatch in worktree2: got %q, want %q", string(restoredContent), sharedContent)
	}
}
