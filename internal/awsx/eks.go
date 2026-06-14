package awsx

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type EKSCluster struct {
	Name   string
	Region string
}

func ListEKSClusters(profile, region string) ([]string, error) {
	args := []string{"eks", "list-clusters", "--region", region, "--output", "json"}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	out, err := exec.Command("aws", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("list EKS clusters (profile=%q region=%q): %w", profile, region, err)
	}
	var resp struct {
		Clusters []string `json:"clusters"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, err
	}
	return resp.Clusters, nil
}

func ListProfiles() ([]string, error) {
	out, err := exec.Command("aws", "configure", "list-profiles").Output()
	if err != nil {
		return nil, fmt.Errorf("aws configure list-profiles: %w", err)
	}
	var profiles []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			profiles = append(profiles, line)
		}
	}
	return profiles, nil
}

func ProfileRegion(profile string) string {
	out, err := exec.Command("aws", "configure", "get", "region", "--profile", profile).Output()
	if err != nil {
		return "us-east-1"
	}
	r := strings.TrimSpace(string(out))
	if r == "" {
		return "us-east-1"
	}
	return r
}

func ProfileAccountID(profile string) (string, error) {
	id, err := GetCallerIdentity(profile)
	if err != nil {
		return "", err
	}
	return id.Account, nil
}

func SSOLogout(profile string) error {
	args := []string{"sso", "logout"}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	cmd := exec.Command("aws", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func IsSSOProfile(profile string) bool {
	v, err := profileConfigValue(profile, "sso_start_url")
	return err == nil && v != ""
}
