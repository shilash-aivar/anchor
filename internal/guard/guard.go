package guard

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"ctxly/internal/config"
)

func ConfirmProjectSwitch(p *config.Project, globalConfirm bool) error {
	if !p.ShouldConfirm(globalConfirm) {
		return nil
	}
	text := p.EffectiveConfirmText()
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
