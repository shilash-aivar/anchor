# Completions

Generate and install shell completions:

```bash
# zsh
anchor completion zsh > ~/.zsh/completions/_anchor
# add to .zshrc: fpath=(~/.zsh/completions $fpath); autoload -Uz compinit && compinit

# bash
anchor completion bash > /etc/bash_completion.d/anchor

# or from repo
make install-completions
```

## Completed arguments

- `anchor project use <name>` — project names from `~/.config/anchor/projects/`
- `anchor with <project>` — project names
- `anchor ns <namespace>` — namespaces in active cluster (requires active session)
- `anchor init --project` — project names

Disable fzf pickers: `export ANCHOR_NO_FZF=1`
