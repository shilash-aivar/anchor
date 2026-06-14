package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"anchor/internal/dashboard"

	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Launch local web dashboard",
	Long:  "Starts a local web UI at http://127.0.0.1:8765 with session status, projects, links, and all anchor commands.",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		noOpen, _ := cmd.Flags().GetBool("no-open")
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		opts := dashboard.Options{
			Addr:    addr,
			Version: Version,
			Open:    !noOpen,
		}
		go func() {
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
			<-sig
			os.Exit(0)
		}()
		if err := dashboard.Run(opts); err != nil {
			exitErr(err)
		}
	},
}

func init() {
	dashboardCmd.Flags().IntP("port", "p", 8765, "HTTP port")
	dashboardCmd.Flags().Bool("no-open", false, "Do not open browser automatically")
}
