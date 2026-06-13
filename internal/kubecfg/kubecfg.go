package kubecfg

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"anchor/internal/config"
)

type ContextInfo struct {
	Name      string
	Cluster   string
	Namespace string
}

func DefaultKubeconfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kube", "config")
}

func ListContexts(kubeconfig string) ([]ContextInfo, error) {
	if kubeconfig == "" {
		kubeconfig = DefaultKubeconfigPath()
	}
	cmd := exec.Command("kubectl", "config", "get-contexts", "-o", "name")
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list contexts: %w", err)
	}
	var contexts []ContextInfo
	for _, name := range strings.Fields(string(out)) {
		ns := contextNamespace(kubeconfig, name)
		cluster := contextCluster(kubeconfig, name)
		contexts = append(contexts, ContextInfo{Name: name, Cluster: cluster, Namespace: ns})
	}
	return contexts, nil
}

func contextNamespace(kubeconfig, name string) string {
	cmd := exec.Command("kubectl", "config", "view", "--context="+name, "--minify", "-o", "jsonpath={.contexts[0].context.namespace}")
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

func contextCluster(kubeconfig, name string) string {
	cmd := exec.Command("kubectl", "config", "view", "--context="+name, "--minify", "-o", "jsonpath={.contexts[0].context.cluster}")
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

type LintIssue struct {
	Level   string
	Project string
	Message string
}

func LintAll() ([]LintIssue, error) {
	var issues []LintIssue
	projects, err := config.LoadAllProjects()
	if err != nil {
		return nil, err
	}
	aliases := map[string]string{}
	for _, p := range projects {
		alias := p.EffectiveContextAlias()
		if prev, ok := aliases[alias]; ok {
			issues = append(issues, LintIssue{
				Level: "warn", Project: p.Name,
				Message: fmt.Sprintf("duplicate context alias %q (also used by %s)", alias, prev),
			})
		}
		aliases[alias] = p.Name

		kubePath, _ := config.KubeconfigPath(p.Name)
		if _, err := os.Stat(kubePath); os.IsNotExist(err) {
			issues = append(issues, LintIssue{
				Level: "info", Project: p.Name,
				Message: "kubeconfig not generated yet — run project use",
			})
		}

		if p.AWSProfile == "" {
			issues = append(issues, LintIssue{Level: "error", Project: p.Name, Message: "missing aws_profile"})
		}
		if p.Cluster == "" {
			issues = append(issues, LintIssue{Level: "error", Project: p.Name, Message: "missing cluster"})
		}
		if p.Region == "" {
			issues = append(issues, LintIssue{Level: "warn", Project: p.Name, Message: "missing region"})
		}
	}

	dir, err := config.ConfigDir()
	if err != nil {
		return issues, err
	}
	kubeDir := filepath.Join(dir, config.KubeDirName)
	entries, _ := os.ReadDir(kubeDir)
	projectSet := map[string]bool{}
	for _, p := range projects {
		projectSet[p.Name] = true
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		if !projectSet[name] {
			issues = append(issues, LintIssue{
				Level: "warn", Project: name,
				Message: "orphan kubeconfig without project definition — run anchor prune",
			})
		}
	}
	return issues, nil
}

func PruneOrphans(dryRun bool) ([]string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return nil, err
	}
	names, err := config.ListProjects()
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, n := range names {
		set[n] = true
	}
	kubeDir := filepath.Join(dir, config.KubeDirName)
	entries, err := os.ReadDir(kubeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var removed []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		if set[name] {
			continue
		}
		path := filepath.Join(kubeDir, e.Name())
		if !dryRun {
			if err := os.Remove(path); err != nil {
				return removed, err
			}
		}
		removed = append(removed, path)
	}
	return removed, nil
}

func CopyContextToProject(kubeconfig, contextName, projectName string) error {
	dest, err := config.KubeconfigPath(projectName)
	if err != nil {
		return err
	}
	if _, err := config.EnsureConfigDir(); err != nil {
		return err
	}
	cmd := exec.Command("kubectl", "config", "view", "--context="+contextName, "--minify", "--flatten")
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("export context: %w", err)
	}
	return os.WriteFile(dest, out.Bytes(), 0o600)
}
