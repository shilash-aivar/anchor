package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"ctxly/internal/audit"
	"ctxly/internal/config"
	"ctxly/internal/guard"
	"ctxly/internal/hooks"
	"ctxly/internal/kube"
	"ctxly/internal/picker"
	"ctxly/internal/use"

	"github.com/spf13/cobra"
)

var pfCmd = &cobra.Command{
	Use:   "pf <target> [local:remote]",
	Short: "Port-forward (svc/name or pod/name)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		s, _, err := activeSession()
		if err != nil {
			exitErr(err)
			return
		}
		ports := args[1:]
		if len(ports) == 0 {
			ports = []string{"8080:80"}
		}
		if err := kube.PortForward(s.Kubeconfig, s.KubeContext, s.Namespace, args[0], ports); err != nil {
			exitErr(err)
		}
	},
}

var watchCmd = &cobra.Command{
	Use:   "watch <resource>",
	Short: "Watch rollout status or resource changes",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		s, _, err := activeSession()
		if err != nil {
			exitErr(err)
			return
		}
		target := args[0]
		if strings.HasPrefix(target, "deploy/") || strings.HasPrefix(target, "deployment/") {
			if err := kube.RolloutWatch(s.Kubeconfig, s.KubeContext, s.Namespace, target); err != nil {
				exitErr(err)
			}
			return
		}
		if err := kube.GetWatch(s.Kubeconfig, s.KubeContext, s.Namespace, append([]string{"get", target}, args[1:]...)); err != nil {
			exitErr(err)
		}
	},
}

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Show namespace events",
	Run: func(cmd *cobra.Command, args []string) {
		s, _, err := activeSession()
		if err != nil {
			exitErr(err)
			return
		}
		warn, _ := cmd.Flags().GetBool("warnings")
		if err := kube.Events(s.Kubeconfig, s.KubeContext, s.Namespace, warn); err != nil {
			exitErr(err)
		}
	},
}

var cpCmd = &cobra.Command{
	Use:   "cp <remote-path> [local-path]",
	Short: "Copy from pod (picker if --pod omitted)",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		s, _, err := activeSession()
		if err != nil {
			exitErr(err)
			return
		}
		remote := args[0]
		local := "."
		if len(args) == 2 {
			local = args[1]
		}
		pod, _ := cmd.Flags().GetString("pod")
		if pod == "" {
			pods, err := kube.ListPods(s.Kubeconfig, s.KubeContext, s.Namespace)
			if err != nil {
				exitErr(err)
				return
			}
			pod, err = picker.Choose("Select pod:", pods)
			if err != nil {
				exitErr(err)
				return
			}
		}
		container, _ := cmd.Flags().GetString("container")
		if err := kube.CopyFromPod(s.Kubeconfig, s.KubeContext, s.Namespace, pod, container, remote, local); err != nil {
			exitErr(err)
		}
	},
}

var findCmd = &cobra.Command{
	Use:   "find <query>",
	Short: "Search pods, deployments, and services by name",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		s, _, err := activeSession()
		if err != nil {
			exitErr(err)
			return
		}
		out, err := kube.FindResourcesSimple(s.Kubeconfig, s.KubeContext, s.Namespace, args[0])
		if err != nil {
			exitErr(err)
			return
		}
		if out == "" {
			fmt.Println("(no matches)")
			return
		}
		fmt.Println(out)
	},
}

var debugCmd = &cobra.Command{
	Use:   "debug <pod> [-- args...]",
	Short: "kubectl debug wrapper",
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		s, _, err := activeSession()
		if err != nil {
			exitErr(err)
			return
		}
		if len(args) == 0 {
			exitErr(fmt.Errorf("usage: ctxly debug <pod> [-- args...]"))
			return
		}
		pod := args[0]
		rest := args[1:]
		if err := kube.DebugPod(s.Kubeconfig, s.KubeContext, s.Namespace, pod, "", rest); err != nil {
			exitErr(err)
		}
	},
}

var helmCmd = &cobra.Command{
	Use:                "helm",
	Short:              "helm passthrough with active session",
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		s, p, err := activeSession()
		if err != nil {
			exitErr(err)
			return
		}
		if len(args) == 0 {
			exitErr(fmt.Errorf("usage: ctxly helm <helm args...>"))
			return
		}
		cfg, _ := config.LoadConfig()
		if err := guard.CheckMutating(s, p, cfg, "helm", args); err != nil {
			exitErr(err)
			return
		}
		_ = audit.Log("helm", strings.Join(args, " "))
		if err := kube.RunHelm(s.Kubeconfig, s.KubeContext, s.Namespace, args); err != nil {
			exitErr(err)
		}
	},
}

