package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCloneCmd(t *testing.T) {
	// Create a temporary directory for the "remote" repository
	remoteDir := t.TempDir()
	remoteRepo := filepath.Join(remoteDir, "remote.git")

	// Initialize a bare repository
	initCmd := exec.Command("git", "init", "--bare", remoteRepo)
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}

	// Create a temporary directory for the destination
	destDir := t.TempDir()
	cloneDir := filepath.Join(destDir, "cloned-repo")

	// Run the clone command
	// We pass the remote URL (file path) and the destination directory
	err := cloneCmd.RunE(cloneCmd, []string{remoteRepo, cloneDir})
	if err != nil {
		t.Fatalf("cloneCmd failed: %v", err)
	}

	// Verify the directory exists
	if _, err := os.Stat(cloneDir); os.IsNotExist(err) {
		t.Errorf("Cloned directory does not exist at %s", cloneDir)
	}

	// Verify the config was set
	// git config --get remote.origin.fetch
	configCmd := exec.Command("git", "config", "--get", "remote.origin.fetch")
	configCmd.Dir = cloneDir
	output, err := configCmd.Output()
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	expected := "+refs/heads/*:refs/remotes/origin/*\n"
	if string(output) != expected {
		t.Errorf("Expected fetch refspec %q, got %q", expected, string(output))
	}
}
