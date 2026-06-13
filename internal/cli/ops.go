package cli

import (
	"fmt"
	"strings"

	"ctxly/internal/audit"
	"ctxly/internal/config"
	"ctxly/internal/guard"
	"ctxly/internal/kube"
	"ctxly/internal/picker"
	"ctxly/internal/session"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [query]",
	Short: "Tail logs via stern (deployment/pod prefix)",
	Run: func(cmd *cobra.Command, args []string) {
		s, _, err := activeSession()
		if err != nil {
			exitErr(err)
			return
		}
		query := strings.Join(args, " ")
		if query == "" {
			exitErr(fmt.Errorf("usage: ctxly logs <deployment|pod-prefix>"))
			return
		}
		if _, err := kube.LookPath("stern"); err != nil {
			exitErr(fmt.Errorf("stern not found — install: brew install stern"))
			return
		}
		sternArgs := []string{query, "--context", s.KubeContext, "--kubeconfig", s.Kubeconfig}
		if s.Namespace != "" {
			sternArgs = append(sternArgs, "--namespace", s.Namespace)
		}
		if prev, _ := cmd.Flags().GetBool("previous"); prev {
			sternArgs = append(sternArgs, "--previous")
		}
		if since, _ := cmd.Flags().GetString("since"); since != "" {
			sternArgs = append(sternArgs, "--since", since)
		}
		if tail, _ := cmd.Flags().GetInt("tail"); tail > 0 {
			sternArgs = append(sternArgs, "--tail", fmt.Sprintf("%d", tail))
		}
		extras, _ := cmd.Flags().GetStringArray("stern-arg")
		sternArgs = append(sternArgs, extras...)
		if err := kube.RunExternal("stern", sternArgs, map[string]string{"KUBECONFIG": s.Kubeconfig}); err != nil {
			exitErr(err)
		}
	},
}

var execCmd = &cobra.Command{
	Use:   "exec [pod] [-- command...]",
	Short: "Exec into a pod (interactive picker if pod omitted)",
	Run: func(cmd *cobra.Command, args []string) {
		s, p, err := activeSession()
		if err != nil {
			exitErr(err)
			return
		}
		pod := ""
		cmdArgs := []string{"/bin/sh"}
		if len(args) > 0 {
			pod = args[0]
			if len(args) > 1 {
				cmdArgs = args[1:]
			}
		} else {
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
		if container == "" {
			if containers, err := kube.ListContainers(s.Kubeconfig, s.KubeContext, s.Namespace, pod); err == nil && len(containers) > 1 {
				container, err = picker.Choose("Select container:", containers)
				if err != nil {
					exitErr(err)
					return
				}
			} else if len(containers) == 1 {
				container = containers[0]
			}
		}
		cfg, _ := config.LoadConfig()
		_ = guard.CheckMutating(s, p, cfg, "exec", nil)
		_ = audit.Log("exec", fmt.Sprintf("pod=%s container=%s", pod, container))
		if err := kube.ExecPod(s.Kubeconfig, s.KubeContext, s.Namespace, pod, container, cmdArgs); err != nil {
			exitErr(err)
		}
	},
}

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch k9s for the active session",
	Run: func(cmd *cobra.Command, args []string) {
		s, _, err := activeSession()
		if err != nil {
			exitErr(err)
			return
		}
		if _, err := kube.LookPath("k9s"); err != nil {
			exitErr(fmt.Errorf("k9s not found — install: brew install k9s"))
			return
		}
		k9sArgs := []string{"--context", s.KubeContext, "--namespace", s.Namespace, "--kubeconfig", s.Kubeconfig}
		if err := kube.RunExternal("k9s", k9sArgs, map[string]string{"KUBECONFIG": s.Kubeconfig}); err != nil {
			exitErr(err)
		}
	},
}

var kCmd = &cobra.Command{
	Use:                "k",
	Short:              "kubectl passthrough with active session",
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		s, p, err := activeSession()
		if err != nil {
			exitErr(err)
			return
		}
		if len(args) == 0 {
			exitErr(fmt.Errorf("usage: ctxly k <kubectl args...>"))
			return
		}
		if err := runGuardedKubectl(s, p, args); err != nil {
			exitErr(err)
		}
	},
}

func runGuardedKubectl(s *session.State, p *config.Project, args []string) error {
	cfg, _ := config.LoadConfig()
	if err := guard.CheckMutating(s, p, cfg, "k", args); err != nil {
		return err
	}
	if len(args) > 0 && (args[0] == "apply" || args[0] == "delete" || args[0] == "patch") {
		_ = audit.Log("kubectl", strings.Join(args, " "))
	}
	return kube.Passthrough(s.Kubeconfig, s.KubeContext, s.Namespace, args)
}

func init() {
	logsCmd.Flags().Bool("previous", false, "Previous container logs (crashed pods)")
	logsCmd.Flags().String("since", "", "Log time window (e.g. 1h, 10m)")
	logsCmd.Flags().Int("tail", 0, "Number of lines to tail")
	logsCmd.Flags().StringArray("stern-arg", nil, "Extra args passed to stern")
	execCmd.Flags().StringP("container", "c", "", "Container name")
}
