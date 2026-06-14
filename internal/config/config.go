package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	AppDirName  = "anchor"
	ConfigFile  = "config.yaml"
	StateFile   = "state.json"
	ProjectsDir = "projects"
	KubeDirName = "kube"
	AuditFile   = "audit.log"
)

type Config struct {
	SSO     SSOConfig     `yaml:"sso,omitempty"`
	Shell   ShellConfig   `yaml:"shell,omitempty"`
	Options OptionsConfig `yaml:"options,omitempty"`
	Hooks   HooksConfig   `yaml:"hooks,omitempty"`
}

type SSOConfig struct {
	StartURL string `yaml:"start_url,omitempty"`
	Region   string `yaml:"region,omitempty"`
}

type ShellConfig struct {
	DefaultProject string `yaml:"default_project,omitempty"`
}

type OptionsConfig struct {
	ConfirmProduction  bool   `yaml:"confirm_production,omitempty"`
	AnnounceContext    bool   `yaml:"announce_context,omitempty"`
	AuditLog           bool   `yaml:"audit_log,omitempty"`
	BlockDangerous     bool   `yaml:"block_dangerous,omitempty"`
	DryRunProduction   bool   `yaml:"dry_run_production,omitempty"`
	ProtectContextRegex string `yaml:"protect_context_regex,omitempty"`
	AutoLoginOnUse     bool   `yaml:"auto_login_on_use,omitempty"`
}

type HooksConfig struct {
	PreUse   string `yaml:"pre_use,omitempty"`
	PostUse  string `yaml:"post_use,omitempty"`
	PreApply string `yaml:"pre_apply,omitempty"`
}

type Project struct {
	Name             string            `yaml:"name"`
	AWSProfile       string            `yaml:"aws_profile"`
	AccountID        string            `yaml:"account_id,omitempty"`
	Region           string            `yaml:"region"`
	Tier             string            `yaml:"tier"`
	Cluster          string            `yaml:"cluster"`
	ContextAlias     string            `yaml:"context_alias,omitempty"`
	DefaultNamespace string            `yaml:"default_namespace,omitempty"`
	RequireConfirm   *bool             `yaml:"require_confirm,omitempty"`
	ConfirmText      string            `yaml:"confirm_text,omitempty"`
	ReadOnly         bool              `yaml:"readonly,omitempty"`
	Notes            string            `yaml:"notes,omitempty"`
	Env              map[string]string `yaml:"env,omitempty"`
	Links            map[string]string `yaml:"links,omitempty"`
	VPNRequired      bool              `yaml:"vpn_required,omitempty"`
}

func (p Project) EffectiveContextAlias() string {
	if p.ContextAlias != "" {
		return p.ContextAlias
	}
	return p.Name
}

func (p Project) EffectiveConfirmText() string {
	if p.ConfirmText != "" {
		return p.ConfirmText
	}
	return p.Name
}

func (p Project) IsProduction() bool {
	return IsProductionTier(p.Tier)
}

func (p Project) ShouldConfirm(globalConfirmProduction bool) bool {
	if p.RequireConfirm != nil {
		return *p.RequireConfirm
	}
	return globalConfirmProduction && p.IsProduction()
}

func (c *Config) AuditEnabled() bool {
	if c == nil {
		return true
	}
	// default on when unset in fresh config - InitDefaults sets explicitly
	return c.Options.AuditLog || (!c.Options.AuditLog && c.Options.ConfirmProduction)
}

func DefaultOptions() OptionsConfig {
	return OptionsConfig{
		ConfirmProduction: true,
		AnnounceContext:   true,
		AuditLog:          true,
		BlockDangerous:    true,
		DryRunProduction:  false,
	}
}

func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", AppDirName), nil
}

func AuditPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, AuditFile), nil
}

func EnsureConfigDir() (string, error) {
	maybeMigrateFromLegacy()
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(dir, ProjectsDir), 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(dir, KubeDirName), 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFile), nil
}

func ProjectPath(name string) (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ProjectsDir, name+".yaml"), nil
}

func KubeconfigPath(projectName string) (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, KubeDirName, projectName+".yaml"), nil
}

func LoadConfig() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := &Config{Options: DefaultOptions()}
			return cfg, nil
		}
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	dir, err := EnsureConfigDir()
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ConfigFile), data, 0o644)
}

func LoadProject(name string) (*Project, error) {
	path, err := ProjectPath(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("project %q: %w", name, err)
	}
	var p Project
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse project %q: %w", name, err)
	}
	if p.Name == "" {
		p.Name = name
	}
	return &p, nil
}

func SaveProject(p *Project) error {
	if _, err := EnsureConfigDir(); err != nil {
		return err
	}
	path, err := ProjectPath(p.Name)
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func ListProjects() ([]string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(dir, ProjectsDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) == ".yaml" {
			names = append(names, name[:len(name)-len(".yaml")])
		}
	}
	return names, nil
}

func InitDefaults() error {
	dir, err := EnsureConfigDir()
	if err != nil {
		return err
	}
	cfgPath := filepath.Join(dir, ConfigFile)
	if _, err := os.Stat(cfgPath); err == nil {
		return nil
	}
	return SaveConfig(&Config{Options: DefaultOptions()})
}

func LoadAllProjects() ([]*Project, error) {
	names, err := ListProjects()
	if err != nil {
		return nil, err
	}
	out := make([]*Project, 0, len(names))
	for _, n := range names {
		p, err := LoadProject(n)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func maybeMigrateFromLegacy() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	oldDir := filepath.Join(home, ".config", "ctxly")
	newDir := filepath.Join(home, ".config", AppDirName)
	if _, err := os.Stat(newDir); err == nil {
		return
	}
	if _, err := os.Stat(oldDir); err != nil {
		return
	}
	_ = os.Rename(oldDir, newDir)
}
