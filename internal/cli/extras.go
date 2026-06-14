package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"anchor/internal/audit"
	"anchor/internal/awsx"
	"anchor/internal/config"
	"anchor/internal/discover"
	"anchor/internal/prompt"
	"anchor/internal/session"
	"anchor/internal/use"

	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show active AWS account, profile, cluster, and namespace (one line)",
	Run: func(cmd *cobra.Command, args []string) {
		jsonOut, _ := cmd.Flags().GetBool("json")
		info, err := prompt.LoadFromMarker()
		if err != nil || info == nil {
			info, err = prompt.Load()
		}
		if err != nil || info == nil {
			exitErr(fmt.Errorf("no active project — run `anchor use`"))
			return
		}
		acct := info.AccountID
		if acct == "" {
			acct = info.Profile
		}
		cluster := info.Cluster
		if cluster == "" {
			cluster = info.Context
		}
		badge := config.TierAbbrev(info.Tier)
		if jsonOut {
			fmt.Printf(`{"tier":%q,"badge":%q,"project":%q,"account":%q,"profile":%q,"cluster":%q,"namespace":%q}`+"\n",
				info.Tier, badge, info.Project, acct, info.Profile, cluster, info.Namespace)
			return
		}
		fmt.Printf("%s · %s · %s · %s · %s\n", badge, acct, info.Profile, cluster, info.Namespace)
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Morning routine: check login, sync projects, show status",
	Run: func(cmd *cobra.Command, args []string) {
		skip, _ := cmd.Flags().GetBool("yes")
		if err := loginStatus(false); err != nil {
			exitErr(err)
			return
		}
		if err := loginProfiles(false, true, false); err != nil {
			fmt.Fprintf(os.Stderr, "⚠ login: %v\n", err)
		}
		res := syncAllCLI(skip)
		fmt.Printf("\nSync: %d ok, %d failed\n", res.ok, res.fail)
		if err := runStatus(false); err != nil {
			fmt.Fprintf(os.Stderr, "status: %v\n", err)
		}
		fmt.Println("\n→ anchor use   or   anchor dashboard")
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout [profile]",
	Short: "Log out of AWS SSO session",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		all, _ := cmd.Flags().GetBool("all")
		keepSession, _ := cmd.Flags().GetBool("keep-session")
		profile := ""
		if len(args) == 1 {
			profile = args[0]
		} else if !all {
			if s, _, err := activeSession(); err == nil && s != nil {
				profile = s.AWSProfile
			}
		}
		if err := awsx.SSOLogout(profile); err != nil {
			exitErr(err)
		}
		if all {
			fmt.Println("✓ logged out all SSO sessions")
		} else if profile != "" {
			fmt.Printf("✓ logged out profile %s\n", profile)
		} else {
			fmt.Println("✓ logged out SSO session")
		}
		if !keepSession {
			_ = session.Clear()
			fmt.Println("✓ cleared active anchor session")
			fmt.Println("→ in your shell: unset AWS_PROFILE KUBECONFIG KUBE_NAMESPACE ANCHOR_PROJECT")
		}
	},
}

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Show audit log entries",
	Run: func(cmd *cobra.Command, args []string) {
		today, _ := cmd.Flags().GetBool("today")
		lines, _ := cmd.Flags().GetInt("lines")
		entries, err := audit.ReadLines(lines, today)
		if err != nil {
			exitErr(err)
			return
		}
		if len(entries) == 0 {
			fmt.Println("(empty)")
			return
		}
		for _, e := range entries {
			fmt.Println(e)
		}
	},
}

var pinCmd = &cobra.Command{
	Use:   "pin",
	Short: "Pin favorite projects to top of picker",
}

var pinAddCmd = &cobra.Command{
	Use:   "add <project>",
	Short: "Pin a project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := config.LoadProject(args[0]); err != nil {
			exitErr(err)
			return
		}
		if err := config.PinProject(args[0]); err != nil {
			exitErr(err)
			return
		}
		fmt.Printf("✓ pinned %s\n", args[0])
	},
}

