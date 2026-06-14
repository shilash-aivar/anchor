package dashboard

type Command struct {
	Name        string `json:"name"`
	Usage       string `json:"usage"`
	Short       string `json:"short"`
	Group       string `json:"group"`
	Interactive bool   `json:"interactive"`
	Example     string `json:"example,omitempty"`
	API         string `json:"api,omitempty"`
}

func AllCommands() []Command {
	return []Command{
		// Session
		{Name: "use", Usage: "anchor use [project] [--auto-login|--no-login]", Short: "Activate project (AWS + EKS + namespace)", Group: "Session", Example: "anchor use client-a", API: "POST /api/use"},
		{Name: "whoami", Usage: "anchor whoami [--json]", Short: "One-line active account + cluster", Group: "Session", API: "GET /api/session"},
		{Name: "start", Usage: "anchor start [-y]", Short: "Morning routine: login, sync, status", Group: "Session", API: "POST /api/sync"},
		{Name: "logout", Usage: "anchor logout [profile]", Short: "AWS SSO logout", Group: "Session", Interactive: true},
		{Name: "audit", Usage: "anchor audit [--today] [--lines N]", Short: "Read audit log", Group: "Session", API: "GET /api/audit"},
		{Name: "pin", Usage: "anchor pin add|remove|list", Short: "Pin favorite projects", Group: "Session", Example: "anchor pin add client-a"},
		{Name: "with-all", Usage: "anchor with-all '<pattern>' -- <cmd>", Short: "Run command across matching projects", Group: "Session", Example: "anchor with-all 'client-*' -- anchor k get pods"},
		{Name: "project discover", Usage: "anchor project discover [--dry-run]", Short: "Scan AWS profiles + EKS clusters", Group: "Session", Interactive: true, Example: "anchor project discover --pick"},
		{Name: "login profiles", Usage: "anchor login profiles list|sync", Short: "List or refresh AWS SSO profiles", Group: "Session", Interactive: true},
		{Name: "eks list", Usage: "anchor eks list [--profile] [--region]", Short: "List EKS clusters", Group: "Session", Example: "anchor eks list"},
		{Name: "project use", Usage: "anchor project use [name]", Short: "Same as use", Group: "Session", Example: "anchor project use client-a", API: "POST /api/use"},
		{Name: "project list", Usage: "anchor project list", Short: "List configured projects", Group: "Session", API: "GET /api/projects"},
		{Name: "project add", Usage: "anchor project add [name]", Short: "Create project interactively", Group: "Session", Interactive: true, Example: "anchor project add"},
		{Name: "project import", Usage: "anchor project import", Short: "Import from kubeconfig context", Group: "Session", Interactive: true},
		{Name: "project notes", Usage: "anchor project notes [name]", Short: "Show project notes", Group: "Session", API: "GET /api/projects"},
		{Name: "login", Usage: "anchor login [profile]", Short: "AWS SSO login", Group: "Session", Interactive: true, Example: "anchor login client-a-admin", API: "POST /api/login"},
		{Name: "login --all", Usage: "anchor login --all", Short: "SSO login for every project profile", Group: "Session", Interactive: true, API: "POST /api/login?all=true"},
		{Name: "status", Usage: "anchor status [--json]", Short: "Active session and AWS credential status", Group: "Session", API: "GET /api/session"},
		{Name: "doctor", Usage: "anchor doctor", Short: "Health check: tools, auth, cluster", Group: "Session", API: "GET /api/doctor"},
		{Name: "env", Usage: "anchor env --shell zsh", Short: "Print shell exports for active session", Group: "Session", Example: "eval \"$(anchor env --shell zsh)\""},
		{Name: "with", Usage: "anchor with <project> -- <cmd>", Short: "Run command in another project context", Group: "Session", Example: "anchor with client-b -- anchor k get pods"},
		{Name: "recent", Usage: "anchor recent [--pick]", Short: "Recent project + namespace combos", Group: "Session", API: "GET /api/recent"},
		{Name: "shell", Usage: "anchor shell [project]", Short: "Subshell with project environment", Group: "Session", Interactive: true},
		{Name: "share", Usage: "anchor share [--json]", Short: "Pasteable session block for Slack", Group: "Session", API: "GET /api/share"},
		{Name: "links", Usage: "anchor links [name] [--open|--copy]", Short: "Project bookmarks (grafana, runbook)", Group: "Session", Example: "anchor links grafana --open", API: "GET /api/projects"},
		{Name: "info", Usage: "anchor info", Short: "Cheat sheet: notes, links, quick commands", Group: "Session", API: "GET /api/overview"},
		{Name: "sync", Usage: "anchor sync [-y]", Short: "Refresh kubeconfigs and verify all projects", Group: "Session", API: "POST /api/sync"},
		{Name: "prompt", Usage: "anchor prompt --format segment|starship", Short: "Shell prompt segment output", Group: "Session"},
		{Name: "ns", Usage: "anchor ns [namespace]", Short: "Switch namespace in active project", Group: "Session", API: "POST /api/ns"},
		{Name: "init", Usage: "anchor init --project X", Short: "Write .ctx.yaml in repo", Group: "Session", Example: "anchor init --project client-a -n app"},
		{Name: "dashboard", Usage: "anchor dashboard [--port 8765]", Short: "Local web UI (this page)", Group: "Session", API: "GET /"},

		// Kubernetes ops
		{Name: "logs", Usage: "anchor logs <query>", Short: "Stern log tail", Group: "Kubernetes", Interactive: true, Example: "anchor logs api --since 1h"},
		{Name: "exec", Usage: "anchor exec [pod]", Short: "Exec into pod (picker if omitted)", Group: "Kubernetes", Interactive: true, Example: "anchor exec my-pod-abc"},
		{Name: "ui", Usage: "anchor ui", Short: "Launch k9s", Group: "Kubernetes", Interactive: true, Example: "anchor ui"},
		{Name: "k", Usage: "anchor k <args>", Short: "Guarded kubectl passthrough", Group: "Kubernetes", Interactive: true, Example: "anchor k get pods"},
		{Name: "helm", Usage: "anchor helm <args>", Short: "Guarded helm passthrough", Group: "Kubernetes", Interactive: true, Example: "anchor helm list -n app"},
		{Name: "apply", Usage: "anchor apply <args>", Short: "Guarded kubectl apply", Group: "Kubernetes", Interactive: true, Example: "anchor apply -f deploy.yaml"},
		{Name: "pf", Usage: "anchor pf <target> [ports]", Short: "Port-forward", Group: "Kubernetes", Interactive: true, Example: "anchor pf svc/api 8080:80"},
		{Name: "watch", Usage: "anchor watch <resource>", Short: "Rollout status or kubectl get -w", Group: "Kubernetes", Interactive: true, Example: "anchor watch deploy/api"},
		{Name: "events", Usage: "anchor events [--warnings]", Short: "Namespace events", Group: "Kubernetes", Interactive: true, Example: "anchor events --warnings"},
		{Name: "cp", Usage: "anchor cp <remote> [local]", Short: "Copy file from pod", Group: "Kubernetes", Interactive: true},
		{Name: "find", Usage: "anchor find <query>", Short: "Search pods, deployments, services", Group: "Kubernetes", API: "GET /api/find?q=", Example: "anchor find payment"},
		{Name: "debug", Usage: "anchor debug <pod>", Short: "kubectl debug wrapper", Group: "Kubernetes", Interactive: true},

		// Maintenance
		{Name: "lint", Usage: "anchor lint", Short: "Validate configs and kubeconfig hygiene", Group: "Maintenance", API: "GET /api/lint"},
		{Name: "prune", Usage: "anchor prune [--dry-run]", Short: "Remove orphan kubeconfigs", Group: "Maintenance", API: "POST /api/prune"},
		{Name: "validate", Usage: "anchor validate [path]", Short: "Validate .ctx.yaml and projects", Group: "Maintenance"},
		{Name: "onboard", Usage: "anchor onboard", Short: "First-time setup checklist", Group: "Maintenance"},
		{Name: "version", Usage: "anchor version", Short: "Print version", Group: "Maintenance"},
		{Name: "completion", Usage: "anchor completion zsh|bash|fish", Short: "Shell completion scripts", Group: "Maintenance"},
	}
}
