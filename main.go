// Copyright 2025-2026 Divy Srivastava <dj.srivastava23@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

var useHappy bool

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Check for --happy flag
	args := []string{}
	for _, arg := range os.Args[1:] {
		if arg == "--happy" {
			useHappy = true
		} else {
			args = append(args, arg)
		}
	}

	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "config":
		runConfig()
	case "check":
		runCheck()
	case "start":
		runDaemon()
	case "sessions":
		runSessions()
	case "open":
		if len(args) < 2 {
			fmt.Println("Usage: autowork open [--happy] <session-id>")
			os.Exit(1)
		}
		runOpen(args[1])
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`autowork - Automatic Work Scheduler

Usage:
  autowork config           Configure Slack and OpenAI credentials
  autowork check [--happy]  Check for new messages once
  autowork start [--happy]  Run as daemon (polls continuously)
  autowork sessions         List pending work sessions
  autowork open [--happy] <id>  Resume a work session

Flags:
  --happy  Use Happy Coder (remote UI) instead of Claude Code`)
}

func runConfig() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Autowork Configuration")
	fmt.Println("======================")
	fmt.Println()
	fmt.Println("To get your Slack credentials:")
	fmt.Println("1. Open Slack in your browser")
	fmt.Println("2. Open DevTools (F12) -> Application -> Cookies")
	fmt.Println("3. Copy the 'd' cookie value")
	fmt.Println("4. In Network tab, find any API call and copy the 'token' parameter (starts with xoxc-)")
	fmt.Println()

	cfg := &Config{
		ReposDir:            os.ExpandEnv("$HOME/gh"),
		PollIntervalSeconds: 300,
	}

	fmt.Print("Slack token (xoxc-...): ")
	cfg.SlackToken, _ = reader.ReadString('\n')
	cfg.SlackToken = strings.TrimSpace(cfg.SlackToken)

	fmt.Print("Slack 'd' cookie (xoxd-...): ")
	cfg.SlackCookie, _ = reader.ReadString('\n')
	cfg.SlackCookie = strings.TrimSpace(cfg.SlackCookie)

	fmt.Print("Repos directory [" + cfg.ReposDir + "]: ")
	reposDir, _ := reader.ReadString('\n')
	reposDir = strings.TrimSpace(reposDir)
	if reposDir != "" {
		cfg.ReposDir = reposDir
	}

	fmt.Print("Channel IDs to watch (comma-separated): ")
	channels, _ := reader.ReadString('\n')
	channels = strings.TrimSpace(channels)
	cfg.ChannelIDs = strings.Split(channels, ",")
	for i := range cfg.ChannelIDs {
		cfg.ChannelIDs[i] = strings.TrimSpace(cfg.ChannelIDs[i])
	}

	fmt.Print("Poll interval in seconds [300]: ")
	interval, _ := reader.ReadString('\n')
	interval = strings.TrimSpace(interval)
	if interval != "" {
		if i, err := strconv.Atoi(interval); err == nil {
			cfg.PollIntervalSeconds = i
		}
	}

	if err := SaveConfig(cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Configuration saved to ~/.autowork/config.json")
}

func runCheck() {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	state, err := LoadState()
	if err != nil {
		fmt.Printf("Error loading state: %v\n", err)
		os.Exit(1)
	}

	processMessages(cfg, state)
}

func runDaemon() {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	state, err := LoadState()
	if err != nil {
		fmt.Printf("Error loading state: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Starting autowork daemon (polling every %d seconds)\n", cfg.PollIntervalSeconds)
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	ticker := time.NewTicker(time.Duration(cfg.PollIntervalSeconds) * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	processMessages(cfg, state)

	for range ticker.C {
		processMessages(cfg, state)
	}
}

func processMessages(cfg *Config, state *State) {
	slack := NewSlackClient(cfg.SlackToken, cfg.SlackCookie)
	classifier := NewClassifier(cfg.ReposDir)
	spawner := NewSpawner(cfg.ReposDir, useHappy)

	fmt.Printf("[%s] Checking for new messages...\n", time.Now().Format("15:04:05"))

	messages, err := slack.FetchNewMessages(cfg, state)
	if err != nil {
		fmt.Printf("Error fetching messages: %v\n", err)
		return
	}

	if err := SaveState(state); err != nil {
		fmt.Printf("Error saving state: %v\n", err)
	}

	if len(messages) == 0 {
		fmt.Println("No new messages")
		return
	}

	fmt.Printf("Found %d new messages\n", len(messages))

	for _, msg := range messages {
		fmt.Printf("\nProcessing: %s\n", truncate(msg.Text, 50))

		result, err := classifier.Classify(msg.Text)
		if err != nil {
			fmt.Printf("  Classification error: %v\n", err)
			continue
		}

		if !result.Actionable {
			fmt.Println("  -> Not actionable, skipping")
			continue
		}

		fmt.Printf("  -> Actionable! Repo: %s, Branch: %s\n", result.Repo, result.Branch)
		fmt.Printf("  -> Summary: %s\n", result.Summary)

		session, err := spawner.SpawnSession(result, msg.Text)
		if err != nil {
			fmt.Printf("  -> Error spawning session: %v\n", err)
			continue
		}

		fmt.Printf("  -> Session started: %s\n", session.ID)
	}
}

func runSessions() {
	store, err := LoadSessions()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if len(store.Sessions) == 0 {
		fmt.Println("No pending sessions")
		return
	}

	fmt.Println("Pending work sessions:")
	fmt.Println()

	for _, s := range store.Sessions {
		fmt.Printf("  [%s] %s/%s\n", s.ID, s.Repo, s.Branch)
		fmt.Printf("        %s\n", s.Summary)
		fmt.Printf("        Created: %s\n", s.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Println()
	}

	fmt.Println("Run 'autowork open <id>' to resume a session")
}

func runOpen(sessionID string) {
	store, err := LoadSessions()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	var session *Session
	for _, s := range store.Sessions {
		if s.ID == sessionID {
			session = &s
			break
		}
	}

	if session == nil {
		fmt.Printf("Session not found: %s\n", sessionID)
		os.Exit(1)
	}

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	spawner := NewSpawner(cfg.ReposDir, useHappy)
	if err := spawner.ResumeSession(session); err != nil {
		fmt.Printf("Error resuming session: %v\n", err)
		os.Exit(1)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
