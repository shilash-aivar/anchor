package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"ctxly/internal/config"
)

type RecentEntry struct {
	Project   string    `json:"project"`
	Namespace string    `json:"namespace"`
	UsedAt    time.Time `json:"used_at"`
}

const maxRecent = 10

func recentPath() (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "recent.json"), nil
}

func RecordRecent(s *State) error {
	if s == nil {
		return nil
	}
	path, err := recentPath()
	if err != nil {
		return err
	}

	var entries []RecentEntry
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &entries)
	}

	filtered := make([]RecentEntry, 0, len(entries)+1)
	for _, e := range entries {
		if e.Project == s.Project && e.Namespace == s.Namespace {
			continue
		}
		filtered = append(filtered, e)
	}
	entries = append([]RecentEntry{{
		Project:   s.Project,
		Namespace: s.Namespace,
		UsedAt:    time.Now().UTC(),
	}}, filtered...)
	if len(entries) > maxRecent {
		entries = entries[:maxRecent]
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	if _, err := config.EnsureConfigDir(); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func LoadRecent() ([]RecentEntry, error) {
	path, err := recentPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []RecentEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}
