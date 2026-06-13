# Completions

Generate and install shell completions:

```bash
# zsh
ctxly completion zsh > ~/.zsh/completions/_ctxly
# add to .zshrc: fpath=(~/.zsh/completions $fpath); autoload -Uz compinit && compinit

# bash
ctxly completion bash > /etc/bash_completion.d/ctxly

# or from repo
make install-completions
```

## Completed arguments

- `ctxly project use <name>` — project names from `~/.config/ctxly/projects/`
- `ctxly with <project>` — project names
- `ctxly ns <namespace>` — namespaces in active cluster (requires active session)
- `ctxly init --project` — project names

Disable fzf pickers: `export CTXLY_NO_FZF=1`
