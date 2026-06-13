package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "0.3.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}

func init() {
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("ctxly {{.Version}}\n")
	rootCmd.AddCommand(versionCmd)
}
