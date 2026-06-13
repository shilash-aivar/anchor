package guard

import (
	"fmt"
	"os"
	"strings"

	"anchor/internal/config"
	"anchor/internal/session"
)

var dangerousPatterns = []string{
	"delete all",
	"delete --all",
	"delete -A",
	"delete namespace",
}

func CheckMutating(s *session.State, p *config.Project, cfg *config.Config, verb string, args []string) error {
	line := strings.ToLower(strings.Join(append([]string{verb}, args...), " "))

	if p != nil && p.ReadOnly {
		if isMutatingVerb(verb, args) {
			return fmt.Errorf("project %q is read-only — mutating commands blocked", p.Name)
		}
	}

	if cfg != nil && cfg.Options.BlockDangerous && p != nil && p.IsProduction() {
		for _, d := range dangerousPatterns {
			if strings.Contains(line, d) {
				return fmt.Errorf("blocked dangerous command on production tier: %s", d)
			}
		}
	}

	if cfg != nil && cfg.Options.AnnounceContext {
		fmt.Fprintf(os.Stderr, "[anchor] project=%s context=%s namespace=%s tier=%s\n",
			s.Project, s.KubeContext, s.Namespace, s.Tier)
	}
	return nil
}

func isMutatingVerb(verb string, args []string) bool {
	switch verb {
	case "apply", "delete", "patch", "replace", "edit", "scale", "rollout", "create":
		return true
	case "k", "kubectl":
		if len(args) == 0 {
			return false
		}
		return isMutatingVerb(args[0], args[1:])
	case "helm":
		if len(args) == 0 {
			return false
		}
		switch args[0] {
		case "upgrade", "install", "uninstall", "rollback", "delete":
			return true
		}
	}
	return false
}

func ConfirmApply(p *config.Project, cfg *config.Config) error {
	if p == nil || cfg == nil {
		return nil
	}
	if !p.IsProduction() || !cfg.Options.ConfirmProduction {
		return nil
	}
	return ConfirmProjectSwitch(p, cfg.Options.ConfirmProduction)
}
