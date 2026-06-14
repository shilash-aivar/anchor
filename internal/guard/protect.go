package guard

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"anchor/internal/config"
	"anchor/internal/session"
)

func StrictMode() bool {
	return os.Getenv("ANCHOR_STRICT") != "" || os.Getenv("ANCHOR_STRICT") == "1"
}

func IsProtected(p *config.Project, cfg *config.Config, s *session.State) bool {
	if p != nil && p.IsProduction() {
		return true
	}
	if cfg == nil || cfg.Options.ProtectContextRegex == "" || s == nil {
		return false
	}
	re, err := regexp.Compile(cfg.Options.ProtectContextRegex)
	if err != nil {
		return false
	}
	for _, target := range []string{s.Project, s.KubeContext, s.Namespace, p.Cluster} {
		if target != "" && re.MatchString(target) {
			return true
		}
	}
	return false
}

func ConfirmMutating(s *session.State, p *config.Project, cfg *config.Config, verb string, args []string) error {
	if !isMutatingVerb(verb, args) {
		return nil
	}
	if !IsProtected(p, cfg, s) {
		return nil
	}
	if StrictMode() && !isInteractive() {
		return fmt.Errorf("ANCHOR_STRICT: mutating command blocked on protected context (non-interactive)")
	}
	if StrictMode() && isInteractive() {
		// still require confirm in strict interactive mode for protected
	}
	line := strings.Join(append([]string{verb}, args...), " ")
	fmt.Fprintf(os.Stderr, "⚠  Protected context — confirm mutating command:\n  %s\nType 'yes' to continue: ", line)
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if strings.TrimSpace(text) != "yes" {
		return fmt.Errorf("command cancelled")
	}
	return nil
}

func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