var applyCmd = &cobra.Command{
	Use:                "apply",
	Short:              "Guarded kubectl apply with production confirm",
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		s, p, err := activeSession()
		if err != nil {
			exitErr(err)
			return
		}
		cfg, _ := config.LoadConfig()
		if err := guard.CheckMutating(s, p, cfg, "apply", args); err != nil {
			exitErr(err)
			return
		}
		if err := guard.ConfirmApply(p, cfg); err != nil {
			exitErr(err)
			return
		}
		if err := hooks.PreApply(s); err != nil {
			exitErr(err)
			return
		}
		kargs := append([]string{"apply"}, args...)
		if cfg != nil && cfg.Options.DryRunProduction && p.IsProduction() {
			hasDry := false
			for _, a := range args {
				if strings.Contains(a, "dry-run") {
					hasDry = true
					break
				}
			}
			if !hasDry {
				kargs = append(kargs, "--dry-run=server")
				fmt.Fprintln(os.Stderr, "→ production dry-run=server (pass --dry-run=none to override)")
			}
		}
		fmt.Fprintf(os.Stderr, "→ apply context=%s namespace=%s\n", s.KubeContext, s.Namespace)
		_ = audit.Log("apply", strings.Join(kargs, " "))
		if err := kube.Passthrough(s.Kubeconfig, s.KubeContext, s.Namespace, kargs); err != nil {
			exitErr(err)
		}
	},
}

var shareCmd = &cobra.Command{
	Use:   "share",
	Short: "Print pasteable session info for Slack",
	Run:   runShare,
}

var linksCmd = &cobra.Command{
	Use:   "links [name]",
	Short: "List or open project links (grafana, runbook, etc.)",
	Run:   runLinks,
}

var shellCmd = &cobra.Command{
	Use:   "shell [project]",
	Short: "Spawn a subshell with project environment (does not change saved session unless project given with activate flag)",
	Run:   runShell,
}

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Validate project configs and kubeconfig hygiene",
	Run:   runLint,
}

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove orphan kubeconfig files without a project",
	Run:   runPrune,
}

var validateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validate .ctx.yaml or project config",
	Run:   runValidate,
}

var useCmd = &cobra.Command{
	Use:   "use [project]",
	Short: "Activate project (reads .ctx.yaml in repo if no name)",
	Run:   runUse,
}

func runLinks(cmd *cobra.Command, args []string) {
	s, p, err := activeSession()
	if err != nil {
		exitErr(err)
		return
	}
	if len(p.Links) == 0 {
		fmt.Println("(no links in project config)")
		return
	}
	if len(args) == 0 {
		for k, v := range p.Links {
			fmt.Printf("  %-12s %s\n", k, v)
		}
		fmt.Printf("\nOpen: ctxly links <name> --open   (project: %s)\n", s.Project)
		return
	}
	url, ok := p.Links[args[0]]
	if !ok {
		exitErr(fmt.Errorf("unknown link %q", args[0]))
		return
	}
	open, _ := cmd.Flags().GetBool("open")
	if open {
		if err := openBrowser(url); err != nil {
			exitErr(err)
			return
		}
		return
	}
	fmt.Println(url)
}

func runShell(cmd *cobra.Command, args []string) {
	activate, _ := cmd.Flags().GetBool("activate")
	skip, _ := cmd.Flags().GetBool("yes")

	var r *use.Result
	var err error
	if len(args) == 1 {
		if activate {
			r, err = use.Activate(args[0], "", skip)
		} else {
			r, err = use.Prepare(args[0], "", skip)
		}
	} else {
		s, p, err2 := activeSession()
		if err2 != nil {
			exitErr(err2)
			return
		}
		r = &use.Result{State: s, Project: p}
	}
	if err != nil {
		exitErr(err)
		return
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	fmt.Fprintf(os.Stderr, "→ subshell project=%s (exit to return)\n", r.State.Project)
	c := exec.Command(shell)
	c.Env = use.SessionEnviron(r.State, r.Project.Env)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		exitErr(err)
	}
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Run()
	case "linux":
		return exec.Command("xdg-open", url).Run()
	default:
		fmt.Println(url)
		return nil
	}
}

func init() {
	eventsCmd.Flags().Bool("warnings", false, "Show Warning/Error events only")
	cpCmd.Flags().String("pod", "", "Pod name")
	cpCmd.Flags().StringP("container", "c", "", "Container name")
	shareCmd.Flags().Bool("json", false, "JSON output")
	linksCmd.Flags().Bool("open", false, "Open URL in browser")
	shellCmd.Flags().Bool("activate", false, "Also save as active session")
	shellCmd.Flags().BoolP("yes", "y", false, "Skip production confirmation")
	pruneCmd.Flags().Bool("dry-run", false, "Show what would be removed")
	initUseCmd()
}
