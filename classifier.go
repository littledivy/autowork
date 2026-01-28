package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Classifier struct {
	reposDir string
}

func NewClassifier(reposDir string) *Classifier {
	return &Classifier{
		reposDir: reposDir,
	}
}

func (c *Classifier) listRepos() ([]string, error) {
	entries, err := os.ReadDir(c.reposDir)
	if err != nil {
		return nil, err
	}

	var repos []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			// Check if it's a git repo
			gitDir := filepath.Join(c.reposDir, entry.Name(), ".git")
			if _, err := os.Stat(gitDir); err == nil {
				repos = append(repos, entry.Name())
			}
		}
	}
	return repos, nil
}

func (c *Classifier) Classify(message string) (*ClassificationResult, error) {
	repos, err := c.listRepos()
	if err != nil {
		return nil, fmt.Errorf("failed to list repos: %w", err)
	}

	prompt := c.buildPrompt(repos, message)

	result, err := c.callClaude(prompt)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Classifier) buildPrompt(repos []string, message string) string {
	repoList := strings.Join(repos, "\n- ")

	return fmt.Sprintf(`You classify Slack messages to determine if they're actionable coding/development work items.

Available repositories:
- %s

Slack message:
"%s"

Analyze this message and determine:
1. Is this an actionable work item? (bug fix, feature request, code task, etc.)
2. If yes, which repository from the list above does it relate to?
3. Generate a brief summary and a git branch name.

IMPORTANT: Only respond with valid JSON, no other text.

If actionable, respond with:
{"actionable": true, "repo": "repo-name-from-list", "summary": "Brief description of the task", "branch": "kebab-case-branch-name"}

If NOT actionable (casual chat, questions without action items, greetings, etc):
{"actionable": false}`, repoList, message)
}

func (c *Classifier) callClaude(prompt string) (*ClassificationResult, error) {
	// Use claude CLI with --print for non-interactive output
	cmd := exec.Command("claude", "--print", "--model", "sonnet", prompt)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("claude CLI error: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("claude CLI error: %w", err)
	}

	content := strings.TrimSpace(string(output))

	// Clean up potential markdown code blocks
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var classification ClassificationResult
	if err := json.Unmarshal([]byte(content), &classification); err != nil {
		return nil, fmt.Errorf("failed to parse classification: %w (content: %s)", err, content)
	}

	return &classification, nil
}
