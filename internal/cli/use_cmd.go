package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"anchor/internal/config"
	"anchor/internal/kubecfg"
	"anchor/internal/picker"
	"anchor/internal/repo"
	"anchor/internal/use"

	"github.com/spf13/cobra"
)

func runUse(cmd *cobra.Command, args []string) {
	ns, _ := cmd.Flags().GetString("namespace")
	fromRepo, _ := cmd.Flags().GetBool("from-repo")

	name := ""
	if len(args) == 1 {
		name = args[0]
	} else if fromRepo || len(args) == 0 {
		rc, path, err := repo.FindRepoContext("")
		if err != nil {
			exitErr(err)
			return
		}
		if rc != nil {
			name = rc.Project
			if ns == "" && rc.Namespace != "" {
				ns = rc.Namespace
			}
			fmt.Fprintf(os.Stderr, "→ from %s\n", path)
		} else if len(args) == 0 {
			names, err := config.ListProjects()
			if err != nil {
				exitErr(err)
				return
			}
			if len(names) == 0 {
				exitErr(fmt.Errorf("no projects — run `anchor project add`"))
				return
			}
			names = config.SortProjectsPinnedFirst(names)
			name, err = picker.Choose("Select project:", names)
			if err != nil {
				exitErr(err)
				return
			}
		}
	}
	if name == "" {
		exitErr(fmt.Errorf("project name required"))
		return
	}

	r, err := use.Activate(name, ns, useOpts(cmd))
	if err != nil {
		exitErr(err)
		return
	}
	use.PrintSuccess(r)
	fmt.Println("\nShell hook: eval \"$(anchor env --shell zsh)\"")
}

func runLint(cmd *cobra.Command, args []string) {
	issues, err := kubecfg.LintAll()
	if err != nil {
		exitErr(err)
		return
	}
	if len(issues) == 0 {
		fmt.Println("✓ no issues found")
		return
	}
	for _, i := range issues {
		prefix := "•"
		switch i.Level {
		case "error":
			prefix = "✗"
		case "warn":
			prefix = "⚠"
		}
		fmt.Printf("%s [%s] %s: %s\n", prefix, i.Level, i.Project, i.Message)
	}
}

func runPrune(cmd *cobra.Command, args []string) {
	dry, _ := cmd.Flags().GetBool("dry-run")
	removed, err := kubecfg.PruneOrphans(dry)
	if err != nil {
		exitErr(err)
		return
	}
	if len(removed) == 0 {
		fmt.Println("Nothing to prune.")
		return
	}
	for _, p := range removed {
		if dry {
			fmt.Printf("would remove: %s\n", p)
		} else {
			fmt.Printf("removed: %s\n", p)
		}
	}
}

func runValidate(cmd *cobra.Command, args []string) {
	path := ".ctx.yaml"
	if len(args) == 1 {
		path = args[0]
	}
	if _, err := os.Stat(path); err == nil {
		if err := repo.ValidateRepoFile(path); err != nil {
			exitErr(fmt.Errorf("%s: %w", path, err))
			return
		}
		fmt.Printf("✓ %s valid\n", path)
	}

	names, _ := config.ListProjects()
	if len(names) == 0 {
		fmt.Println("(no projects configured)")
		return
	}
	for _, n := range names {
		p, err := config.LoadProject(n)
		if err != nil {
			fmt.Printf("✗ project %s: %v\n", n, err)
			continue
		}
		if p.AWSProfile == "" || p.Cluster == "" {
			fmt.Printf("✗ project %s: incomplete config\n", n)
			continue
		}
		fmt.Printf("✓ project %s\n", n)
	}
}

func runShare(cmd *cobra.Command, args []string) {
	s, p, err := activeSession()
	if err != nil {
		exitErr(err)
		return
	}
	jsonOut, _ := cmd.Flags().GetBool("json")
	block := map[string]string{
		"project":   s.Project,
		"tier":      s.Tier,
		"namespace": s.Namespace,
		"context":   s.KubeContext,
		"cluster":   p.Cluster,
		"region":    s.AWSRegion,
		"profile":   s.AWSProfile,
	}
	if s.AccountID != "" {
		block["account_id"] = s.AccountID
	}
	if jsonOut {
		enc, _ := json.MarshalIndent(block, "", "  ")
		fmt.Println(string(enc))
		return
	}
	fmt.Println("── anchor session ──")
	for k, v := range block {
		fmt.Printf("  %-12s %s\n", k+":", v)
	}
}

func useOpts(cmd *cobra.Command) use.Options {
	skip, _ := cmd.Flags().GetBool("yes")
	autoLogin, _ := cmd.Flags().GetBool("auto-login")
	noLogin, _ := cmd.Flags().GetBool("no-login")
	return use.Options{SkipConfirm: skip, AutoLogin: autoLogin, NoLogin: noLogin}
}

func initUseCmd() {
	useCmd.Flags().StringP("namespace", "n", "", "Namespace override")
	useCmd.Flags().BoolP("yes", "y", false, "Skip production confirmation")
	useCmd.Flags().Bool("from-repo", false, "Load project from .ctx.yaml in directory tree")
	useCmd.Flags().Bool("auto-login", false, "Run SSO login if AWS credentials expired")
	useCmd.Flags().Bool("no-login", false, "Fail instead of SSO login when credentials expired")
}
