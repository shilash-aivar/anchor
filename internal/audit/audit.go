package audit

import (
	"fmt"
	"os"
	"time"

	"ctxly/internal/config"
)

func Log(action, detail string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if cfg != nil && !cfg.Options.AuditLog {
		return nil
	}
	path, err := config.AuditPath()
	if err != nil {
		return err
	}
	if _, err := config.EnsureConfigDir(); err != nil {
		return err
	}
	user := os.Getenv("USER")
	if user == "" {
		user = "unknown"
	}
	line := fmt.Sprintf("%s user=%s action=%s %s\n",
		time.Now().UTC().Format(time.RFC3339), user, action, detail)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}
