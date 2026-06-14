# anchor

**Session-first DevOps CLI** — switch AWS profile, EKS cluster, and namespace in one command.

## Install

### From source (recommended for dev)

```bash
cd anchor
make install-all          # anchor binary + stern/k9s/helm/etc + completions
./scripts/shell-setup.sh  # paste into ~/.zshrc
source ~/.zshrc
anchor onboard            # verify everything is on PATH
```

`make install-all` runs `brew install awscli kubectl fzf stern k9s helm` if Homebrew is available.

Install dependencies only:

```bash
make install-deps
# or
anchor onboard --install
```

### Homebrew

```bash
brew install --build-from-source ./packaging/homebrew/anchor.rb
```

The Homebrew formula pulls in **awscli, kubectl, fzf, stern, k9s, and helm** automatically.

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
| `use --auto-login` | SSO login if AWS credentials expired or expiring |
| `use --no-login` | Fail instead of prompting SSO when expired |
| `whoami [--json]` | PRD/STG/DEV · account · profile · cluster · namespace |
| `start [-y]` | Morning routine: login check, sync, status |
| `logout [profile] [--all]` | AWS SSO logout (clears anchor session by default) |
| `audit [--today] [--lines N]` | Read audit log |
| `pin add\|remove\|list` | Pin favorite projects to top of picker |
| `eks list` | List EKS clusters for profile/region |
| `with-all '<glob>' -- <cmd>` | Fan-out across projects (`--tier`, `--continue`, glob) |
| `project use [name]` | Same as `use` |
| `project add` | Interactive new project |
| `project import` | Import from existing kubeconfig context |
| `project list` | List projects |
| `project discover [--dry-run] [--pick]` | Scan `~/.aws/config` + EKS → project yaml |
| `project discover --include-no-cluster` | Also create profile-only projects |
| `project notes` | Show notes |
| `with <proj> -- <cmd>` | Run once without changing session |
| `recent [--pick]` | Recent project+namespace |
| `shell [project] [--refresh]` | Isolated subshell (strips stale AWS creds) |
| `login [profile]` | AWS SSO login |
| `login --all` | SSO for every project profile |
| `login --missing` | SSO only for expired/expiring profiles |
| `login --status [--json]` | Show credential status per profile (no browser) |
| `login profiles list` | List all AWS profiles + SSO status |
| `login profiles import` | **Create** SSO account/role profiles in `~/.aws/config` |
| `login profiles sync` | SSO login for every project profile |
| `status [--json]` | Active session (+ SSO expiry) |
| `sync` | Refresh kubeconfigs and verify AWS for all projects |
| `prompt --format plain\|zsh\|starship\|install` | AWS account + cluster in shell prompt (like git branch) |
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
| `info` | Project cheat sheet (notes, links, quick commands) |
| `dashboard` | Local web UI for session, projects, and all commands |

### Maintenance
| Command | Description |
|---------|-------------|
| `lint` | Config + kubeconfig hygiene |
| `prune [--dry-run]` | Remove orphan kubeconfigs |
| `validate [path]` | Validate `.ctx.yaml` + projects |
| `onboard [--install]` | Dependency checklist; `--install` brew-installs missing tools |
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
vpn_required: false   # set true for private EKS endpoints (blocks use until cluster reachable)
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
  auto_login_on_use: false    # SSO login automatically when using expired creds
  protect_context_regex: "(prod|production|live)"  # extra guard on matching contexts

sso:
  start_url: https://your-org.awsapps.com/start   # used by login profiles import
  region: us-east-1                               # SSO region (not default AWS region)

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

Strict mode (block mutating kubectl/helm on protected contexts when non-interactive): `export ANCHOR_STRICT=1`

## Prompt context (like git branch)

Show which AWS account and EKS cluster you're on in every terminal line:

```bash
anchor use client-a
anchor prompt --format install >> ~/.zshrc   # one-time setup
source ~/.zshrc
```

Example prompt:

```text
~/code/myapp [PRD] (client-a · 123456789012 · client-a-prod · app) »
```

Tier badges: **PRD** (red), **STG** (yellow), **DEV** (cyan).

```bash
anchor prompt --format plain        # one-line text with PRD/STG/DEV badge
anchor prompt --format zsh-right    # show on right side (RPROMPT)
anchor prompt --format starship     # Starship config snippet
```

### Auto-discover workflow (awsx / ksw / awsctx style)

```bash
# 1. Import all SSO account/role profiles into ~/.aws/config
anchor login profiles import

# 2. Scan profiles + EKS clusters → write ~/.config/anchor/projects/*.yaml
anchor project discover --pick

# 3. Enable auto SSO on switch (optional)
# options.auto_login_on_use: true   # or: anchor use --auto-login
anchor use my-project
```

**Cursor IDE status line** (bottom bar, like git branch):

```json
// ~/.cursor/cli-config.json
"statusLine": {
  "type": "command",
  "command": "/path/to/anchor/scripts/cursor-statusline.sh"
}
```

See [docs/REQUIREMENTS.md](docs/REQUIREMENTS.md) and [packaging/homebrew/README.md](packaging/homebrew/README.md).

## License

MIT
