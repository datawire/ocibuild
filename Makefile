# Configuration

DESTDIR     ?=
prefix      ?= /usr/local
exec_prefix ?= $(prefix)
bindir      ?= $(prefix)/bin
datarootdir ?= $(prefix)/share
datadir     ?= $(datarootdir)

bash_completion_dir ?= $(datadir)/bash-completion/completions
fish_completion_dir ?= $(datadir)/fish/completions
zsh_completion_dir  ?= $(datadir)/zsh/site-functions

# Build

name = ocibuild
build: $(name)
build: completion.bash
build: completion.fish
build: completion.zsh
.PHONY: build

.$(name).stamp: FORCE
	go build -o $@ .
$(name): .$(name).stamp tools/bin/copy-ifchanged
	tools/bin/copy-ifchanged $< $@
completion.%: $(name) main_aux.go
	go run -tags=aux . completion $* > $@

# Install

install: $(DESTDIR)$(bindir)/$(name)
install: $(DESTDIR)$(bash_completion_dir)/$(name)
install: $(DESTDIR)$(fish_completion_dir)/$(name).fish
install: $(DESTDIR)$(zsh_completion_dir)/_$(name)
.PHONY: install

$(DESTDIR)$(bindir)/$(name): $(name)
	install -Dm755 $< $@
$(DESTDIR)$(bash_completion_dir)/$(name): completion.bash
	install -Dm7644 $< $@
$(DESTDIR)$(fish_completion_dir)/$(name).fish: completion.fish
	install -Dm7644 $< $@
$(DESTDIR)$(zsh_completion_dir)/_$(name): completion.zsh
	install -Dm7644 $< $@

# Check

check:
	go test -race ./...
.PHONY: check

lint: tools/bin/golangci-lint
	tools/bin/golangci-lint run ./...
.PHONY: lint

# Aux

tools/bin/%: tools/src/%/pin.go tools/src/%/go.mod
	cd $(<D) && GOOS= GOARCH= go build -o $(abspath $@) $$(sed -En 's,^import "(.*)".*,\1,p' pin.go)
tools/bin/%: tools/src/%.sh
	install -Dm755 $< $@

.DELETE_ON_ERROR:
.PHONY: FORCE
FORCE:
