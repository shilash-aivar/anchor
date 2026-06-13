# Add to .envrc in a repo with .anchor.yaml or .ctx.yaml (legacy)

use_anchor() {
  if [[ -f .anchor.yaml ]] || [[ -f .ctx.yaml ]]; then
    anchor use --from-repo >/dev/null
    eval "$(anchor env --shell zsh)"
  fi
}

# direnv: allow and add `use_anchor` to .envrc
