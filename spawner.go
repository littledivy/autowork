package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Spawner struct {
	reposDir string
	useHappy bool
}

func NewSpawner(reposDir string, useHappy bool) *Spawner {
	return &Spawner{reposDir: reposDir, useHappy: useHappy}
}

func (s *Spawner) claudeBinary() string {
	if s.useHappy {
		return "happy"
	}
	return "claude"
}

func (s *Spawner) notify(msg string) {
	if !s.useHappy {
		return
	}
	cmd := exec.Command("happy", "notify", "-p", msg)
	cmd.Run()
}

func generateID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Spawner) SpawnSession(classification *ClassificationResult, slackMsg string) (*Session, error) {
	repoPath := filepath.Join(s.reposDir, classification.Repo)

	// Verify repo exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("repo not found: %s", repoPath)
	}

	// Create a new branch
	branchName := classification.Branch
	if err := s.createBranch(repoPath, branchName); err != nil {
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}

	// Generate session ID
	sessionID := generateID()

	// Build the prompt for Claude
	prompt := fmt.Sprintf(`You have been assigned this task from a Slack message:

Task: %s

Original message: "%s"

Please analyze the codebase and work on this task. When you're done with initial analysis and have a plan, stop and wait for the user to continue the session.`, classification.Summary, slackMsg)

	// Start Claude Code in the background
	if err := s.startClaudeSession(repoPath, prompt, sessionID); err != nil {
		return nil, fmt.Errorf("failed to start Claude session: %w", err)
	}

	session := &Session{
		ID:        sessionID,
		Repo:      classification.Repo,
		Branch:    branchName,
		Summary:   classification.Summary,
		RepoPath:  repoPath,
		CreatedAt: time.Now(),
		SlackMsg:  slackMsg,
	}

	// Save session for later
	if err := AddSession(*session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// Send push notification
	s.notify(fmt.Sprintf("New session: %s - %s", session.Repo, session.Summary))

	return session, nil
}

func (s *Spawner) createBranch(repoPath, branchName string) error {
	// First, fetch to make sure we have latest
	fetchCmd := exec.Command("git", "fetch", "--all")
	fetchCmd.Dir = repoPath
	fetchCmd.Run() // Ignore errors, might not have remote

	// Check if branch already exists
	checkCmd := exec.Command("git", "rev-parse", "--verify", branchName)
	checkCmd.Dir = repoPath
	if checkCmd.Run() == nil {
		// Branch exists, check it out
		checkoutCmd := exec.Command("git", "checkout", branchName)
		checkoutCmd.Dir = repoPath
		return checkoutCmd.Run()
	}

	// Create and checkout new branch
	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}

func (s *Spawner) startClaudeSession(repoPath, prompt, sessionID string) error {
	// Create a session file with the prompt
	sessionDir := filepath.Join(getConfigDir(), "active_sessions")
	os.MkdirAll(sessionDir, 0755)

	promptFile := filepath.Join(sessionDir, sessionID+".prompt")
	if err := os.WriteFile(promptFile, []byte(prompt), 0644); err != nil {
		return err
	}

	// Start Claude Code in a detached process
	// Using nohup to keep it running after parent exits
	script := fmt.Sprintf(`cd "%s" && %s --print "%s" > "%s/%s.log" 2>&1 &`,
		repoPath, s.claudeBinary(), prompt, sessionDir, sessionID)

	cmd := exec.Command("bash", "-c", script)
	cmd.Dir = repoPath

	if err := cmd.Start(); err != nil {
		return err
	}

	// Don't wait for it - let it run in background
	go cmd.Wait()

	return nil
}

func (s *Spawner) ResumeSession(session *Session) error {
	// Change to repo directory and open Claude in interactive mode
	fmt.Printf("Resuming session in %s on branch %s\n", session.RepoPath, session.Branch)
	fmt.Printf("Task: %s\n\n", session.Summary)

	// Switch to the branch
	checkoutCmd := exec.Command("git", "checkout", session.Branch)
	checkoutCmd.Dir = session.RepoPath
	if err := checkoutCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	// Start interactive Claude session
	cmd := exec.Command(s.claudeBinary(), "--resume")
	cmd.Dir = session.RepoPath
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
