package audit

import (
	"bufio"
	"os"
	"strings"
	"time"

	"anchor/internal/config"
)

func ReadLines(max int, todayOnly bool) ([]string, error) {
	path, err := config.AuditPath()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if todayOnly {
			prefix := time.Now().UTC().Format("2006-01-02")
			if !strings.HasPrefix(line, prefix) {
				continue
			}
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if max > 0 && len(lines) > max {
		lines = lines[len(lines)-max:]
	}
	return lines, nil
}
