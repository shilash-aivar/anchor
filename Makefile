.PHONY: build install test tidy install-completions install-all uninstall

PREFIX ?= $(HOME)/.local
VERSION ?= 1.0.0
BINDIR := $(PREFIX)/bin

build:
	go build -buildvcs=false -ldflags "-s -w -X anchor/internal/cli.Version=$(VERSION)" -o bin/anchor ./cmd/anchor

install: build
	@mkdir -p $(BINDIR)
	install -m 755 bin/anchor $(BINDIR)/anchor
	@echo "Installed $(BINDIR)/anchor"

install-all: install install-completions
	@echo ""
	@echo "Add to ~/.zshrc:"
	@echo '  export PATH="$(HOME)/.local/bin:$$PATH"'

install-completions: build
	@mkdir -p $(PREFIX)/share/zsh/site-functions $(PREFIX)/share/bash-completion/completions
	./bin/anchor completion zsh > $(PREFIX)/share/zsh/site-functions/_anchor
	./bin/anchor completion bash > $(PREFIX)/share/bash-completion/completions/anchor
	@echo "Installed completions to $(PREFIX)/share/..."

uninstall:
	rm -f $(BINDIR)/anchor
	rm -f $(PREFIX)/share/zsh/site-functions/_anchor
	rm -f $(PREFIX)/share/bash-completion/completions/anchor

test:
	go test ./...

tidy:
	go mod tidy

brew-install:
	brew install --build-from-source ./packaging/homebrew/anchor.rb
