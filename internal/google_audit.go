package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type googleAuditEvent struct {
	Time       time.Time `json:"time"`
	Action     string    `json:"action"`
	Account    string    `json:"account,omitempty"`
	Resource   string    `json:"resource,omitempty"`
	DryRun     bool      `json:"dry_run"`
	Operations []string  `json:"operations,omitempty"`
}

type googleAuditWriter struct {
	path string
}

func defaultGoogleAuditPath() string {
	if stateHome := os.Getenv("XDG_STATE_HOME"); stateHome != "" {
		return filepath.Join(stateHome, "wfctl", "plugins", "analytics", "google-audit.jsonl")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "state", "wfctl", "plugins", "analytics", "google-audit.jsonl")
}

func (w googleAuditWriter) Append(_ context.Context, event googleAuditEvent) error {
	if w.path == "" {
		return nil
	}
	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	}
	if err := os.MkdirAll(filepath.Dir(w.path), 0700); err != nil {
		return fmt.Errorf("analytics google audit: create audit dir: %w", err)
	}
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("analytics google audit: open audit file: %w", err)
	}
	defer f.Close()
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("analytics google audit: marshal event: %w", err)
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("analytics google audit: write event: %w", err)
	}
	return nil
}
