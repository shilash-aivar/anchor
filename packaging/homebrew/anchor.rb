class Anchor < Formula
  desc "Session-first DevOps CLI for AWS profile, EKS cluster, and namespace"
  homepage "https://github.com/shilash-aivar/anchor"
  license "MIT"
  version "1.0.0"

  on_macos do
    depends_on "go" => :build
    depends_on "awscli"
    depends_on "kubectl"
    depends_on "fzf"
    depends_on "stern"
    depends_on "k9s"
  end

  head "https://github.com/shilash-aivar/anchor.git", branch: "main"

  def install
    ldflags = "-s -w -X anchor/internal/cli.Version=#{version}"
    system "go", "build", "-buildvcs=false", "-ldflags", ldflags, "-o", bin/"anchor", "./cmd/anchor"
  end

  def caveats
    <<~EOS
      anchor stores config in ~/.config/anchor
      Legacy ~/.config/ctxly is migrated automatically on first run.

      Setup:
        anchor onboard
        anchor project add
        anchor login --all
        anchor use

      Completions:
        anchor completion zsh > $(brew --prefix)/share/zsh/site-functions/_anchor
    EOS
  end

  test do
    assert_match "anchor", shell_output("#{bin}/anchor --help")
    system bin/"anchor", "onboard"
  end
end
