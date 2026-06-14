package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"anchor/internal/awsx"
	"anchor/internal/config"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login [profile]",
	Short: "Authenticate with AWS SSO",
	Long: `Runs aws sso login for a profile.

  anchor login                  # active session profile
  anchor login my-profile       # specific profile
  anchor login --all            # every profile used by projects
  anchor login --missing        # only profiles that need refresh
  anchor login --status         # show credential status (no browser)`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		all, _ := cmd.Flags().GetBool("all")
		missing, _ := cmd.Flags().GetBool("missing")
		status, _ := cmd.Flags().GetBool("status")
		jsonOut, _ := cmd.Flags().GetBool("json")
		continueOnErr, _ := cmd.Flags().GetBool("continue")

		if status {
			if err := loginStatus(jsonOut); err != nil {
				exitErr(err)
			}
			return
		}
		if missing || all {
			if err := loginProfiles(all, missing, continueOnErr); err != nil {
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

type profileLoginStatus struct {
	Profile   string `json:"profile"`
	Valid     bool   `json:"valid"`
	Account   string `json:"account_id,omitempty"`
	ExpiresIn string `json:"expires_in,omitempty"`
	Hint      string `json:"hint,omitempty"`
	Projects  []string `json:"projects,omitempty"`
}

func projectProfiles() (map[string][]string, []string, error) {
	projects, err := config.LoadAllProjects()
	if err != nil {
		return nil, nil, err
	}
	byProfile := map[string][]string{}
	var order []string
	seen := map[string]bool{}
	for _, p := range projects {
		if p.AWSProfile == "" {
			continue
		}
		byProfile[p.AWSProfile] = append(byProfile[p.AWSProfile], p.Name)
		if !seen[p.AWSProfile] {
			seen[p.AWSProfile] = true
			order = append(order, p.AWSProfile)
		}
	}
	sort.Strings(order)
	return byProfile, order, nil
}

func loginStatus(jsonOut bool) error {
	byProfile, order, err := projectProfiles()
	if err != nil {
		return err
	}
	if len(order) == 0 {
		return fmt.Errorf("no projects configured")
	}

	var rows []profileLoginStatus
	needLogin := 0
	for _, profile := range order {
		st := awsx.CredentialStatusForProfile(profile)
		row := profileLoginStatus{
			Profile:   profile,
			Valid:     st.Valid,
			Account:   st.Account,
			ExpiresIn: st.ExpiresIn,
			Hint:      st.Hint,
			Projects:  byProfile[profile],
		}
		rows = append(rows, row)
		if awsx.NeedsLogin(profile, 30*time.Minute) {
			needLogin++
		}
	}

	if jsonOut {
		enc, _ := json.MarshalIndent(rows, "", "  ")
		fmt.Println(string(enc))
		return nil
	}

	fmt.Println("AWS SSO status (project profiles):")
	for _, row := range rows {
		icon := "✓"
		if !row.Valid {
			icon = "✗"
		} else if row.Hint != "" {
			icon = "⚠"
		}
		line := fmt.Sprintf("  %s %-24s", icon, row.Profile)
		if row.Account != "" {
			line += fmt.Sprintf(" account=%s", row.Account)
		}
		if row.ExpiresIn != "" {
			line += fmt.Sprintf(" (%s)", row.ExpiresIn)
		}
		fmt.Println(line)
		if row.Hint != "" {
			fmt.Printf("      %s\n", row.Hint)
		}
		if len(row.Projects) > 0 {
			fmt.Printf("      projects: %s\n", strings.Join(row.Projects, ", "))
		}
	}
	fmt.Println()
	if needLogin > 0 {
		fmt.Printf("%d profile(s) need login — run: anchor login --missing\n", needLogin)
	} else {
		fmt.Println("All profiles OK.")
	}
	return nil
}

func loginProfiles(all, missingOnly, continueOnErr bool) error {
	_, order, err := projectProfiles()
	if err != nil {
		return err
	}
	if len(order) == 0 {
		return fmt.Errorf("no projects configured")
	}

	var targets []string
	if missingOnly {
		for _, profile := range order {
			if awsx.NeedsLogin(profile, 30*time.Minute) {
				targets = append(targets, profile)
			}
		}
	} else {
		targets = order
	}
	if len(targets) == 0 {
		fmt.Println("✓ all profiles have valid credentials")
		return nil
	}

	fail := 0
	for _, profile := range targets {
		fmt.Printf("\n── profile %s ──\n", profile)
		if err := awsx.SSOLogin(profile); err != nil {
			fail++
			fmt.Fprintf(os.Stderr, "✗ %v\n", err)
			if !continueOnErr {
				return err
			}
			continue
		}
	}
	if fail > 0 {
		return fmt.Errorf("%d profile(s) failed to login", fail)
	}
	fmt.Fprintln(os.Stderr, "\n✓ login complete")
	return nil
}

func init() {
	loginCmd.Flags().Bool("all", false, "Login every AWS profile used by configured projects")
	loginCmd.Flags().Bool("missing", false, "Login only profiles with expired or expiring credentials")
	loginCmd.Flags().Bool("status", false, "Show SSO credential status (no browser)")
	loginCmd.Flags().Bool("json", false, "JSON output (with --status)")
	loginCmd.Flags().Bool("continue", false, "With --all or --missing, continue if a profile fails")
}
