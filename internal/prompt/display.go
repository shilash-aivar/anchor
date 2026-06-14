package prompt

import (
	"fmt"
	"os"
	"strings"

	"anchor/internal/config"
	"anchor/internal/session"
)

type Info struct {
	Project   string `json:"project"`
	Tier      string `json:"tier"`
	Namespace string `json:"namespace"`
	Context   string `json:"context"`
	Cluster   string `json:"cluster"`
	AccountID string `json:"account_id,omitempty"`
	Profile   string `json:"aws_profile,omitempty"`
	Region    string `json:"aws_region,omitempty"`
}

func FromState(s *session.State, p *config.Project) Info {
	info := Info{
		Project:   s.Project,
		Tier:      config.NormalizeTier(s.Tier),
		Namespace: s.Namespace,
		Context:   s.KubeContext,
		AccountID: s.AccountID,
		Profile:   s.AWSProfile,
		Region:    s.AWSRegion,
	}
	if p != nil {
		info.Cluster = p.Cluster
	}
	return info
}

func Load() (*Info, error) {
	s, err := session.Load()
	if err != nil || s == nil {
		return nil, err
	}
	p, _ := config.LoadProject(s.Project)
	info := FromState(s, p)
	return &info, nil
}

// Plain is a single-line context string (no escape codes).
func Plain(info Info) string {
	badge := config.TierAbbrev(info.Tier)
	acct := accountLabel(info)
	cluster := clusterLabel(info)
	return fmt.Sprintf("%s %s · %s · %s · %s", badge, info.Project, acct, cluster, info.Namespace)
}

// PlainCompact shortens account to last 4 digits when numeric.
func PlainCompact(info Info) string {
	badge := config.TierAbbrev(info.Tier)
	acct := accountLabel(info)
	if len(acct) > 4 && isDigits(acct) {
		acct = "…" + acct[len(acct)-4:]
	}
	cluster := clusterLabel(info)
	return fmt.Sprintf("%s %s · %s · %s · %s", badge, info.Project, acct, cluster, info.Namespace)
}

func accountLabel(info Info) string {
	if info.AccountID != "" {
		return info.AccountID
	}
	if info.Profile != "" {
		return info.Profile
	}
	return "?"
}

func clusterLabel(info Info) string {
	if info.Cluster != "" {
		return info.Cluster
	}
	if info.Context != "" {
		return info.Context
	}
	return "?"
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}

// StarshipSegment prints Starship custom module output for info.
func StarshipSegment(info Info) {
	tier := config.NormalizeTier(info.Tier)
	style := tierStyleStarship(tier)
	if config.IsProductionTier(tier) {
		fmt.Println("[#anchor]")
		fmt.Println("style = 'bold red'")
	}
	text := PlainCompact(info)
	fmt.Printf("[%s](%s)", text, style)
}

// ZshRPrompt returns zsh RPROMPT segment with colors (shown on the right).
func ZshRPrompt(info Info) {
	color := tierColorZsh(info.Tier)
	badge := config.TierAbbrev(info.Tier)
	acct := accountLabel(info)
	cluster := clusterLabel(info)
	fmt.Printf("%s[%s] %s · %s · %s · ns:%s%%f", color, badge, info.Project, acct, cluster, info.Namespace)
}

func tierColorZsh(tier string) string {
	switch config.NormalizeTier(tier) {
	case "production":
		return "%F{1}"
	case "staging":
		return "%F{3}"
	default:
		return "%F{6}"
	}
}

func tierStyleStarship(tier string) string {
	switch config.NormalizeTier(tier) {
	case "production":
		return "bold red"
	case "staging":
		return "bold yellow"
	default:
		return "bold cyan"
	}
}

// ZshLeft returns a left-prompt segment (after %~ path), git-branch style.
func ZshLeft(info Info) {
	color := tierColorZsh(info.Tier)
	badge := config.TierAbbrev(info.Tier)
	acct := accountLabel(info)
	cluster := clusterLabel(info)
	fmt.Printf("%s[%s] (%s · %s · %s · %s)%%f", color, badge, info.Project, acct, cluster, info.Namespace)
}

// BashPS1 returns bash PS1 fragment with ANSI colors.
func BashPS1(info Info) {
	reset := "\033[0m"
	color := bashTierColor(info.Tier)
	badge := config.TierAbbrev(info.Tier)
	acct := accountLabel(info)
	cluster := clusterLabel(info)
	fmt.Printf("%s[%s] (%s · %s · %s · %s)%s", color, badge, info.Project, acct, cluster, info.Namespace, reset)
}

func bashTierColor(tier string) string {
	switch config.NormalizeTier(tier) {
	case "production":
		return "\033[1;31m"
	case "staging":
		return "\033[1;33m"
	default:
		return "\033[1;36m"
	}
}

// ParseMarker reads the active marker file (fast path for shell hooks).
func ParseMarker(data string) (Info, bool) {
	parts := strings.Split(strings.TrimSpace(data), "|")
	if len(parts) < 3 {
		return Info{}, false
	}
	info := Info{
		Project:   parts[0],
		Context:   parts[1],
		Namespace: parts[2],
	}
	if len(parts) > 3 {
		info.AccountID = parts[3]
	}
	if len(parts) > 4 {
		info.Cluster = parts[4]
	}
	if len(parts) > 5 {
		info.Tier = config.NormalizeTier(parts[5])
	}
	if len(parts) > 6 {
		info.Profile = parts[6]
	}
	if len(parts) > 7 {
		info.Region = parts[7]
	}
	return info, true
}

func LoadFromMarker() (*Info, error) {
	path, err := session.ActiveMarkerPath()
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
	info, ok := ParseMarker(string(data))
	if !ok {
		return nil, nil
	}
	return &info, nil
}
