package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type SlackClient struct {
	token  string
	cookie string
	client *http.Client
}

func NewSlackClient(token, cookie string) *SlackClient {
	return &SlackClient{
		token:  token,
		cookie: cookie,
		client: &http.Client{},
	}
}

type slackHistoryResponse struct {
	OK       bool `json:"ok"`
	Messages []struct {
		User    string `json:"user"`
		Text    string `json:"text"`
		TS      string `json:"ts"`
		Type    string `json:"type"`
		Subtype string `json:"subtype,omitempty"`
	} `json:"messages"`
	Error string `json:"error,omitempty"`
}

func (s *SlackClient) FetchMessages(channelID string, oldest string) ([]SlackMessage, error) {
	endpoint := "https://slack.com/api/conversations.history"

	// xoxc- tokens require POST with form data
	params := url.Values{}
	params.Set("token", s.token)
	params.Set("channel", channelID)
	params.Set("limit", "50")
	if oldest != "" {
		params.Set("oldest", oldest)
	}

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", "d="+s.cookie)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result slackHistoryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("slack API error: %s", result.Error)
	}

	var messages []SlackMessage
	for _, msg := range result.Messages {
		// Skip bot messages, system messages, etc.
		if msg.Type != "message" || msg.Subtype != "" {
			continue
		}
		messages = append(messages, SlackMessage{
			User:      msg.User,
			Text:      msg.Text,
			Timestamp: msg.TS,
			Channel:   channelID,
		})
	}

	return messages, nil
}

func (s *SlackClient) FetchNewMessages(cfg *Config, state *State) ([]SlackMessage, error) {
	var allMessages []SlackMessage

	for _, channelID := range cfg.ChannelIDs {
		oldest := state.LastTimestamps[channelID]

		messages, err := s.FetchMessages(channelID, oldest)
		if err != nil {
			return nil, fmt.Errorf("error fetching channel %s: %w", channelID, err)
		}

		// Update state with newest timestamp
		for _, msg := range messages {
			if msg.Timestamp > state.LastTimestamps[channelID] {
				state.LastTimestamps[channelID] = msg.Timestamp
			}
		}

		allMessages = append(allMessages, messages...)
	}

	return allMessages, nil
}
