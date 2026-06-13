#!/usr/bin/env bash
# Print lines to add to ~/.zshrc for ctxly (does not modify files automatically).
set -euo pipefail

PREFIX="${HOME}/.local"

cat <<EOF
# ctxly — add to ~/.zshrc

export PATH="${PREFIX}/bin:\$PATH"
fpath=(${PREFIX}/share/zsh/site-functions \$fpath)
autoload -Uz compinit && compinit

# Auto-export AWS/K8s env after switching projects
ctxly() {
  command ${PREFIX}/bin/ctxly "\$@"
  if [[ "\$1" == "project" && "\$2" == "use" ]] || [[ "\$1" == "recent" && "\$2" == "--pick" ]]; then
    eval "\$(command ${PREFIX}/bin/ctxly env --shell zsh)"
  fi
}
EOF
