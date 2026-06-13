package cli

import (
	"anchor/internal/kubecfg"
)

func lintIssues() ([]kubecfg.LintIssue, error) {
	return kubecfg.LintAll()
}
