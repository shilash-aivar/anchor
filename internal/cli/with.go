package cli

import (
	"fmt"
	"strings"

	"anchor/internal/use"

	"github.com/spf13/cobra"
)

var withCmd = &cobra.Command{
	Use:   "with <project> [flags] -- <command> [args...]",
	Short: "Run a command in a project without changing the active session",
	Long: `Run a single command with another project's AWS profile and kubeconfig.
Your saved session (anchor project use) is not modified.

  anchor with client-a -- kubectl get pods
  anchor with client-a -n app -- stern api --since 1h`,
	Example: `  anchor with staging -- kubectl get deploy
  anchor with client-a -y -- helm list`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runWith(cmd, args); err != nil {
			exitErr(err)
		}
	},
}

func runWith(cmd *cobra.Command, args []string) error {
	sep := -1
	for i, a := range args {
		if a == "--" {
			sep = i
			break
		}
	}
	if sep < 1 {
		return fmt.Errorf("usage: anchor with <project> [-n ns] [-y] -- <command> [args...]")
	}
	if sep >= len(args)-1 {
		return fmt.Errorf("missing command after --")
	}

	project := args[0]
	ns, _ := cmd.Flags().GetString("namespace")
	skip, _ := cmd.Flags().GetBool("yes")
	command := args[sep+1]
	cmdArgs := args[sep+2:]

	r, err := use.Prepare(project, ns, skip)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "→ with %s (%s/%s): %s\n",
		r.State.Project, r.State.KubeContext, r.State.Namespace, strings.Join(append([]string{command}, cmdArgs...), " "))

	return use.RunCommand(r, command, cmdArgs)
}

var recentCmd = &cobra.Command{
	Use:   "recent",
	Short: "List or switch to a recently used project",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runRecent(cmd, args); err != nil {
			exitErr(err)
		}
	},
}

func runRecent(cmd *cobra.Command, args []string) error {
	entries, err := loadRecentEntries()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("No recent projects.")
		return nil
	}

	pick, _ := cmd.Flags().GetBool("pick")
	if len(args) == 0 && !pick {
		for i, e := range entries {
			fmt.Printf("  %d) %s / %s  (%s)\n", i+1, e.Project, e.Namespace, e.UsedAt.Format("2006-01-02 15:04"))
		}
		fmt.Println("\nSwitch: anchor recent --pick")
		return nil
	}

	labels := make([]string, len(entries))
	for i, e := range entries {
		labels[i] = fmt.Sprintf("%s / %s", e.Project, e.Namespace)
	}
	label, err := pickRecentLabel(labels)
	if err != nil {
		return err
	}
	idx := 0
	for i, l := range labels {
		if l == label {
			idx = i
			break
		}
	}
	e := entries[idx]
	r, err := use.Activate(e.Project, e.Namespace, false)
	if err != nil {
		return err
	}
	use.PrintSuccess(r)
	fmt.Println("\nShell hook: eval \"$(anchor env --shell zsh)\"")
	return nil
}

func init() {
	withCmd.Flags().StringP("namespace", "n", "", "Namespace override")
	withCmd.Flags().BoolP("yes", "y", false, "Skip production confirmation")
	recentCmd.Flags().Bool("pick", false, "Interactively switch to a recent project")
}
