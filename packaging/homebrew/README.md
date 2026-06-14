# Homebrew tap for anchor

## Install from local clone (development)

```bash
cd /path/to/anchor
brew install --build-from-source ./packaging/homebrew/anchor.rb
```

This installs **anchor** plus all runtime dependencies:

| Package | Purpose |
|---------|---------|
| awscli | SSO login, EKS kubeconfig |
| kubectl / kubernetes-cli | Cluster access |
| fzf | Interactive pickers |
| stern | `anchor logs` |
| k9s | `anchor ui` |
| helm | `anchor helm` |

## From source without Homebrew formula

```bash
make install-all    # builds anchor + brew install deps + completions
anchor onboard --install
```

## Publish a tap (team install)

1. Create a GitHub repo `homebrew-tap` (or `homebrew-anchor`).
2. Copy `anchor.rb` into the repo root or `Formula/anchor.rb`.
3. Tag a release and update `url` + `sha256` in the formula.

4. Teammates install with:

```bash
brew tap your-org/tap
brew install anchor
```

## Completions

```bash
# zsh (Homebrew)
anchor completion zsh > $(brew --prefix)/share/zsh/site-functions/_anchor

# bash
anchor completion bash > $(brew --prefix)/etc/bash_completion.d/anchor
```

Or: `make install-completions`
