package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"anchor/internal/config"
)

type State struct {
	Project     string    `json:"project"`
	AWSProfile  string    `json:"aws_profile"`
	AWSRegion   string    `json:"aws_region"`
	AccountID   string    `json:"account_id,omitempty"`
	KubeContext string    `json:"kube_context"`
	Namespace   string    `json:"namespace"`
	Tier        string    `json:"tier"`
	Kubeconfig  string    `json:"kubeconfig"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func StatePath() (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, config.StateFile), nil
}

func ActiveMarkerPath() (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "active"), nil
}

func Load() (*State, error) {
	path, err := StatePath()
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
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	return &s, nil
}

func Save(s *State) error {
	if _, err := config.EnsureConfigDir(); err != nil {
		return err
	}
	s.UpdatedAt = time.Now().UTC()
	path, err := StatePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	marker, err := ActiveMarkerPath()
	if err != nil {
		return err
	}
	line := fmt.Sprintf("%s|%s|%s", s.Project, s.KubeContext, s.Namespace)
	return os.WriteFile(marker, []byte(line), 0o644)
}

func Clear() error {
	path, err := StatePath()
	if err != nil {
		return err
	}
	_ = os.Remove(path)
	marker, err := ActiveMarkerPath()
	if err != nil {
		return err
	}
	return os.Remove(marker)
}

func ApplyProject(p *config.Project, namespace string) (*State, error) {
	kubePath, err := config.KubeconfigPath(p.Name)
	if err != nil {
		return nil, err
	}
	ns := namespace
	if ns == "" {
		ns = p.DefaultNamespace
	}
	if ns == "" {
		ns = "default"
	}
	s := &State{
		Project:     p.Name,
		AWSProfile:  p.AWSProfile,
		AWSRegion:   p.Region,
		AccountID:   p.AccountID,
		KubeContext: p.EffectiveContextAlias(),
		Namespace:   ns,
		Tier:        p.Tier,
		Kubeconfig:  kubePath,
	}
	return s, Save(s)
}

func EnvExports(s *State, projectEnv map[string]string) []string {
	if s == nil {
		return nil
	}
	exports := []string{
		fmt.Sprintf("export AWS_PROFILE=%q", s.AWSProfile),
		fmt.Sprintf("export AWS_REGION=%q", s.AWSRegion),
		fmt.Sprintf("export AWS_DEFAULT_REGION=%q", s.AWSRegion),
		fmt.Sprintf("export KUBECONFIG=%q", s.Kubeconfig),
		fmt.Sprintf("export KUBE_NAMESPACE=%q", s.Namespace),
		fmt.Sprintf("export ANCHOR_PROJECT=%q", s.Project),
		fmt.Sprintf("export ANCHOR_TIER=%q", s.Tier),
	}
	for k, v := range projectEnv {
		exports = append(exports, fmt.Sprintf("export %s=%q", k, v))
	}
	return exports
}
