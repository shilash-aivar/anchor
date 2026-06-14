#!/usr/bin/env bash
# Print lines to add to ~/.zshrc for anchor (does not modify files automatically).
set -euo pipefail

PREFIX="${HOME}/.local"

cat <<EOF
# anchor — add to ~/.zshrc

export PATH="${PREFIX}/bin:\$PATH"
fpath=(${PREFIX}/share/zsh/site-functions \$fpath)
autoload -Uz compinit && compinit

# Show AWS account + EKS cluster in prompt (like git branch):
#   anchor prompt --format install >> ~/.zshrc
# Or paste the block from: anchor prompt --format install

anchor() {
  command ${PREFIX}/bin/anchor "\$@"
  if [[ "\$1" == "use" ]] || [[ "\$1" == "project" && "\$2" == "use" ]] || [[ "\$1" == "recent" && "\$2" == "--pick" ]]; then
    eval "\$(command ${PREFIX}/bin/anchor env --shell zsh)"
  fi
}
EOF

echo ""
echo "# Quick setup — append prompt context to ~/.zshrc:"
echo "#   anchor prompt --format install >> ~/.zshrc"
