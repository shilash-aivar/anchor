class Ctxly < Formula
  desc "Session-first DevOps CLI for AWS profile, EKS cluster, and namespace"
  homepage "https://github.com/your-org/ctxly"
  license "MIT"
  version "0.2.0"

  on_macos do
    depends_on "go" => :build
    depends_on "awscli"
    depends_on "kubectl"
    depends_on "fzf"
    depends_on "stern"
    depends_on "k9s"
  end

  on_linux do
    depends_on "go" => :build
    depends_on "awscli"
    depends_on "kubernetes-cli"
    depends_on "fzf"
  end

  # Replace URL and sha256 after publishing a release tarball:
  #   git tag v0.2.0 && git archive --format=tar.gz -o ctxly-0.2.0.tar.gz v0.2.0
  #   shasum -a 256 ctxly-0.2.0.tar.gz
  url "https://github.com/your-org/ctxly/archive/refs/tags/v0.2.0.tar.gz"
  sha256 "0000000000000000000000000000000000000000000000000000000000000000"

  # Local development install:
  #   brew install --build-from-source ./packaging/homebrew/ctxly.rb
  #
  # Or use HEAD while iterating:
  head "https://github.com/your-org/ctxly.git", branch: "main"

  def install
    ldflags = "-s -w -X ctxly/internal/cli.Version=#{version}"
    system "go", "build", "-ldflags", ldflags, "-o", bin/"ctxly", "./cmd/ctxly"
  end

  def caveats
    <<~EOS
      ctxly stores config in ~/.config/ctxly

      First-time setup:
        ctxly onboard
        ctxly project add
        ctxly login <aws-profile>
        ctxly project use <name>

      Shell completions:
        ctxly completion zsh > $(brew --prefix)/share/zsh/site-functions/_ctxly

      Optional zsh hook — auto-export env after project use:
        ctxly() {
          command ctxly "$@"
          if [[ "$1" == "project" && "$2" == "use" ]] || [[ "$1" == "recent" && "$3" == "--pick" ]]; then
            eval "$(command ctxly env --shell zsh)"
          fi
        }
    EOS
  end

  test do
    assert_match "ctxly", shell_output("#{bin}/ctxly --help")
    system bin/"ctxly", "onboard"
  end
end
