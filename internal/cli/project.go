package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"ctxly/internal/config"
	"ctxly/internal/picker"
	"ctxly/internal/session"
	"ctxly/internal/use"

	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage personal AWS/EKS projects",
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured projects",
	Run: func(cmd *cobra.Command, args []string) {
		names, err := config.ListProjects()
		if err != nil {
			exitErr(err)
		}
		if len(names) == 0 {
			fmt.Println("No projects configured.")
			fmt.Println("  ctxly project add")
			fmt.Println("  or copy examples/project.yaml → ~/.config/ctxly/projects/")
			return
		}
		active, _ := session.Load()
		for _, n := range names {
			marker := " "
			if active != nil && active.Project == n {
				marker = "*"
			}
			p, err := config.LoadProject(n)
			if err != nil {
				fmt.Printf("%s %s (error: %v)\n", marker, n, err)
				continue
			}
			fmt.Printf("%s %-20s tier=%-12s cluster=%s\n", marker, n, p.Tier, p.Cluster)
		}
	},
}

var projectUseCmd = &cobra.Command{
	Use:   "use [name]",
	Short: "Activate a project (AWS profile + EKS context + namespace)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ns, _ := cmd.Flags().GetString("namespace")
		skip, _ := cmd.Flags().GetBool("yes")

		name := ""
		if len(args) == 1 {
			name = args[0]
		} else {
			names, err := config.ListProjects()
			if err != nil {
				exitErr(err)
				return
			}
			if len(names) == 0 {
				exitErr(fmt.Errorf("no projects — run `ctxly project add`"))
				return
			}
			name, err = picker.Choose("Select project:", names)
			if err != nil {
				exitErr(err)
				return
			}
		}

		r, err := use.Activate(name, ns, skip)
		if err != nil {
			exitErr(err)
		}
		use.PrintSuccess(r)
		fmt.Println("\nShell hook: eval \"$(ctxly env --shell zsh)\"")
	},
}

var projectAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Interactively add a new project",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runProjectAdd(args); err != nil {
			exitErr(err)
		}
	},
}

func runProjectAdd(args []string) error {
	if err := config.InitDefaults(); err != nil {
		return err
	}
	reader := bufio.NewReader(os.Stdin)
	ask := func(prompt, def string) (string, error) {
		if def != "" {
			fmt.Fprintf(os.Stderr, "%s [%s]: ", prompt, def)
		} else {
			fmt.Fprintf(os.Stderr, "%s: ", prompt)
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return def, nil
		}
		return line, nil
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	name, err := ask("Project name", name)
	if err != nil || name == "" {
		return fmt.Errorf("project name required")
	}

	profile, err := ask("AWS profile", name)
	if err != nil {
		return err
	}
	region, err := ask("AWS region", "us-east-1")
	if err != nil {
		return err
	}
	cluster, err := ask("EKS cluster name", "")
	if err != nil || cluster == "" {
		return fmt.Errorf("cluster name required")
	}
	alias, err := ask("Context alias", name)
	if err != nil {
		return err
	}
	ns, err := ask("Default namespace", "default")
	if err != nil {
		return err
	}
	tier, err := ask("Tier (dev/staging/production)", "dev")
	if err != nil {
		return err
	}
	accountID, _ := ask("Account ID (optional)", "")

	p := &config.Project{
		Name:             name,
		AWSProfile:       profile,
		AccountID:        accountID,
		Region:           region,
		Cluster:          cluster,
		ContextAlias:     alias,
		DefaultNamespace: ns,
		Tier:             tier,
	}
	if err := config.SaveProject(p); err != nil {
		return err
	}
	path, _ := config.ProjectPath(name)
	fmt.Printf("✓ Wrote project config: %s\n", path)
	fmt.Printf("  Activate with: ctxly project use %s\n", name)
	return nil
}

var projectNotesCmd = &cobra.Command{
	Use:   "notes",
	Short: "Show notes for the active or named project",
	Run: func(cmd *cobra.Command, args []string) {
		name := ""
		if len(args) > 0 {
			name = args[0]
		} else if s, _, err := activeSession(); err == nil {
			name = s.Project
		} else {
			exitErr(err)
			return
		}
		p, err := config.LoadProject(name)
		if err != nil {
			exitErr(err)
			return
		}
		if p.Notes == "" {
			fmt.Println("(no notes)")
			return
		}
		fmt.Println(p.Notes)
	},
}

var nsCmd = &cobra.Command{
	Use:   "ns [namespace]",
	Short: "Switch namespace in the active project",
	Run: func(cmd *cobra.Command, args []string) {
		s, _, err := activeSession()
		if err != nil {
			exitErr(err)
			return
		}
		ns := ""
		if len(args) > 0 {
			ns = args[0]
		} else {
			namespaces, err := listNamespaces(s)
			if err != nil {
				exitErr(err)
				return
			}
			ns, err = picker.Choose("Select namespace:", namespaces)
			if err != nil {
				exitErr(err)
				return
			}
		}
		if err := switchNamespace(s, ns); err != nil {
			exitErr(err)
			return
		}
		fmt.Printf("✓ Namespace: %s (project: %s)\n", ns, s.Project)
	},
}

func init() {
	projectUseCmd.Flags().StringP("namespace", "n", "", "Namespace to activate")
	projectUseCmd.Flags().BoolP("yes", "y", false, "Skip production confirmation")

	projectCmd.AddCommand(projectListCmd, projectUseCmd, projectAddCmd, projectNotesCmd, projectImportCmd)
	rootCmd.AddCommand(nsCmd)
}
