.PHONY: build install test tidy install-completions install-all uninstall

PREFIX ?= $(HOME)/.local
VERSION ?= 0.3.0
BINDIR := $(PREFIX)/bin

build:
	go build -buildvcs=false -ldflags "-s -w -X ctxly/internal/cli.Version=$(VERSION)" -o bin/ctxly ./cmd/ctxly

# Install binary to ~/.local/bin (recommended)
install: build
	@mkdir -p $(BINDIR)
	install -m 755 bin/ctxly $(BINDIR)/ctxly
	@echo "Installed $(BINDIR)/ctxly"
	@echo ""
	@echo "Add to ~/.zshrc if needed:"
	@echo '  export PATH="$(HOME)/.local/bin:$$PATH"'

# Install binary + shell completions
install-all: install install-completions
	@echo ""
	@echo "Add to ~/.zshrc if needed:"
	@echo '  export PATH="$(HOME)/.local/bin:$$PATH"'
	@echo '  fpath=($(HOME)/.local/share/zsh/site-functions $$fpath)'
	@echo '  autoload -Uz compinit && compinit'

install-completions: build
	@mkdir -p $(PREFIX)/share/zsh/site-functions $(PREFIX)/share/bash-completion/completions
	./bin/ctxly completion zsh > $(PREFIX)/share/zsh/site-functions/_ctxly
	./bin/ctxly completion bash > $(PREFIX)/share/bash-completion/completions/ctxly
	@echo "Installed completions to $(PREFIX)/share/..."

uninstall:
	rm -f $(BINDIR)/ctxly
	rm -f $(PREFIX)/share/zsh/site-functions/_ctxly
	rm -f $(PREFIX)/share/bash-completion/completions/ctxly

test:
	go test ./...

tidy:
	go mod tidy

brew-install:
	brew install --build-from-source ./packaging/homebrew/ctxly.rb
