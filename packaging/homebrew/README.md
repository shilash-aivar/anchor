# Homebrew tap for anchor

## Install from local clone (development)

```bash
cd /path/to/anchor
brew install --build-from-source ./packaging/homebrew/anchor.rb
```

## Publish a tap (team install)

1. Create a GitHub repo `homebrew-tap` (or `homebrew-anchor`).
2. Copy `anchor.rb` into the repo root or `Formula/anchor.rb`.
3. Tag a release and update `url` + `sha256` in the formula:

```bash
git tag v0.2.0
git archive --format=tar.gz -o anchor-0.2.0.tar.gz v0.2.0
shasum -a 256 anchor-0.2.0.tar.gz
```

4. Teammates install with:

```bash
brew tap your-org/tap
brew install anchor
```

## Dependencies

The formula installs **anchor** only. These should already be on a DevOps machine:

| Tool | Purpose |
|------|---------|
| awscli | SSO login, EKS kubeconfig |
| kubectl | Cluster access |
| fzf | Interactive pickers (optional; falls back to numbered menu) |
| stern | `anchor logs` |
| k9s | `anchor ui` |

Recommended (not required by formula on Linux):

```bash
brew install stern k9s
```

## Completions

```bash
# zsh (Homebrew)
anchor completion zsh > $(brew --prefix)/share/zsh/site-functions/_anchor

# bash
anchor completion bash > $(brew --prefix)/etc/bash_completion.d/anchor
```

Or use the Makefile target from the repo: `make install-completions`.
