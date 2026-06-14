package cli

import (
	"fmt"

	"anchor/internal/awsx"
	"anchor/internal/config"
	"anchor/internal/picker"

	"github.com/spf13/cobra"
)

var loginProfilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Import, list, or refresh AWS SSO profiles",
}

var loginProfilesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List AWS profiles from ~/.aws/config",
	Run: func(cmd *cobra.Command, args []string) {
		profiles, err := awsx.ListProfiles()
		if err != nil {
			exitErr(err)
			return
		}
		for _, p := range profiles {
			marker := " "
			if awsx.IsSSOProfile(p) {
				marker = "S"
			}
			st := awsx.CredentialStatusForProfile(p)
			status := "✓"
			if !st.Valid {
				status = "✗"
			}
			fmt.Printf("%s %s %s %s\n", status, marker, p, st.ExpiresIn)
		}
	},
}

var loginProfilesImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Create SSO profiles in ~/.aws/config from your SSO portal",
	Long: `Logs into SSO and writes a profile for each account/role pair.

Requires at least one SSO profile in ~/.aws/config (or sso.start_url in anchor config).

  anchor login profiles import
  anchor login profiles import --dry-run
  anchor login profiles import --profile bootstrap-admin`,
	Run: func(cmd *cobra.Command, args []string) {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		bootstrap, _ := cmd.Flags().GetString("profile")
		cfg, _ := config.LoadConfig()
		opts := awsx.SSOImportOptions{
			StartURL:  cfg.SSO.StartURL,
			SSORegion: cfg.SSO.Region,
			Bootstrap: bootstrap,
			DryRun:    dryRun,
		}
		res, err := awsx.ImportSSOProfiles(opts)
		if err != nil {
			exitErr(err)
			return
		}
		if len(res.Created) == 0 && len(res.Skipped) == 0 {
			fmt.Println("No profiles created.")
			return
		}
		if !dryRun {
			fmt.Printf("\n✓ created %d profile(s)", len(res.Created))
			if len(res.Skipped) > 0 {
				fmt.Printf(", skipped %d existing", len(res.Skipped))
			}
			fmt.Println()
			fmt.Println("→ anchor project discover")
		}
	},
}

var loginProfilesSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "SSO login for all profiles used by anchor projects",
	Run: func(cmd *cobra.Command, args []string) {
		cont, _ := cmd.Flags().GetBool("continue")
		if err := loginProfiles(true, false, cont); err != nil {
			exitErr(err)
		}
	},
}

var projectDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Scan ~/.aws/config + EKS clusters and create project yaml",
	Run: func(cmd *cobra.Command, args []string) {
		dry, _ := cmd.Flags().GetBool("dry-run")
		pick, _ := cmd.Flags().GetBool("pick")
		includeNoCluster, _ := cmd.Flags().GetBool("include-no-cluster")
		if err := projectDiscoverRun(dry, pick, includeNoCluster); err != nil {
			exitErr(err)
		}
	},
}

func initDiscoverLogin() {
	pickerChoose = picker.Choose
	loginProfilesCmd.AddCommand(loginProfilesListCmd, loginProfilesImportCmd, loginProfilesSyncCmd)
	loginCmd.AddCommand(loginProfilesCmd)
	projectCmd.AddCommand(projectDiscoverCmd)
	projectDiscoverCmd.Flags().Bool("dry-run", false, "Show what would be created")
	projectDiscoverCmd.Flags().Bool("pick", false, "Interactively pick one")
	projectDiscoverCmd.Flags().Bool("include-no-cluster", false, "Create projects for profiles without EKS clusters")
	loginProfilesSyncCmd.Flags().Bool("continue", false, "Continue if a profile fails")
	loginProfilesImportCmd.Flags().Bool("dry-run", false, "Show profiles that would be created")
	loginProfilesImportCmd.Flags().String("profile", "", "Bootstrap SSO profile (default: first matching sso.start_url)")
}
