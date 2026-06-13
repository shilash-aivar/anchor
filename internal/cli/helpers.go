package cli

import (
	"anchor/internal/awsx"
	"anchor/internal/kube"
	"anchor/internal/session"
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

func awsIdentity(profile string) (*awsx.Identity, error) {
	return awsx.GetCallerIdentity(profile)
}
