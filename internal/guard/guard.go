package guard

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"anchor/internal/config"
)

func ConfirmProjectSwitch(p *config.Project, globalConfirm bool) error {
	return ConfirmProjectSwitchTyped(p, globalConfirm, false, "")
}

// ConfirmProjectSwitchTyped validates production switch. skip bypasses; typed must match when confirmation is required.
func ConfirmProjectSwitchTyped(p *config.Project, globalConfirm, skip bool, typed string) error {
	if skip {
		return nil
	}
	if !p.ShouldConfirm(globalConfirm) {
		return nil
	}
	text := p.EffectiveConfirmText()
	if typed != "" {
		if strings.TrimSpace(typed) != text {
			return fmt.Errorf("confirmation failed — type %q", text)
		}
		return nil
	}
	fmt.Fprintf(os.Stderr, "⚠  Switching to %s environment (%s, tier: %s)\n", p.Name, p.Tier, p.Tier)
	fmt.Fprintf(os.Stderr, "Type %q to continue: ", text)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if strings.TrimSpace(line) != text {
		return fmt.Errorf("confirmation failed")
	}
	return nil
}
