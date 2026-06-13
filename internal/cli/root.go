package cli

import (
	"fmt"
	"os"

	"ctxly/internal/config"
	"ctxly/internal/session"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ctxly",
	Short: "DevOps context CLI — AWS profile, EKS cluster, and namespace in one session",
	Long: `ctxly is a session-first CLI for DevOps engineers working across
multiple AWS accounts and EKS clusters.

Switch everything at once:
  ctxly use my-client
  ctxly project use my-client

Then run daily ops:
  ctxly logs api
  ctxly exec
  ctxly ui
  ctxly pf svc/api 8080:80`,
	SilenceUsage: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "✗", err)
	os.Exit(1)
}

func activeSession() (*session.State, *config.Project, error) {
	s, err := session.Load()
	if err != nil {
		return nil, nil, err
	}
	if s == nil {
		return nil, nil, fmt.Errorf("no active project — run `ctxly use` or `ctxly project use <name>`")
	}
	p, err := config.LoadProject(s.Project)
	if err != nil {
		return nil, nil, err
	}
	return s, p, nil
}

func init() {
	rootCmd.AddCommand(
		loginCmd,
		statusCmd,
		doctorCmd,
		envCmd,
		logsCmd,
		execCmd,
		uiCmd,
		kCmd,
		helmCmd,
		applyCmd,
		onboardCmd,
		projectCmd,
		initCmd,
		withCmd,
		recentCmd,
		useCmd,
		pfCmd,
		watchCmd,
		eventsCmd,
		cpCmd,
		findCmd,
		debugCmd,
		shareCmd,
		linksCmd,
		shellCmd,
		lintCmd,
		pruneCmd,
		validateCmd,
	)
	registerCompletions()
}
