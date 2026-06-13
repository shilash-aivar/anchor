package picker

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func FzfEnabled() bool {
	if os.Getenv("ANCHOR_NO_FZF") != "" {
		return false
	}
	_, err := exec.LookPath("fzf")
	return err == nil
}

// Choose picks one item using fzf when available, otherwise a numbered menu.
func Choose(label string, items []string) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("no items to choose from")
	}
	if len(items) == 1 {
		return items[0], nil
	}
	if FzfEnabled() {
		if picked, err := chooseFzf(label, items); err == nil {
			return picked, nil
		}
	}
	return chooseNumber(label, items)
}

func chooseFzf(label string, items []string) (string, error) {
	prompt := label
	if prompt == "" {
		prompt = "select> "
	} else if !strings.HasSuffix(prompt, " ") {
		prompt += " "
	}

	cmd := exec.Command("fzf",
		"--prompt", prompt,
		"--height", "40%",
		"--reverse",
		"--border",
		"--info=inline",
	)
	cmd.Stdin = strings.NewReader(strings.Join(items, "\n"))
	cmd.Stderr = os.Stderr
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("selection cancelled")
	}
	picked := strings.TrimSpace(out.String())
	if picked == "" {
		return "", fmt.Errorf("selection cancelled")
	}
	return picked, nil
}

func chooseNumber(label string, items []string) (string, error) {
	fmt.Fprintf(os.Stderr, "%s\n", label)
	for i, item := range items {
		fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, item)
	}
	fmt.Fprint(os.Stderr, "Select [1]: ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return items[0], nil
	}
	n, err := strconv.Atoi(line)
	if err != nil || n < 1 || n > len(items) {
		return "", fmt.Errorf("invalid selection")
	}
	return items[n-1], nil
}
