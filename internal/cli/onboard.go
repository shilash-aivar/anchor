package cli

import (
	"fmt"
	"os"

	"ctxly/internal/config"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Check dependencies and initialize config directory",
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.InitDefaults(); err != nil {
			exitErr(err)
			return
		}
		dir, _ := config.ConfigDir()
		fmt.Printf("Config dir: %s\n\n", dir)

		checks := []struct {
			name, hint string
			ok         bool
		}{
			{"aws", "brew install awscli", awsOK()},
			{"kubectl", "brew install kubectl", kubectlOK()},
			{"fzf", "brew install fzf (optional; better pickers)", toolOK("fzf")},
			{"stern", "brew install stern", toolOK("stern")},
			{"k9s", "brew install k9s", toolOK("k9s")},
		}
		for _, c := range checks {
			if c.ok {
				fmt.Printf("  ✓ %s\n", c.name)
			} else {
				fmt.Printf("  ✗ %s — %s\n", c.name, c.hint)
			}
		}

		fmt.Println("\nNext steps:")
		fmt.Println("  1. ctxly project add")
		fmt.Println("  2. ctxly login <aws-profile>")
		fmt.Println("  3. ctxly project use        # fzf picker when name omitted")
		fmt.Println("  4. Shell hook — see README (auto-export after project use)")
		fmt.Println("\nCompletions:")
		fmt.Println("  ctxly completion zsh > $(brew --prefix)/share/zsh/site-functions/_ctxly")
		fmt.Println("  make install-completions")
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Write .ctx.yaml in the current repo",
	Run: func(cmd *cobra.Command, args []string) {
		project, _ := cmd.Flags().GetString("project")
		namespace, _ := cmd.Flags().GetString("namespace")
		if project == "" {
			exitErr(fmt.Errorf("--project is required"))
			return
		}
		type repoCtx struct {
			Project   string `yaml:"project"`
			Namespace string `yaml:"namespace,omitempty"`
		}
		data, err := yaml.Marshal(repoCtx{Project: project, Namespace: namespace})
		if err != nil {
			exitErr(err)
			return
		}
		if err := os.WriteFile(".ctx.yaml", data, 0o644); err != nil {
			exitErr(err)
			return
		}
		fmt.Println("✓ Wrote .ctx.yaml")
		fmt.Printf("  project: %s\n", project)
		if namespace != "" {
			fmt.Printf("  namespace: %s\n", namespace)
		}
	},
}

func awsOK() bool {
	return awsAvailable()
}

func awsAvailable() bool {
	_, err := lookPath("aws")
	return err == nil
}

func toolOK(name string) bool {
	_, err := lookPath(name)
	return err == nil
}

func init() {
	initCmd.Flags().String("project", "", "Project name from ~/.config/ctxly/projects/")
	initCmd.Flags().String("namespace", "", "Default namespace for this repo")
}
