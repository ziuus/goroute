package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentsDocIntegrity(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	// Resolve repository root
	repoRoot := filepath.Dir(wd)
	if _, err := os.Stat(filepath.Join(repoRoot, "AGENTS.md")); os.IsNotExist(err) {
		repoRoot = wd
	}

	agentsPath := filepath.Join(repoRoot, "AGENTS.md")
	content, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("AGENTS.md missing at %s: %v", agentsPath, err)
	}

	text := string(content)
	lines := strings.Split(text, "\n")
	if len(lines) > 150 {
		t.Errorf("AGENTS.md exceeds 150 lines (got %d)", len(lines))
	}

	requiredSections := []string{
		"Quick Commands",
		"Key Repository Structure",
		"Conventions Agents Commonly Get Wrong",
		"Do Not Modify",
		"Deeper Documentation",
	}

	for _, sec := range requiredSections {
		if !strings.Contains(text, sec) {
			t.Errorf("AGENTS.md is missing required section %q", sec)
		}
	}
}
