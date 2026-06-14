package awsx

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type CredentialStatus struct {
	Valid      bool
	ExpiresAt  *time.Time
	ExpiresIn  string
	Hint       string
	Account    string
	ARN        string
}

// CredentialStatusForProfile checks STS and SSO cache expiry for a profile.
func CredentialStatusForProfile(profile string) CredentialStatus {
	id, err := GetCallerIdentity(profile)
	if err != nil {
		return CredentialStatus{
			Valid: false,
			Hint:  fmt.Sprintf("run `anchor login %s`", profile),
		}
	}
	st := CredentialStatus{
		Valid:   true,
		Account: id.Account,
		ARN:     id.ARN,
	}
	exp, ok := ssoCacheExpiryForProfile(profile)
	if !ok {
		st.ExpiresIn = "valid"
		return st
	}
	remaining := time.Until(exp)
	if remaining <= 0 {
		st.Valid = false
		st.ExpiresAt = &exp
		st.Hint = fmt.Sprintf("SSO session expired — run `anchor login %s`", profile)
		return st
	}
	st.ExpiresAt = &exp
	if remaining < 30*time.Minute {
		st.ExpiresIn = fmt.Sprintf("expires in %dm", int(remaining.Minutes()))
		st.Hint = fmt.Sprintf("SSO %s — run `anchor login %s` soon", st.ExpiresIn, profile)
	} else {
		st.ExpiresIn = fmt.Sprintf("valid for %s", remaining.Round(time.Minute))
	}
	return st
}

func ssoCacheExpiryForProfile(profile string) (time.Time, bool) {
	startURL, err := profileConfigValue(profile, "sso_start_url")
	if err != nil || startURL == "" {
		return ssoCacheExpiryLatest()
	}
	return ssoCacheExpiryMatching(startURL)
}

func profileConfigValue(profile, key string) (string, error) {
	args := []string{"configure", "get", key}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	out, err := exec.Command("aws", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func ssoCacheExpiryMatching(startURL string) (time.Time, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return time.Time{}, false
	}
	cacheDir := filepath.Join(home, ".aws", "sso", "cache")
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return time.Time{}, false
	}
	var best time.Time
	found := false
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(cacheDir, e.Name()))
		if err != nil {
			continue
		}
		var payload struct {
			StartURL  string `json:"startUrl"`
			ExpiresAt string `json:"expiresAt"`
		}
		if err := json.Unmarshal(data, &payload); err != nil || payload.ExpiresAt == "" {
			continue
		}
		if payload.StartURL != "" && payload.StartURL != startURL {
			continue
		}
		exp, err := time.Parse(time.RFC3339, payload.ExpiresAt)
		if err != nil {
			continue
		}
		if !found || exp.After(best) {
			best = exp
			found = true
		}
	}
	return best, found
}

// ssoCacheExpiryLatest is a fallback when profile SSO URL is unknown.
func ssoCacheExpiryLatest() (time.Time, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return time.Time{}, false
	}
	cacheDir := filepath.Join(home, ".aws", "sso", "cache")
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return time.Time{}, false
	}
	var latest time.Time
	found := false
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(cacheDir, e.Name()))
		if err != nil {
			continue
		}
		var payload struct {
			ExpiresAt string `json:"expiresAt"`
		}
		if err := json.Unmarshal(data, &payload); err != nil || payload.ExpiresAt == "" {
			continue
		}
		exp, err := time.Parse(time.RFC3339, payload.ExpiresAt)
		if err != nil {
			continue
		}
		if !found || exp.After(latest) {
			latest = exp
			found = true
		}
	}
	return latest, found
}

// NeedsLogin returns true when credentials are missing or expiring within warnWindow.
func NeedsLogin(profile string, warnWindow time.Duration) bool {
	st := CredentialStatusForProfile(profile)
	if !st.Valid {
		return true
	}
	if st.ExpiresAt != nil && time.Until(*st.ExpiresAt) < warnWindow {
		return true
	}
	return false
}
