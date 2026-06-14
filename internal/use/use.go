package use

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"anchor/internal/audit"
	"anchor/internal/awsx"
	"anchor/internal/config"
	"anchor/internal/guard"
	"anchor/internal/hooks"
	"anchor/internal/kube"
	"anchor/internal/session"
)

type Result struct {
	State   *session.State
	Project *config.Project
}

type Options struct {
	SkipConfirm bool
	AutoLogin   bool
	NoLogin     bool
}

func DefaultOptions(skipConfirm bool) Options {
	return Options{SkipConfirm: skipConfirm}
}

func Prepare(name, namespace string, opt Options) (*Result, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}
	p, err := config.LoadProject(name)
	if err != nil {
		return nil, err
	}
	if !opt.SkipConfirm {
		if err := guard.ConfirmProjectSwitch(p, cfg.Options.ConfirmProduction); err != nil {
			return nil, err
		}
	}

	kubePath, err := config.KubeconfigPath(p.Name)
	if err != nil {
		return nil, err
	}

	if !awsx.AWSAvailable() {
		return nil, fmt.Errorf("aws CLI not found in PATH")
	}

	autoLogin := opt.AutoLogin || (cfg.Options.AutoLoginOnUse && !opt.NoLogin)
	if err := ensureAWSAuth(p.AWSProfile, autoLogin, opt.NoLogin); err != nil {
		return nil, err
	}

	if id, err := awsx.GetCallerIdentity(p.AWSProfile); err == nil {
		if p.AccountID == "" {
			p.AccountID = id.Account
		}
	}

	alias := p.EffectiveContextAlias()
	if err := awsx.UpdateKubeconfig(p.AWSProfile, p.Region, p.Cluster, alias, kubePath); err != nil {
		return nil, err
	}

	ns := namespace
	if ns == "" {
		ns = p.DefaultNamespace
	}
	if ns == "" {
		ns = "default"
	}
	if err := kube.UseNamespace(kubePath, alias, ns); err != nil {
		return nil, err
	}

	s := &session.State{
		Project:     p.Name,
		AWSProfile:  p.AWSProfile,
		AWSRegion:   p.Region,
		AccountID:   p.AccountID,
		KubeContext: alias,
		Namespace:   ns,
		Tier:        config.NormalizeTier(p.Tier),
		Kubeconfig:  kubePath,
	}

	if p.VPNRequired {
		if err := kube.ClusterReachable(s.Kubeconfig, s.KubeContext); err != nil {
			return nil, fmt.Errorf("cluster unreachable (vpn_required: true) — connect VPN and retry: %w", err)
		}
	}

	return &Result{State: s, Project: p}, nil
}

func ensureAWSAuth(profile string, autoLogin, noLogin bool) error {
	if _, err := awsx.GetCallerIdentity(profile); err == nil {
		if !autoLogin || !awsx.NeedsLogin(profile, 30*time.Minute) {
			return nil
		}
	} else if noLogin || !autoLogin {
		return fmt.Errorf("aws credentials invalid or expired (profile %q): run `anchor login %s` or `anchor use --auto-login`", profile, profile)
	}
	fmt.Fprintln(os.Stderr, "→ AWS credentials expired or expiring, running SSO login…")
	if err := awsx.SSOLogin(profile); err != nil {
		return err
	}
	if _, err := awsx.GetCallerIdentity(profile); err != nil {
		return err
	}
	return nil
}

func Activate(name, namespace string, opt Options) (*Result, error) {
	r, err := Prepare(name, namespace, opt)
	if err != nil {
		return nil, err
	}
	if err := hooks.PreUse(r.State); err != nil {
		return nil, err
	}
	if err := session.Save(r.State); err != nil {
		return nil, err
	}
	_ = session.RecordRecent(r.State)
	_ = audit.Log("use", fmt.Sprintf("project=%s ns=%s", r.State.Project, r.State.Namespace))
	_ = hooks.PostUse(r.State)
	return r, nil
}

// Backward-compatible wrappers
func ActivateSimple(name, namespace string, skipConfirm bool) (*Result, error) {
	return Activate(name, namespace, DefaultOptions(skipConfirm))
}

func PrepareSimple(name, namespace string, skipConfirm bool) (*Result, error) {
	return Prepare(name, namespace, DefaultOptions(skipConfirm))
}

func PrintSuccess(r *Result) {
	fmt.Printf("✓ Project:  %s\n", r.State.Project)
	fmt.Printf("  AWS:      profile=%s region=%s", r.State.AWSProfile, r.State.AWSRegion)
	if r.State.AccountID != "" {
		fmt.Printf(" account=%s", r.State.AccountID)
	}
	fmt.Println()
	fmt.Printf("  EKS:      %s\n", r.Project.Cluster)
	fmt.Printf("  K8s:      context=%s namespace=%s\n", r.State.KubeContext, r.State.Namespace)
	fmt.Printf("  Tier:     %s\n", r.State.Tier)
	if r.Project.ReadOnly {
		fmt.Printf("  Mode:     read-only\n")
	}
	if r.Project.VPNRequired {
		fmt.Printf("  VPN:      required\n")
	}
	fmt.Printf("  Kubeconfig: %s\n", r.State.Kubeconfig)
}

func RunCommand(r *Result, command string, args []string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = SessionEnviron(r.State, r.Project.Env)
	return cmd.Run()
}

func SessionEnviron(s *session.State, projectEnv map[string]string) []string {
	base := os.Environ()
	set := map[string]string{
		"AWS_PROFILE":        s.AWSProfile,
		"AWS_REGION":         s.AWSRegion,
		"AWS_DEFAULT_REGION": s.AWSRegion,
		"KUBECONFIG":         s.Kubeconfig,
		"KUBE_NAMESPACE":     s.Namespace,
		"ANCHOR_PROJECT":     s.Project,
		"ANCHOR_TIER":        s.Tier,
		"ANCHOR_SUBSHELL":    "1",
	}
	if s.AccountID != "" {
		set["ANCHOR_ACCOUNT_ID"] = s.AccountID
	}
	if p, err := config.LoadProject(s.Project); err == nil && p.Cluster != "" {
		set["ANCHOR_CLUSTER"] = p.Cluster
	}
	for k, v := range projectEnv {
		set[k] = v
	}

	strip := map[string]bool{
		"AWS_ACCESS_KEY_ID":     true,
		"AWS_SECRET_ACCESS_KEY": true,
		"AWS_SESSION_TOKEN":     true,
		"AWS_DEFAULT_PROFILE":   true,
	}
	for k := range set {
		strip[k] = true
	}

	out := make([]string, 0, len(base)+len(set))
	for _, kv := range base {
		key := kv
		if i := indexEnvKey(kv); i >= 0 {
			key = kv[:i]
		}
		if strip[key] {
			continue
		}
		out = append(out, kv)
	}
	for k, v := range set {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return out
}

func indexEnvKey(kv string) int {
	for i := 0; i < len(kv); i++ {
		if kv[i] == '=' {
			return i
		}
	}
	return -1
}
