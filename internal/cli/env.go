package cli

import (
	"fmt"

	"ctxly/internal/kube"
	"ctxly/internal/session"

	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Print shell exports for the active session",
	Run: func(cmd *cobra.Command, args []string) {
		shell, _ := cmd.Flags().GetString("shell")
		s, p, err := activeSession()
		if err != nil {
			exitErr(err)
			return
		}
		exports := session.EnvExports(s, p.Env)
		switch shell {
		case "zsh", "bash", "":
			for _, e := range exports {
				fmt.Println(e)
			}
		default:
			exitErr(fmt.Errorf("unsupported shell %q (use zsh or bash)", shell))
		}
	},
}

func listNamespaces(s *session.State) ([]string, error) {
	return kube.ListNamespaces(s.Kubeconfig, s.KubeContext)
}

func switchNamespace(s *session.State, ns string) error {
	if err := kube.UseNamespace(s.Kubeconfig, s.KubeContext, ns); err != nil {
		return err
	}
	s.Namespace = ns
	return session.Save(s)
}

func init() {
	envCmd.Flags().String("shell", "zsh", "Shell format (zsh, bash)")
}
