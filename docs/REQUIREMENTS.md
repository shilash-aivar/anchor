# ctxly requirements

Session-first CLI for DevOps engineers juggling multiple AWS accounts and EKS clusters.

## Implemented (v0.3)

### Session & projects
| Feature | Command |
|---------|---------|
| Atomic project switch | `use`, `project use` |
| Repo binding | `init`, `use --from-repo`, `.ctx.yaml` walk-up |
| Project registry | `project add/list/notes/import` |
| One-shot env | `with` |
| Recent combos | `recent`, `recent --pick` |
| Subshell | `shell`, `shell <project>` |
| SSO login | `login`, `login --all` |

### Daily ops
| Feature | Command |
|---------|---------|
| Logs (stern) | `logs`, `logs --previous` |
| Exec + multi-container picker | `exec`, `exec -c` |
| k9s | `ui` |
| kubectl | `k` |
| helm | `helm` |
| Port-forward | `pf` |
| Rollout watch | `watch` |
| Events | `events`, `events --warnings` |
| File copy | `cp` |
| Resource search | `find` |
| Debug | `debug` |

### Safety
| Feature | Command |
|---------|---------|
| Prod switch confirm | tier=production |
| Guarded apply | `apply` (+ optional dry-run prod) |
| Read-only project | `readonly: true` in project yaml |
| Block dangerous deletes | `block_dangerous` option |
| Context announce | `announce_context` on mutating ops |
| Audit log | `~/.config/ctxly/audit.log` |
| Plugin hooks | `hooks.pre_use/post_use/pre_apply` |

### Tooling
| Feature | Command |
|---------|---------|
| Status / JSON | `status`, `status --json`, `share --json` |
| Doctor | `doctor` |
| Lint | `lint` |
| Prune orphans | `prune`, `prune --dry-run` |
| Validate | `validate` |
| Share for Slack | `share` |
| Project links | `links`, `links grafana --open` |
| fzf pickers | `CTXLY_NO_FZF=1` to disable |
| Completions | `completion zsh/bash/fish` |
| Homebrew | `packaging/homebrew/ctxly.rb` |

## Out of scope

- Replacing k9s or stern
- Web UI / central auth server
- GitOps pipelines
