package cli

import (
	"fmt"
	"os"

	"ctxly/internal/awsx"
	"ctxly/internal/config"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login [profile]",
	Short: "Authenticate with AWS SSO",
	Long:  "Runs `aws sso login`. Pass a profile name, --all for every project profile, or omit for active session.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		all, _ := cmd.Flags().GetBool("all")
		if all {
			if err := loginAllProfiles(); err != nil {
				exitErr(err)
			}
			return
		}
		profile := ""
		if len(args) > 0 {
			profile = args[0]
		} else if s, _, err := activeSession(); err == nil && s != nil {
			profile = s.AWSProfile
		}
		if err := runLogin(profile); err != nil {
			exitErr(err)
		}
	},
}

func runLogin(profile string) error {
	if !awsx.AWSAvailable() {
		return fmt.Errorf("aws CLI not found — install AWS CLI v2")
	}
	if profile != "" {
		fmt.Printf("→ aws sso login --profile %s\n", profile)
	} else {
		fmt.Println("→ aws sso login")
	}
	return awsx.SSOLogin(profile)
}

func loginAllProfiles() error {
	projects, err := config.LoadAllProjects()
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		return fmt.Errorf("no projects configured")
	}
	seen := map[string]bool{}
	for _, p := range projects {
		if p.AWSProfile == "" || seen[p.AWSProfile] {
			continue
		}
		seen[p.AWSProfile] = true
		fmt.Printf("\n── profile %s ──\n", p.AWSProfile)
		if err := awsx.SSOLogin(p.AWSProfile); err != nil {
			return err
		}
	}
	fmt.Fprintln(os.Stderr, "\n✓ logged in all project profiles")
	return nil
}

func init() {
	loginCmd.Flags().Bool("all", false, "Login every AWS profile used by configured projects")
}
