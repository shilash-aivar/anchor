# anchor

**Session-first DevOps CLI** — switch AWS profile, EKS cluster, and namespace in one command.

## Install

```bash
cd anchor
make install-all          # ~/.local/bin/anchor + completions
./scripts/shell-setup.sh  # paste into ~/.zshrc
source ~/.zshrc
```

## Quick start

```bash
anchor onboard
anchor project add
anchor login --all
anchor use                   # fzf picker, or reads .ctx.yaml in repo
anchor status
```

## All commands

### Session
| Command | Description |
|---------|-------------|
| `use [project]` | Activate project (fzf / `.ctx.yaml` / picker) |
| `project use [name]` | Same as `use` |
| `project add` | Interactive new project |
| `project import` | Import from existing kubeconfig context |
| `project list` | List projects |
| `project notes` | Show notes |
| `with <proj> -- <cmd>` | Run once without changing session |
| `recent [--pick]` | Recent project+namespace |
| `shell [project]` | Subshell with env loaded |
| `login [profile]` | AWS SSO |
| `login --all` | SSO for every project profile |
| `status [--json]` | Active session (+ SSO expiry) |
| `sync` | Refresh kubeconfigs and verify AWS for all projects |
| `prompt --format segment\|starship` | Shell / Starship prompt segment |
| `share [--json]` | Pasteable session block |
| `doctor` | Health checks |
| `env --shell zsh` | Shell exports |
| `init --project X` | Write `.ctx.yaml` in repo |

### Kubernetes ops
| Command | Description |
|---------|-------------|
| `ns [name]` | Switch namespace (fzf) |
| `logs <query>` | Stern wrapper (`--previous`, `--since`) |
| `exec [pod]` | Exec (+ container picker) |
| `ui` | Launch k9s |
| `k <args>` | kubectl passthrough (guarded) |
| `helm <args>` | helm passthrough (guarded) |
| `apply <args>` | Guarded kubectl apply |
| `pf <target> [ports]` | Port-forward |
| `watch <resource>` | Rollout status / get -w |
| `events [--warnings]` | Namespace events |
| `cp <remote> [local]` | Copy from pod |
| `find <query>` | Search pods/deploy/svc |
| `debug <pod>` | kubectl debug |
| `links [name] [--open]` | Project URLs from config |

### Maintenance
| Command | Description |
|---------|-------------|
| `lint` | Config + kubeconfig hygiene |
| `prune [--dry-run]` | Remove orphan kubeconfigs |
| `validate [path]` | Validate `.ctx.yaml` + projects |
| `onboard` | Dependency checklist |
| `version` | Print version (1.0.0) |

## Config

```text
~/.config/anchor/
  config.yaml       # global options + hooks
  state.json        # active session
  audit.log         # optional audit trail
  projects/*.yaml   # your projects
  kube/*.yaml       # isolated kubeconfigs
```

### Project example

```yaml
name: client-a
aws_profile: client-a-admin
account_id: "123456789012"
region: us-east-1
tier: production
cluster: client-a-prod
context_alias: client-a
default_namespace: app
readonly: false
require_confirm: true
notes: |
  Deploy: helm upgrade app ./chart -n app
links:
  grafana: https://grafana.example.com/d/client-a
  runbook: https://wiki.example.com/runbooks/client-a
env:
  HELM_VALUES: ./values/client-a.yaml
```

### Global options (`config.yaml`)

```yaml
options:
  confirm_production: true
  announce_context: true
  audit_log: true
  block_dangerous: true
  dry_run_production: false   # apply uses --dry-run=server on prod

hooks:
  pre_use: ~/scripts/anchor-pre-use.sh
  post_use: ""
  pre_apply: ""
```

## Shell hook

```bash
anchor() {
  command anchor "$@"
  if [[ "$1" == "use" || "$1" == "project" && "$2" == "use" || "$1" == "recent" && "$2" == "--pick" ]]; then
    eval "$(command anchor env --shell zsh)"
  fi
}
```

Disable fzf: `export ANCHOR_NO_FZF=1`

See [docs/REQUIREMENTS.md](docs/REQUIREMENTS.md) and [packaging/homebrew/README.md](packaging/homebrew/README.md).

## License

MIT
