package awsx

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type SSOImportOptions struct {
	StartURL string
	SSORegion string
	Region    string
	Bootstrap string
	DryRun    bool
}

type SSOImportResult struct {
	Created []string
	Skipped []string
}

func ImportSSOProfiles(opts SSOImportOptions) (*SSOImportResult, error) {
	if !AWSAvailable() {
		return nil, fmt.Errorf("aws CLI not found")
	}
	bootstrap := opts.Bootstrap
	if bootstrap == "" {
		var err error
		bootstrap, err = FindSSOBootstrapProfile(opts.StartURL)
		if err != nil {
			return nil, err
		}
	}
	startURL := opts.StartURL
	if startURL == "" {
		startURL, _ = profileSSOStartURL(bootstrap)
	}
	ssoRegion := opts.SSORegion
	if ssoRegion == "" {
		ssoRegion, _ = profileConfigValue(bootstrap, "sso_region")
	}
	if ssoRegion == "" {
		ssoRegion = "us-east-1"
	}
	defaultRegion := opts.Region
	if defaultRegion == "" {
		defaultRegion = ProfileRegion(bootstrap)
	}

	fmt.Fprintf(os.Stderr, "→ SSO login bootstrap profile %q\n", bootstrap)
	if err := SSOLogin(bootstrap); err != nil {
		return nil, err
	}

	accounts, err := listSSOAccounts(bootstrap, ssoRegion)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no SSO accounts returned — check sso.start_url in anchor config")
	}

	res := &SSOImportResult{}
	existing, _ := ListProfiles()
	existSet := map[string]bool{}
	for _, p := range existing {
		existSet[p] = true
	}

	for _, acct := range accounts {
		roles, err := listSSOAccountRoles(bootstrap, ssoRegion, acct.AccountID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ account %s: %v\n", acct.AccountID, err)
			continue
		}
		for _, role := range roles {
			name := sanitizeProfileName(acct.AccountName, role.RoleName)
			if name == "" {
				name = sanitizeProfileName(acct.AccountID, role.RoleName)
			}
			if existSet[name] {
				res.Skipped = append(res.Skipped, name)
				continue
			}
			block := buildSSOProfileBlock(name, startURL, ssoRegion, acct.AccountID, role.RoleName, defaultRegion)
			if opts.DryRun {
				fmt.Printf("would create profile %q (account=%s role=%s)\n", name, acct.AccountID, role.RoleName)
				res.Created = append(res.Created, name)
				continue
			}
			if err := appendAWSConfig(block); err != nil {
				return res, err
			}
			existSet[name] = true
			res.Created = append(res.Created, name)
			fmt.Printf("✓ profile %s (account=%s role=%s)\n", name, acct.AccountID, role.RoleName)
		}
	}
	return res, nil
}

func FindSSOBootstrapProfile(startURL string) (string, error) {
	profiles, err := ListProfiles()
	if err != nil {
		return "", err
	}
	var fallback string
	for _, p := range profiles {
		if !IsSSOProfile(p) {
			continue
		}
		url, _ := profileSSOStartURL(p)
		if startURL == "" || url == startURL {
			return p, nil
		}
		if fallback == "" {
			fallback = p
		}
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", fmt.Errorf("no SSO profile in ~/.aws/config — set sso.start_url in anchor config and run `aws configure sso`, or pass --profile")
}

func profileSSOStartURL(profile string) (string, error) {
	if url, err := profileConfigValue(profile, "sso_start_url"); err == nil && url != "" {
		return url, nil
	}
	sessionName, err := profileConfigValue(profile, "sso_session")
	if err != nil || sessionName == "" {
		return "", fmt.Errorf("no sso_start_url for profile %q", profile)
	}
	return sessionStartURL(sessionName)
}

func sessionStartURL(sessionName string) (string, error) {
	path, err := awsConfigPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	section := fmt.Sprintf("[sso-session %s]", sessionName)
	inSection := false
	for _, line := range strings.Split(string(data), "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "[") {
			inSection = trim == section
			continue
		}
		if inSection && strings.HasPrefix(trim, "sso_start_url") {
			parts := strings.SplitN(trim, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}
	return "", fmt.Errorf("sso-session %q not found in ~/.aws/config", sessionName)
}

type ssoAccount struct {
	AccountID   string `json:"accountId"`
	AccountName string `json:"accountName"`
}

type ssoRole struct {
	RoleName string `json:"roleName"`
}

func listSSOAccounts(profile, ssoRegion string) ([]ssoAccount, error) {
	args := []string{"sso", "list-accounts", "--region", ssoRegion, "--output", "json", "--profile", profile}
	out, err := exec.Command("aws", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("aws sso list-accounts: %w", err)
	}
	var resp struct {
		AccountList []ssoAccount `json:"accountList"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, err
	}
	return resp.AccountList, nil
}

func listSSOAccountRoles(profile, ssoRegion, accountID string) ([]ssoRole, error) {
	args := []string{
		"sso", "list-account-roles",
		"--account-id", accountID,
		"--region", ssoRegion,
		"--output", "json",
		"--profile", profile,
	}
	out, err := exec.Command("aws", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("aws sso list-account-roles: %w", err)
	}
	var resp struct {
		RoleList []ssoRole `json:"roleList"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, err
	}
	return resp.RoleList, nil
}

var profileNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeProfileName(parts ...string) string {
	raw := strings.Join(parts, "-")
	raw = strings.NewReplacer(" ", "-", "/", "-", "@", "-").Replace(raw)
	raw = profileNameSanitizer.ReplaceAllString(raw, "-")
	raw = strings.Trim(raw, "-")
	if len(raw) > 64 {
		raw = raw[:64]
	}
	return strings.ToLower(raw)
}

func buildSSOProfileBlock(name, startURL, ssoRegion, accountID, roleName, region string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n[profile %s]\n", name))
	if startURL != "" {
		b.WriteString(fmt.Sprintf("sso_start_url = %s\n", startURL))
	}
	b.WriteString(fmt.Sprintf("sso_region = %s\n", ssoRegion))
	b.WriteString(fmt.Sprintf("sso_account_id = %s\n", accountID))
	b.WriteString(fmt.Sprintf("sso_role_name = %s\n", roleName))
	b.WriteString(fmt.Sprintf("region = %s\n", region))
	return b.String()
}

func appendAWSConfig(block string) error {
	path, err := awsConfigPath()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(block); err != nil {
		return err
	}
	return nil
}

func awsConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".aws", "config"), nil
}
