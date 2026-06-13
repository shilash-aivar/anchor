package cli

import (
	"fmt"

	"anchor/internal/awsx"
	"anchor/internal/config"
	"anchor/internal/use"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Refresh kubeconfig and verify AWS credentials for all projects",
	Run: func(cmd *cobra.Command, args []string) {
		skip, _ := cmd.Flags().GetBool("yes")
		projects, err := config.LoadAllProjects()
		if err != nil {
			exitErr(err)
			return
		}
		if len(projects) == 0 {
			fmt.Println("No projects configured.")
			return
		}
		ok, fail := 0, 0
		for _, p := range projects {
			fmt.Printf("\n── %s ──\n", p.Name)
			r, err := use.Prepare(p.Name, p.DefaultNamespace, skip)
			if err != nil {
				fmt.Printf("✗ %v\n", err)
				fail++
				continue
			}
			st := awsx.CredentialStatusForProfile(r.State.AWSProfile)
			if !st.Valid {
				fmt.Printf("✗ AWS: %s\n", st.Hint)
				fail++
				continue
			}
			if st.ExpiresIn != "" {
				fmt.Printf("  AWS: ✓ (%s)\n", st.ExpiresIn)
			} else {
				fmt.Printf("  AWS: ✓\n")
			}
			if err := clusterReachable(r.State); err != nil {
				fmt.Printf("✗ cluster: %v\n", err)
				fail++
				continue
			}
			fmt.Printf("  EKS: ✓ %s / %s\n", r.Project.Cluster, r.State.KubeContext)
			ok++
		}
		fmt.Printf("\nSync complete: %d ok, %d failed\n", ok, fail)
	},
}

func init() {
	syncCmd.Flags().BoolP("yes", "y", false, "Skip production confirmation")
}
