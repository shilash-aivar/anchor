package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"anchor/internal/awsx"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show active AWS profile, cluster, and namespace",
	Run: func(cmd *cobra.Command, args []string) {
		jsonOut, _ := cmd.Flags().GetBool("json")
		if err := runStatus(jsonOut); err != nil {
			exitErr(err)
		}
	},
}

func runStatus(jsonOut bool) error {
	s, p, err := activeSession()
	if err != nil {
		return err
	}

	type statusJSON struct {
		Project    string `json:"project"`
		Tier       string `json:"tier"`
		Namespace  string `json:"namespace"`
		Context    string `json:"context"`
		Cluster    string `json:"cluster"`
		AWSProfile string `json:"aws_profile"`
		AWSRegion  string `json:"aws_region"`
		AccountID  string `json:"account_id,omitempty"`
		Kubeconfig string `json:"kubeconfig"`
		ReadOnly   bool   `json:"readonly"`
		UpdatedAt  string `json:"updated_at"`
		AWSValid   bool   `json:"aws_valid"`
		AWSARN     string `json:"aws_arn,omitempty"`
		AWSExpires string `json:"aws_expires,omitempty"`
	}

	st := statusJSON{
		Project:    s.Project,
		Tier:       s.Tier,
		Namespace:  s.Namespace,
		Context:    s.KubeContext,
		Cluster:    p.Cluster,
		AWSProfile: s.AWSProfile,
		AWSRegion:  s.AWSRegion,
		AccountID:  s.AccountID,
		Kubeconfig: s.Kubeconfig,
		ReadOnly:   p.ReadOnly,
		UpdatedAt:  s.UpdatedAt.Format(timeRFC3339),
	}

	cred := awsx.CredentialStatusForProfile(s.AWSProfile)
	st.AWSValid = cred.Valid
	st.AWSExpires = cred.ExpiresIn
	if cred.Valid {
		if id, err := awsx.GetCallerIdentity(s.AWSProfile); err == nil {
			st.AWSARN = id.ARN
		}
	}

	if jsonOut {
		enc, _ := json.MarshalIndent(st, "", "  ")
		fmt.Println(string(enc))
		return nil
	}

	fmt.Printf("Project:    %s (%s)\n", s.Project, s.Tier)
	if p.ReadOnly {
		fmt.Printf("Mode:       read-only\n")
	}
	fmt.Printf("AWS:        profile=%s region=%s\n", s.AWSProfile, s.AWSRegion)
	if s.AccountID != "" {
		fmt.Printf("            account=%s\n", s.AccountID)
	}
	fmt.Printf("EKS:        %s\n", p.Cluster)
	fmt.Printf("K8s:        context=%s namespace=%s\n", s.KubeContext, s.Namespace)
	fmt.Printf("Kubeconfig: %s\n", s.Kubeconfig)
	fmt.Printf("Updated:    %s\n", s.UpdatedAt.Format("2006-01-02 15:04:05 UTC"))
	if cred.Valid {
		line := cred.ExpiresIn
		if line == "" {
			line = "ok"
		}
		fmt.Printf("AWS creds:  ✓ %s", st.AWSARN)
		if cred.ExpiresIn != "" {
			fmt.Printf(" (%s)", cred.ExpiresIn)
		}
		fmt.Println()
		if cred.Hint != "" {
			fmt.Fprintf(os.Stderr, "            ⚠ %s\n", cred.Hint)
		}
	} else {
		fmt.Fprintf(os.Stderr, "AWS creds:  ✗ %s\n", cred.Hint)
	}
	return nil
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check AWS credentials, kubeconfig, and cluster connectivity",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDoctor(); err != nil {
			exitErr(err)
		}
	},
}

func runDoctor() error {
	s, _, err := activeSession()
	if err != nil {
		fmt.Println("Session:    ✗ no active project")
	} else {
		fmt.Printf("Session:    ✓ %s / %s / %s\n", s.Project, s.KubeContext, s.Namespace)
	}

	check := func(name string, ok bool, hint string) {
		if ok {
			fmt.Printf("%-12s ✓\n", name+":")
		} else {
			fmt.Printf("%-12s ✗ %s\n", name+":", hint)
		}
	}

	check("aws", awsx.AWSAvailable(), "install AWS CLI v2")
	check("kubectl", kubectlOK(), "install kubectl")
	check("fzf", toolOK("fzf"), "optional: brew install fzf")
	for _, tool := range []struct{ name, pkg string }{
		{"stern", "brew install stern"},
		{"k9s", "brew install k9s"},
		{"helm", "brew install helm"},
	} {
		check(tool.name, toolOK(tool.name), tool.pkg)
	}

	if s == nil {
		return nil
	}

	cred := awsx.CredentialStatusForProfile(s.AWSProfile)
	if !cred.Valid {
		fmt.Printf("aws-auth:   ✗ %s\n", cred.Hint)
	} else if id, err := awsx.GetCallerIdentity(s.AWSProfile); err != nil {
		fmt.Printf("aws-auth:   ✗ run `anchor login %s`\n", s.AWSProfile)
	} else {
		fmt.Printf("aws-auth:   ✓ account %s", id.Account)
		if cred.ExpiresIn != "" {
			fmt.Printf(" (%s)", cred.ExpiresIn)
		}
		fmt.Println()
		if s.AccountID != "" && id.Account != s.AccountID {
			fmt.Printf("            ⚠ profile account %s ≠ project account %s\n", id.Account, s.AccountID)
		}
	}

	if err := clusterReachable(s); err != nil {
		fmt.Printf("cluster:    ✗ %v\n", err)
	} else {
		fmt.Printf("cluster:    ✓ reachable\n")
	}

	issues, _ := lintIssues()
	if len(issues) > 0 {
		fmt.Printf("lint:       ⚠ %d issue(s) — run `anchor lint`\n", len(issues))
	} else {
		fmt.Printf("lint:       ✓\n")
	}

	return nil
}

func init() {
	statusCmd.Flags().Bool("json", false, "JSON output")
}
