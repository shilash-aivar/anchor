package cli

import (
	"fmt"
	"os"
	"strings"

	"anchor/internal/config"
	"anchor/internal/session"

	"github.com/spf13/cobra"
)

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Print prompt segments for Starship or bash/zsh",
	Run: func(cmd *cobra.Command, args []string) {
		format, _ := cmd.Flags().GetString("format")
		switch format {
		case "starship":
			printStarshipConfig()
		case "segment":
			printPromptSegment()
		default:
			exitErr(fmt.Errorf("unknown format %q (use starship or segment)", format))
		}
	},
}

func printPromptSegment() {
	marker, _ := session.ActiveMarkerPath()
	data, err := os.ReadFile(marker)
	if err != nil {
		return
	}
	parts := strings.Split(strings.TrimSpace(string(data)), "|")
	if len(parts) < 3 {
		return
	}
	project, ctx, ns := parts[0], parts[1], parts[2]
	p, _ := config.LoadProject(project)
	tier := ""
	if p != nil {
		tier = p.Tier
	}
	style := ""
	if tier == "production" {
		style = "bold red"
		fmt.Printf("[#anchor segment]\nstyle=%s\n", style)
	}
	fmt.Printf("[%s · %s · %s](%s)\n", project, ctx, ns, style)
}

func printStarshipConfig() {
	fmt.Print(`# Add to ~/.config/starship.toml
[custom.anchor]
command = "anchor prompt --format segment"
when = "exec anchor"
description = "Active anchor project/session"
`)
}

func init() {
	promptCmd.Flags().String("format", "segment", "Output format: segment, starship")
}
