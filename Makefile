# Configuration

DESTDIR     ?=
prefix      ?= /usr/local
exec_prefix ?= $(prefix)
bindir      ?= $(exec_prefix)/bin
datarootdir ?= $(prefix)/share
datadir     ?= $(datarootdir)
mandir      ?= $(datarootdir)/man
man1dir     ?= $(mandir)/man1

bash_completion_dir ?= $(datadir)/bash-completion/completions
fish_completion_dir ?= $(datadir)/fish/completions
zsh_completion_dir  ?= $(datadir)/zsh/site-functions

# Build

name = ocibuild
build: $(name)
build: completion.bash
build: completion.fish
build: completion.zsh
build: man
build: userdocs
.PHONY: build

.$(name).stamp: FORCE
	go build -o $@ .
$(name): .$(name).stamp tools/bin/copy-ifchanged
	tools/bin/copy-ifchanged $< $@
completion.%: $(name) main_aux.go
	go run -tags=aux . completion $* > $@
man: $(name) main_aux.go
	go run -tags=aux . man $@ || { r=$$?; rm -rf $@; exit $$r; }
userdocs: $(name) main_aux.go
	go run -tags=aux . mddoc $@ || { r=$$?; rm -rf $@; exit $$r; }

# Generate

generate: userdocs
.PHONY: generate

# Install

install: $(DESTDIR)$(bindir)/$(name)
install: $(DESTDIR)$(bash_completion_dir)/$(name)
install: $(DESTDIR)$(fish_completion_dir)/$(name).fish
install: $(DESTDIR)$(zsh_completion_dir)/_$(name)
install: install-man
.PHONY: install

$(DESTDIR)$(bindir)/$(name): $(name)
	mkdir -p $(@D)
	install -m755 $< $@
$(DESTDIR)$(bash_completion_dir)/$(name): completion.bash
	mkdir -p $(@D)
	install -m644 $< $@
$(DESTDIR)$(fish_completion_dir)/$(name).fish: completion.fish
	mkdir -p $(@D)
	install -m644 $< $@
$(DESTDIR)$(zsh_completion_dir)/_$(name): completion.zsh
	mkdir -p $(@D)
	install -m644 $< $@

install-man: man
	mkdir -p $(DESTDIR)$(man1dir)
	install -m644 man/*.1 $(DESTDIR)$(man1dir)
.PHONY: install-man

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
	mkdir -p $(@D)
	install -m755 $< $@

.DELETE_ON_ERROR:
.PHONY: FORCE
FORCE:
