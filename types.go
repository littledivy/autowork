package main

import "time"

type Config struct {
	SlackToken          string   `json:"slack_token"`
	SlackCookie         string   `json:"slack_cookie"`
	ReposDir            string   `json:"repos_dir"`
	PollIntervalSeconds int      `json:"poll_interval_seconds"`
	ChannelIDs          []string `json:"channel_ids"`
}

type State struct {
	LastTimestamps map[string]string `json:"last_timestamps"` // channel_id -> timestamp
}

type SlackMessage struct {
	User      string `json:"user"`
	Text      string `json:"text"`
	Timestamp string `json:"ts"`
	Channel   string `json:"channel"`
}

type ClassificationResult struct {
	Actionable bool   `json:"actionable"`
	Repo       string `json:"repo,omitempty"`
	Summary    string `json:"summary,omitempty"`
	Branch     string `json:"branch,omitempty"`
}

type Session struct {
	ID        string    `json:"id"`
	Repo      string    `json:"repo"`
	Branch    string    `json:"branch"`
	Summary   string    `json:"summary"`
	RepoPath  string    `json:"repo_path"`
	CreatedAt time.Time `json:"created_at"`
	SlackMsg  string    `json:"slack_msg"`
}

type SessionStore struct {
	Sessions []Session `json:"sessions"`
}
