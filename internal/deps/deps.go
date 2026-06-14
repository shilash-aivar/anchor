package deps

import (
	"fmt"
	"os"
	"os/exec"
)

type Tool struct {
	Binary   string
	Brew     string
	Required bool
	Purpose  string
}

// All tools anchor wraps or relies on for daily ops.
var All = []Tool{
	{Binary: "aws", Brew: "awscli", Required: true, Purpose: "AWS SSO login and EKS kubeconfig"},
	{Binary: "kubectl", Brew: "kubectl", Required: true, Purpose: "Kubernetes cluster access"},
	{Binary: "fzf", Brew: "fzf", Required: false, Purpose: "Interactive project/namespace pickers"},
	{Binary: "stern", Brew: "stern", Required: false, Purpose: "anchor logs"},
	{Binary: "k9s", Brew: "k9s", Required: false, Purpose: "anchor ui"},
	{Binary: "helm", Brew: "helm", Required: false, Purpose: "anchor helm"},
}

type Status struct {
	Tool    Tool
	OK      bool
	Install string
}

func Available(binary string) bool {
	_, err := exec.LookPath(binary)
	return err == nil
}

func BrewAvailable() bool {
	_, err := exec.LookPath("brew")
	return err == nil
}

func CheckAll() []Status {
	out := make([]Status, 0, len(All))
	for _, t := range All {
		out = append(out, Status{
			Tool:    t,
			OK:      Available(t.Binary),
			Install: installHint(t),
		})
	}
	return out
}

func installHint(t Tool) string {
	if BrewAvailable() {
		return "brew install " + t.Brew
	}
	return "install " + t.Binary + " (see README)"
}

func Missing(requiredOnly bool) []Tool {
	var missing []Tool
	for _, t := range All {
		if Available(t.Binary) {
			continue
		}
		if requiredOnly && !t.Required {
			continue
		}
		missing = append(missing, t)
	}
	return missing
}

// InstallMissing installs absent tools via Homebrew. If requiredOnly, skips optional tools.
func InstallMissing(requiredOnly bool) error {
	if !BrewAvailable() {
		return fmt.Errorf("Homebrew not found — install from https://brew.sh then re-run")
	}
	missing := Missing(requiredOnly)
	if len(missing) == 0 {
		return nil
	}
	pkgs := make([]string, 0, len(missing))
	for _, t := range missing {
		pkgs = append(pkgs, t.Brew)
	}
	cmd := exec.Command("brew", append([]string{"install"}, pkgs...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew install: %w", err)
	}
	return nil
}

func BrewPackages() []string {
	pkgs := make([]string, 0, len(All))
	for _, t := range All {
		pkgs = append(pkgs, t.Brew)
	}
	return pkgs
}
