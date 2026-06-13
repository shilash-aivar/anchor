package cli

import (
	"ctxly/internal/kubecfg"
)

func lintIssues() ([]kubecfg.LintIssue, error) {
	return kubecfg.LintAll()
}
