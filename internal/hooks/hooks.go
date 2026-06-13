package hooks

import (
	"fmt"
	"os"
	"os/exec"

	"anchor/internal/config"
	"anchor/internal/session"
)

func RunHook(path string, s *session.State) error {
	if path == "" {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("hook %q: %w", path, err)
	}
	cmd := exec.Command(path)
	cmd.Env = append(os.Environ(),
		"ANCHOR_PROJECT="+s.Project,
		"ANCHOR_TIER="+s.Tier,
		"ANCHOR_NAMESPACE="+s.Namespace,
		"ANCHOR_KUBE_CONTEXT="+s.KubeContext,
		"ANCHOR_AWS_PROFILE="+s.AWSProfile,
		"KUBECONFIG="+s.Kubeconfig,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func PreUse(s *session.State) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	return RunHook(cfg.Hooks.PreUse, s)
}

func PostUse(s *session.State) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	return RunHook(cfg.Hooks.PostUse, s)
}

func PreApply(s *session.State) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	return RunHook(cfg.Hooks.PreApply, s)
}
