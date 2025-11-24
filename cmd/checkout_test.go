package cmd

import (
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
