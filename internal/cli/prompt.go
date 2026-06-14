package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"anchor/internal/prompt"

	"github.com/spf13/cobra"
)

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Show AWS account + cluster context in your shell (like git branch)",
	Long: `Prints the active anchor session for shell prompts and status lines.

Formats:
  plain       One line: ⚓ project · account · cluster · namespace
  zsh         Colored segment for left prompt (git-branch style)
  zsh-right   Colored segment for zsh RPROMPT (right side)
  bash        ANSI segment for bash PS1
  segment     Starship custom module output
  starship    Print starship.toml snippet
  install     Print shell setup for zsh/bash (append to ~/.zshrc)
  json        JSON for scripts / Cursor status line`,
	Run: func(cmd *cobra.Command, args []string) {
		format, _ := cmd.Flags().GetString("format")
		shell, _ := cmd.Flags().GetString("shell")
		switch format {
		case "plain":
			runPrompt(printPlain)
		case "zsh":
			runPrompt(prompt.ZshLeft)
		case "zsh-right":
			runPrompt(prompt.ZshRPrompt)
		case "bash":
			runPrompt(prompt.BashPS1)
		case "segment":
			runPrompt(prompt.StarshipSegment)
		case "starship":
			printStarshipConfig()
		case "install":
			printPromptInstall(shell)
		case "json":
			runPromptJSON()
		default:
			exitErr(fmt.Errorf("unknown format %q — try plain, zsh, bash, starship, install", format))
		}
	},
}

func runPrompt(fn func(prompt.Info)) {
	info, err := loadPromptInfo()
	if err != nil {
		exitErr(err)
		return
	}
	if info == nil {
		return
	}
	fn(*info)
}

func runPromptJSON() {
	info, err := loadPromptInfo()
	if err != nil {
		exitErr(err)
		return
	}
	if info == nil {
		fmt.Println("{}")
		return
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(info)
}

func loadPromptInfo() (*prompt.Info, error) {
	if info, err := prompt.LoadFromMarker(); err != nil {
		return nil, err
	} else if info != nil {
		return info, nil
	}
	return prompt.Load()
}

func printPlain(info prompt.Info) {
	fmt.Println(prompt.Plain(info))
}

func printStarshipConfig() {
	fmt.Print(`# Add to ~/.config/starship.toml
# Shows AWS account + EKS cluster on the right (like git branch)

[custom.anchor]
command = "anchor prompt --format segment"
when = "true"
format = " [$symbol$output]($style) "
symbol = "⚓ "
style = "bold cyan"
description = "AWS account + EKS cluster (anchor session)"
`)
}

func printPromptInstall(shell string) {
	if shell == "" {
		shell = "zsh"
	}
	switch shell {
	case "zsh":
		io.WriteString(os.Stdout, zshPromptHook)
	case "bash":
		io.WriteString(os.Stdout, bashPromptHook)
	default:
		exitErr(fmt.Errorf("unsupported shell %q (use zsh or bash)", shell))
	}
}

const zshPromptHook = `# anchor — show AWS account + cluster in prompt (like git branch)
anchor_prompt_context() {
  local line
  line=$(command anchor prompt --format zsh 2>/dev/null) || return
  print -Pn "$line"
}

setopt PROMPT_SUBST
PROMPT='%F{blue}%~%f $(anchor_prompt_context) » '

# Show on the right instead (uncomment):
# RPROMPT='$(command anchor prompt --format zsh-right 2>/dev/null)'
`

const bashPromptHook = `# anchor — show AWS account + cluster in prompt
anchor_prompt_context() {
  anchor prompt --format bash 2>/dev/null
}
PS1='\[\033[1;34m\]\w\[\033[0m\] $(anchor_prompt_context) » '
`

func init() {
	promptCmd.Flags().String("format", "plain", "Output: plain, zsh, zsh-right, bash, segment, starship, install, json")
	promptCmd.Flags().String("shell", "zsh", "Shell for --format install")
}
