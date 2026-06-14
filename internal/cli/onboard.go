package cli

import (
	"fmt"
	"os"

	"anchor/internal/config"
	"anchor/internal/deps"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Check dependencies and initialize config directory",
	Long: `Verifies aws, kubectl, stern, k9s, helm, and fzf are on PATH.
Use --install to install missing tools via Homebrew (macOS/Linux).`,
	Run: func(cmd *cobra.Command, args []string) {
		install, _ := cmd.Flags().GetBool("install")
		requiredOnly, _ := cmd.Flags().GetBool("required-only")

		if err := config.InitDefaults(); err != nil {
			exitErr(err)
			return
		}
		dir, _ := config.ConfigDir()
		fmt.Printf("Config dir: %s\n\n", dir)

		if install {
			missing := deps.Missing(requiredOnly)
			if len(missing) == 0 {
				fmt.Println("All dependencies already installed.")
			} else {
				fmt.Println("Installing missing dependencies via Homebrew…")
				if err := deps.InstallMissing(requiredOnly); err != nil {
					exitErr(err)
					return
				}
				fmt.Println()
			}
		}

		fmt.Println("Dependencies:")
		missingRequired := 0
		missingOptional := 0
		for _, s := range deps.CheckAll() {
			label := s.Tool.Binary
			if s.Tool.Required {
				label += " (required)"
			}
			if s.OK {
				fmt.Printf("  ✓ %s\n", label)
			} else {
				fmt.Printf("  ✗ %s — %s\n", label, s.Install)
				if s.Tool.Required {
					missingRequired++
				} else {
					missingOptional++
				}
			}
		}

		if missingRequired > 0 && !install {
			fmt.Println("\nInstall required tools:")
			fmt.Println("  anchor onboard --install --required-only")
		}
		if missingOptional > 0 && !install {
			fmt.Println("\nInstall recommended tools (stern, k9s, helm, fzf):")
			fmt.Println("  anchor onboard --install")
			fmt.Println("  make install-deps")
		}

		fmt.Println("\nNext steps:")
		fmt.Println("  1. anchor project add")
		fmt.Println("  2. anchor login --all")
		fmt.Println("  3. anchor use")
		fmt.Println("  4. anchor dashboard          # optional web UI")
		fmt.Println("\nShell setup:")
		fmt.Println("  ./scripts/shell-setup.sh     # paste into ~/.zshrc")
		fmt.Println("\nCompletions:")
		fmt.Println("  make install-completions")
	},
}

func init() {
	onboardCmd.Flags().Bool("install", false, "Install missing dependencies via Homebrew")
	onboardCmd.Flags().Bool("required-only", false, "With --install, only install awscli and kubectl")
	initInitCmd()
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

func initInitCmd() {
	initCmd.Flags().String("project", "", "Project name from ~/.config/anchor/projects/")
	initCmd.Flags().String("namespace", "", "Default namespace for this repo")
}