var pinRemoveCmd = &cobra.Command{
	Use:     "remove <project>",
	Aliases: []string{"rm"},
	Short:   "Unpin a project",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.UnpinProject(args[0]); err != nil {
			exitErr(err)
			return
		}
		fmt.Printf("✓ unpinned %s\n", args[0])
	},
}

var pinListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pinned projects",
	Run: func(cmd *cobra.Command, args []string) {
		p, err := config.LoadPins()
		if err != nil {
			exitErr(err)
			return
		}
		if len(p.Projects) == 0 {
			fmt.Println("(none)")
			return
		}
		for _, n := range p.Projects {
			fmt.Println(n)
		}
	},
}

var eksCmd = &cobra.Command{
	Use:   "eks",
	Short: "EKS cluster helpers",
}

var eksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List EKS clusters for active session or profile",
	Run: func(cmd *cobra.Command, args []string) {
		profile, _ := cmd.Flags().GetString("profile")
		region, _ := cmd.Flags().GetString("region")
		if profile == "" {
			s, _, err := activeSession()
			if err != nil {
				exitErr(err)
				return
			}
			profile = s.AWSProfile
			if region == "" {
				region = s.AWSRegion
			}
		}
		if region == "" {
			region = awsx.ProfileRegion(profile)
		}
		clusters, err := awsx.ListEKSClusters(profile, region)
		if err != nil {
			exitErr(err)
			return
		}
		if len(clusters) == 0 {
			fmt.Println("(no clusters)")
			return
		}
		for _, c := range clusters {
			fmt.Println(c)
		}
	},
}

var withAllCmd = &cobra.Command{
	Use:   "with-all <pattern> -- <command>",
	Short: "Run a command across matching projects (glob or substring)",
	Example: `  anchor with-all 'client-*' -- anchor k get pods
  anchor with-all '*-prod' --tier production -- anchor status`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runWithAll(cmd, args); err != nil {
			exitErr(err)
		}
	},
}

func runWithAll(cmd *cobra.Command, args []string) error {
	sep := -1
	for i, a := range args {
		if a == "--" {
			sep = i
			break
		}
	}
	if sep < 1 {
		return fmt.Errorf("usage: anchor with-all <pattern> -- <command> [args...]")
	}
	pattern := args[0]
	command := args[sep+1]
	cmdArgs := args[sep+2:]
	tierFilter, _ := cmd.Flags().GetString("tier")
	cont, _ := cmd.Flags().GetBool("continue")
	skip, _ := cmd.Flags().GetBool("yes")
	opts := useOptsFrom(cmd, skip)

	names, err := config.ListProjects()
	if err != nil {
		return err
	}
	matched := matchProjects(names, pattern, tierFilter)
	if len(matched) == 0 {
		return fmt.Errorf("no projects match %q", pattern)
	}
	fail := 0
	for _, name := range matched {
		fmt.Printf("\n── %s ──\n", name)
		r, err := use.Prepare(name, "", opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ %v\n", err)
			fail++
			if !cont {
				return err
			}
			continue
		}
		if err := use.RunCommand(r, command, cmdArgs); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %v\n", err)
			fail++
			if !cont {
				return err
			}
		}
	}
	if fail > 0 {
		return fmt.Errorf("%d project(s) failed", fail)
	}
	return nil
}

func matchProjects(names []string, pattern, tierFilter string) []string {
	pattern = strings.Trim(pattern, "'\"")
	tierFilter = config.NormalizeTier(tierFilter)
	useGlob := strings.ContainsAny(pattern, "*?[")
	var out []string
	for _, n := range names {
		if tierFilter != "" {
			p, err := config.LoadProject(n)
			if err != nil || config.NormalizeTier(p.Tier) != tierFilter {
				continue
			}
		}
		ln := strings.ToLower(n)
		lp := strings.ToLower(pattern)
		if useGlob {
			ok, err := filepath.Match(lp, ln)
			if err == nil && ok {
				out = append(out, n)
			}
			continue
		}
		if strings.Contains(ln, lp) || strings.HasPrefix(ln, strings.TrimSuffix(lp, "*")) {
			out = append(out, n)
		}
	}
	return out
}

type syncResult struct {
	ok, fail int
}

