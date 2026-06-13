package hooks

import (
	"fmt"
	"os"
	"os/exec"

	"ctxly/internal/config"
	"ctxly/internal/session"
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
		"CTXLY_PROJECT="+s.Project,
		"CTXLY_TIER="+s.Tier,
		"CTXLY_NAMESPACE="+s.Namespace,
		"CTXLY_KUBE_CONTEXT="+s.KubeContext,
		"CTXLY_AWS_PROFILE="+s.AWSProfile,
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
