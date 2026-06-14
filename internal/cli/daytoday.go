package cli

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"


	"github.com/spf13/cobra"
)

var linksCmd = &cobra.Command{
	Use:   "links [name]",
	Short: "List or open project links (grafana, runbook, etc.)",
	Long:  "Links come from the active project's links: map in ~/.config/anchor/projects/<name>.yaml.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		open, _ := cmd.Flags().GetBool("open")
		copyURL, _ := cmd.Flags().GetBool("copy")
		if err := runLinks(args, open, copyURL); err != nil {
			exitErr(err)
		}
	},
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Project cheat sheet: notes, links, and handy commands",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runInfo(); err != nil {
			exitErr(err)
		}
	},
}

func runLinks(args []string, open, copyURL bool) error {
	s, p, err := activeSession()
	if err != nil {
		return err
	}
	if p == nil {
		return fmt.Errorf("no active project")
	}
	if len(p.Links) == 0 {
		fmt.Fprintf(os.Stderr, "No links configured for %q — add a links: section to the project yaml\n", s.Project)
		return nil
	}

	names := sortedLinkNames(p.Links)
	if len(args) == 0 {
		fmt.Printf("Links for %s:\n", s.Project)
		for _, name := range names {
			fmt.Printf("  %-12s %s\n", name+":", p.Links[name])
		}
		fmt.Fprintln(os.Stderr, "\nOpen:  anchor links <name> --open")
		fmt.Fprintln(os.Stderr, "Copy:  anchor links <name> --copy")
		return nil
	}

	name := args[0]
	url, ok := p.Links[name]
	if !ok {
		return fmt.Errorf("unknown link %q — available: %s", name, strings.Join(names, ", "))
	}
	if copyURL {
		if err := copyToClipboard(url); err != nil {
			fmt.Println(url)
			return err
		}
		fmt.Printf("Copied %s\n", url)
		return nil
	}
	if open {
		return openURL(url)
	}
	fmt.Println(url)
	return nil
}

func runInfo() error {
	s, p, err := activeSession()
	if err != nil {
		return err
	}
	if p == nil {
		return fmt.Errorf("no active project")
	}

	fmt.Printf("── %s (%s) ──\n", s.Project, s.Tier)
	fmt.Printf("AWS  %s / %s\n", s.AWSProfile, s.AWSRegion)
	if s.AccountID != "" {
		fmt.Printf("     account %s\n", s.AccountID)
	}
	fmt.Printf("EKS  %s\n", p.Cluster)
	fmt.Printf("K8s  context=%s namespace=%s\n\n", s.KubeContext, s.Namespace)

	if strings.TrimSpace(p.Notes) != "" {
		fmt.Println("Notes")
		for _, line := range strings.Split(strings.TrimSpace(p.Notes), "\n") {
			fmt.Printf("  %s\n", line)
		}
		fmt.Println()
	}

	if len(p.Links) > 0 {
		fmt.Println("Links")
		for _, name := range sortedLinkNames(p.Links) {
			fmt.Printf("  %-12s %s\n", name+":", p.Links[name])
		}
		fmt.Println()
	}

	fmt.Println("Quick commands")
	fmt.Printf("  anchor logs <app>          # stern in %s\n", s.Namespace)
	fmt.Printf("  anchor exec [pod]          # shell into pod\n")
	fmt.Printf("  anchor pf svc/<name> 8080  # port-forward\n")
	fmt.Printf("  anchor watch deploy/<name> # rollout status\n")
	fmt.Printf("  anchor links grafana --open\n")
	if len(p.Links) > 0 {
		first := sortedLinkNames(p.Links)[0]
		fmt.Printf("  anchor links %s --open\n", first)
	}
	return nil
}

func sortedLinkNames(links map[string]string) []string {
	names := make([]string, 0, len(links))
	for name := range links {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func openURL(url string) error {
	var cmd *exec.Cmd
	switch {
	case commandExists("open"):
		cmd = exec.Command("open", url)
	case commandExists("xdg-open"):
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("cannot open browser — copy URL: %s", url)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch {
	case commandExists("pbcopy"):
		cmd = exec.Command("pbcopy")
	case commandExists("xclip"):
		cmd = exec.Command("xclip", "-selection", "clipboard")
	case commandExists("wl-copy"):
		cmd = exec.Command("wl-copy")
	default:
		return fmt.Errorf("no clipboard tool (pbcopy/xclip) — URL printed above")
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func init() {
	linksCmd.Flags().Bool("open", false, "Open link in default browser")
	linksCmd.Flags().Bool("copy", false, "Copy link URL to clipboard")
}
