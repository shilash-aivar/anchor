package awsx

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Identity struct {
	Account string `json:"Account"`
	ARN     string `json:"Arn"`
	UserID  string `json:"UserId"`
}

func SSOLogin(profile string) error {
	args := []string{"sso", "login"}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	cmd := exec.Command("aws", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func GetCallerIdentity(profile string) (*Identity, error) {
	args := []string{"sts", "get-caller-identity", "--output", "json"}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	out, err := exec.Command("aws", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("aws credentials invalid or expired (profile %q): run `anchor login`", profile)
	}
	var id Identity
	if err := json.Unmarshal(out, &id); err != nil {
		return nil, err
	}
	return &id, nil
}

func UpdateKubeconfig(profile, region, cluster, alias, kubeconfigPath string) error {
	args := []string{
		"eks", "update-kubeconfig",
		"--name", cluster,
		"--region", region,
		"--alias", alias,
		"--kubeconfig", kubeconfigPath,
	}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	out, err := exec.Command("aws", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("aws eks update-kubeconfig: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func AWSAvailable() bool {
	_, err := exec.LookPath("aws")
	return err == nil
}
