package discover

import (
	"fmt"
	"strings"

	"anchor/internal/awsx"
	"anchor/internal/config"
)

type Candidate struct {
	Profile       string
	Region        string
	AccountID     string
	Cluster       string
	SuggestedName string
	Exists        bool
	NeedsLogin    bool
	VPNRequired   bool
}

func ScanProfiles() ([]Candidate, error) {
	profiles, err := awsx.ListProfiles()
	if err != nil {
		return nil, err
	}
	existing, _ := config.ListProjects()
	existSet := map[string]bool{}
	for _, n := range existing {
		existSet[n] = true
	}

	var out []Candidate
	seen := map[string]bool{}
	usedNames := map[string]bool{}
	for _, profile := range profiles {
		region := awsx.ProfileRegion(profile)
		accountID, err := awsx.ProfileAccountID(profile)
		if err != nil {
			key := profile + "|" + region + "|login"
			if !seen[key] {
				seen[key] = true
				out = append(out, Candidate{
					Profile:       profile,
					Region:        region,
					SuggestedName: uniqueName(suggestName(profile, ""), existSet, usedNames),
					NeedsLogin:    true,
				})
			}
			continue
		}
		clusters, err := awsx.ListEKSClusters(profile, region)
		if err != nil || len(clusters) == 0 {
			key := profile + "|" + region + "|"
			if !seen[key] {
				seen[key] = true
				name := uniqueName(suggestName(profile, ""), existSet, usedNames)
				usedNames[name] = true
				out = append(out, Candidate{
					Profile:       profile,
					Region:        region,
					AccountID:     accountID,
					SuggestedName: name,
					Exists:        existSet[name],
				})
			}
			continue
		}
		for _, cluster := range clusters {
			key := profile + "|" + region + "|" + cluster
			if seen[key] {
				continue
			}
			seen[key] = true
			base := suggestName(profile, cluster)
			name := uniqueName(base, existSet, usedNames)
			if existSet[name] && cluster != "" {
				name = uniqueName(base+"-"+clusterSuffix(cluster), existSet, usedNames)
			}
			usedNames[name] = true
			vpnRequired := false
			if private, err := awsx.EKSClusterPrivateOnly(profile, region, cluster); err == nil && private {
				vpnRequired = true
			}
			out = append(out, Candidate{
				Profile:       profile,
				Region:        region,
				AccountID:     accountID,
				Cluster:       cluster,
				SuggestedName: name,
				Exists:        existSet[name],
				VPNRequired:   vpnRequired,
			})
		}
	}
	return out, nil
}

func suggestName(profile, cluster string) string {
	base := profile
	if cluster != "" {
		parts := strings.Split(cluster, "-")
		if len(parts) > 0 && parts[0] != "" {
			base = parts[0]
		} else {
			base = cluster
		}
	}
	base = strings.NewReplacer("_", "-", ".", "-").Replace(base)
	return strings.ToLower(base)
}

func clusterSuffix(cluster string) string {
	parts := strings.Split(cluster, "-")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	if len(cluster) > 12 {
		return cluster[len(cluster)-8:]
	}
	return cluster
}

func uniqueName(base string, existSet, usedNames map[string]bool) string {
	name := base
	for i := 2; existSet[name] || usedNames[name]; i++ {
		name = fmt.Sprintf("%s-%d", base, i)
	}
	return name
}

func TierFromName(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "prod"), strings.Contains(n, "production"), strings.Contains(n, "live"):
		return "production"
	case strings.Contains(n, "stg"), strings.Contains(n, "staging"), strings.Contains(n, "stage"):
		return "staging"
	default:
		return "development"
	}
}

func ValidateCandidate(c Candidate, requireCluster bool) error {
	if c.Profile == "" {
		return fmt.Errorf("profile required")
	}
	if c.Region == "" {
		return fmt.Errorf("region required")
	}
	if requireCluster && c.Cluster == "" {
		return fmt.Errorf("cluster required")
	}
	return nil
}
