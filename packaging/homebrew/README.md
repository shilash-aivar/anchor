# Homebrew tap for ctxly

## Install from local clone (development)

```bash
cd /path/to/ctxly
brew install --build-from-source ./packaging/homebrew/ctxly.rb
```

## Publish a tap (team install)

1. Create a GitHub repo `homebrew-tap` (or `homebrew-ctxly`).
2. Copy `ctxly.rb` into the repo root or `Formula/ctxly.rb`.
3. Tag a release and update `url` + `sha256` in the formula:

```bash
git tag v0.2.0
git archive --format=tar.gz -o ctxly-0.2.0.tar.gz v0.2.0
shasum -a 256 ctxly-0.2.0.tar.gz
```

4. Teammates install with:

```bash
brew tap your-org/tap
brew install ctxly
```

## Dependencies

The formula installs **ctxly** only. These should already be on a DevOps machine:

| Tool | Purpose |
|------|---------|
| awscli | SSO login, EKS kubeconfig |
| kubectl | Cluster access |
| fzf | Interactive pickers (optional; falls back to numbered menu) |
| stern | `ctxly logs` |
| k9s | `ctxly ui` |

Recommended (not required by formula on Linux):

```bash
brew install stern k9s
```

## Completions

```bash
# zsh (Homebrew)
ctxly completion zsh > $(brew --prefix)/share/zsh/site-functions/_ctxly

# bash
ctxly completion bash > $(brew --prefix)/etc/bash_completion.d/ctxly
```

Or use the Makefile target from the repo: `make install-completions`.
