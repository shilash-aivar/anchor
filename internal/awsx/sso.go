package awsx

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CredentialStatus struct {
	Valid      bool
	ExpiresAt  *time.Time
	ExpiresIn  string
	Hint       string
}

// CredentialStatus checks whether the profile can call STS and estimates SSO expiry from cache.
func CredentialStatusForProfile(profile string) CredentialStatus {
	if _, err := GetCallerIdentity(profile); err != nil {
		return CredentialStatus{
			Valid: false,
			Hint:  fmt.Sprintf("run `anchor login %s`", profile),
		}
	}
	if exp, ok := ssoCacheExpiry(profile); ok {
		remaining := time.Until(exp)
		st := CredentialStatus{
			Valid:     true,
			ExpiresAt: &exp,
		}
		if remaining <= 0 {
			st.Valid = false
			st.Hint = fmt.Sprintf("SSO session expired — run `anchor login %s`", profile)
			return st
		}
		if remaining < 30*time.Minute {
			st.ExpiresIn = fmt.Sprintf("expires in %dm", int(remaining.Minutes()))
			st.Hint = fmt.Sprintf("SSO %s — run `anchor login %s` soon", st.ExpiresIn, profile)
		} else {
			st.ExpiresIn = fmt.Sprintf("valid for %s", remaining.Round(time.Minute))
		}
		return st
	}
	return CredentialStatus{Valid: true, ExpiresIn: "valid"}
}

func ssoCacheExpiry(profile string) (time.Time, bool) {
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