func syncAllCLI(skip bool) syncResult {
	projects, err := config.LoadAllProjects()
	if err != nil {
		return syncResult{}
	}
	res := syncResult{}
	for _, p := range projects {
		if _, err := use.Prepare(p.Name, p.DefaultNamespace, use.Options{SkipConfirm: skip}); err != nil {
			res.fail++
		} else {
			res.ok++
		}
	}
	return res
}

func initExtras() {
	whoamiCmd.Flags().Bool("json", false, "JSON output")
	logoutCmd.Flags().Bool("all", false, "Log out of all SSO sessions")
	logoutCmd.Flags().Bool("keep-session", false, "Keep active anchor session after SSO logout")
	auditCmd.Flags().Bool("today", false, "Only today's entries")
	auditCmd.Flags().Int("lines", 50, "Max lines to show")
	eksListCmd.Flags().String("profile", "", "AWS profile")
	eksListCmd.Flags().String("region", "", "AWS region")
	withAllCmd.Flags().String("tier", "", "Filter by tier (production, staging, development)")
	withAllCmd.Flags().Bool("continue", false, "Continue if a project fails")
	withAllCmd.Flags().BoolP("yes", "y", false, "Skip production confirmation")
	registerAutoLoginFlags(withAllCmd)

	pinCmd.AddCommand(pinAddCmd, pinRemoveCmd, pinListCmd)
	eksCmd.AddCommand(eksListCmd)
	startCmd.Flags().BoolP("yes", "y", false, "Skip production confirmation during sync")
}

func projectDiscoverRun(dryRun, pick, includeNoCluster bool) error {
	candidates, err := discover.ScanProfiles()
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		fmt.Println("No AWS profiles found in ~/.aws/config.")
		return nil
	}

	var needsLogin []discover.Candidate
	for _, c := range candidates {
		if c.NeedsLogin {
			needsLogin = append(needsLogin, c)
		}
	}
	if len(needsLogin) > 0 {
		fmt.Printf("%d profile(s) need SSO login before full scan:\n", len(needsLogin))
		for _, c := range needsLogin {
			fmt.Printf("  ✗ %s — run `anchor login %s`\n", c.Profile, c.Profile)
		}
		fmt.Println("  Tip: anchor login profiles import  then  anchor login --missing")
	}

	var toCreate []discover.Candidate
	for _, c := range candidates {
		if c.Exists {
			continue
		}
		if c.Cluster == "" && !includeNoCluster {
			continue
		}
		if c.NeedsLogin {
			continue
		}
		toCreate = append(toCreate, c)
	}
	if len(toCreate) == 0 {
		fmt.Println("No new projects to create.")
		return nil
	}

	labels := make([]string, len(toCreate))
	for i, c := range toCreate {
		label := fmt.Sprintf("%s | %s", c.SuggestedName, c.Profile)
		if c.Cluster != "" {
			label += " | " + c.Cluster
		} else {
			label += " | (no cluster)"
		}
		if c.VPNRequired {
			label += " | vpn"
		}
		labels[i] = label
	}
	selected := toCreate
	if pick {
		label, err := pickerChoose("Select project to import:", labels)
		if err != nil {
			return err
		}
		idx := indexOf(labels, label)
		selected = []discover.Candidate{toCreate[idx]}
	}

	for _, c := range selected {
		if dryRun {
			fmt.Printf("would create: %s profile=%s cluster=%s vpn=%v\n", c.SuggestedName, c.Profile, c.Cluster, c.VPNRequired)
			continue
		}
		p := &config.Project{
			Name:         c.SuggestedName,
			AWSProfile:   c.Profile,
			AccountID:    c.AccountID,
			Region:       c.Region,
			Tier:         config.NormalizeTier(discover.TierFromName(c.SuggestedName + c.Cluster)),
			Cluster:      c.Cluster,
			ContextAlias: c.SuggestedName,
			VPNRequired:  c.VPNRequired,
		}
		if err := config.SaveProject(p); err != nil {
			return err
		}
		fmt.Printf("✓ created project %s\n", p.Name)
	}
	return nil
}

func indexOf(list []string, v string) int {
	for i, s := range list {
		if s == v {
			return i
		}
	}
	return 0
}

// avoid import cycle - thin wrapper set in discover_cmd init
var pickerChoose func(prompt string, items []string) (string, error)
