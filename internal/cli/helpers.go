package cli

import (
	"anchor/internal/awsx"
	"anchor/internal/config"
	"anchor/internal/kube"
	"anchor/internal/session"
	"anchor/internal/use"

	"github.com/spf13/cobra"
)

func kubectlOK() bool {
	return kube.KubectlAvailable()
}

func lookPath(name string) (string, error) {
	return kube.LookPath(name)
}

func clusterReachable(s *session.State) error {
	return kube.ClusterReachable(s.Kubeconfig, s.KubeContext)
}

func clusterReachableForProject(s *session.State, p *config.Project) error {
	if p != nil && !p.VPNRequired {
		return nil
	}
	return clusterReachable(s)
}

func useOptsFrom(cmd *cobra.Command, skipConfirm bool) use.Options {
	autoLogin, _ := cmd.Flags().GetBool("auto-login")
	noLogin, _ := cmd.Flags().GetBool("no-login")
	return use.Options{SkipConfirm: skipConfirm, AutoLogin: autoLogin, NoLogin: noLogin}
}

func registerAutoLoginFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("auto-login", false, "Run SSO login if AWS credentials expired or expiring")
	cmd.Flags().Bool("no-login", false, "Fail instead of SSO login when credentials expired")
}

func awsIdentity(profile string) (*awsx.Identity, error) {
	return awsx.GetCallerIdentity(profile)
}
