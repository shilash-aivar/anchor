package repo

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const RepoFile = ".ctx.yaml"

type RepoContext struct {
	Project   string `yaml:"project"`
	Namespace string `yaml:"namespace,omitempty"`
}

func FindRepoContext(start string) (*RepoContext, string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}
	if start != "" {
		dir = start
	}
	for {
		path := filepath.Join(dir, RepoFile)
		if data, err := os.ReadFile(path); err == nil {
			var rc RepoContext
			if err := yaml.Unmarshal(data, &rc); err != nil {
				return nil, path, fmt.Errorf("parse %s: %w", path, err)
			}
			if rc.Project == "" {
				return nil, path, fmt.Errorf("%s: project is required", path)
			}
			return &rc, path, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil, "", nil
}

func ValidateRepoFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var rc RepoContext
	if err := yaml.Unmarshal(data, &rc); err != nil {
		return err
	}
	if rc.Project == "" {
		return fmt.Errorf("project field required")
	}
	return nil
}
