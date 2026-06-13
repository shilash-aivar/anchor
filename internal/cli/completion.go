package cli

import (
	"anchor/internal/config"
	"anchor/internal/picker"
	"anchor/internal/session"

	"github.com/spf13/cobra"
)

func loadRecentEntries() ([]session.RecentEntry, error) {
	return session.LoadRecent()
}

func pickRecentLabel(labels []string) (string, error) {
	return picker.Choose("Recent project:", labels)
}

func projectNames() ([]string, error) {
	return config.ListProjects()
}

func completeProjects(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	names, err := projectNames()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var out []string
	for _, n := range names {
		if toComplete == "" || stringsHasPrefix(n, toComplete) {
			out = append(out, n)
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func completeNamespaces(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	s, err := session.Load()
	if err != nil || s == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names, err := listNamespaces(s)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var out []string
	for _, n := range names {
		if toComplete == "" || stringsHasPrefix(n, toComplete) {
			out = append(out, n)
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func registerCompletions() {
	rootCmd.RegisterFlagCompletionFunc("shell", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"zsh", "bash"}, cobra.ShellCompDirectiveNoFileComp
	})

	for _, c := range []*cobra.Command{projectUseCmd, withCmd, initCmd} {
		c.ValidArgsFunction = completeProjects
	}
	nsCmd.ValidArgsFunction = completeNamespaces
	projectUseCmd.RegisterFlagCompletionFunc("namespace", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeNamespaces(cmd, args, toComplete)
	})
	withCmd.RegisterFlagCompletionFunc("namespace", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeNamespaces(cmd, args, toComplete)
	})
}
