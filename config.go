package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDir    = ".autowork"
	configFile   = "config.json"
	stateFile    = "state.json"
	sessionsFile = "sessions.json"
)

func getConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, configDir)
}

func ensureConfigDir() error {
	return os.MkdirAll(getConfigDir(), 0755)
}

func LoadConfig() (*Config, error) {
	path := filepath.Join(getConfigDir(), configFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found, run 'autowork config' first")
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	if err := ensureConfigDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(getConfigDir(), configFile)
	return os.WriteFile(path, data, 0600)
}

func LoadState() (*State, error) {
	path := filepath.Join(getConfigDir(), stateFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{LastTimestamps: make(map[string]string)}, nil
		}
		return nil, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	if state.LastTimestamps == nil {
		state.LastTimestamps = make(map[string]string)
	}
	return &state, nil
}

func SaveState(state *State) error {
	if err := ensureConfigDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(getConfigDir(), stateFile)
	return os.WriteFile(path, data, 0644)
}

func LoadSessions() (*SessionStore, error) {
	path := filepath.Join(getConfigDir(), sessionsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SessionStore{Sessions: []Session{}}, nil
		}
		return nil, err
	}
	var store SessionStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	return &store, nil
}

func SaveSessions(store *SessionStore) error {
	if err := ensureConfigDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(getConfigDir(), sessionsFile)
	return os.WriteFile(path, data, 0644)
}

func AddSession(session Session) error {
	store, err := LoadSessions()
	if err != nil {
		return err
	}
	store.Sessions = append(store.Sessions, session)
	return SaveSessions(store)
}
