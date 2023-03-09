# Copyright (C) 2021-2022  Ambassador Labs
# Copyright (C) 2023  Luke Shumaker <lukeshu@lukeshu.com>
#
# SPDX-License-Identifier: Apache-2.0

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

generate/files  = userdocs
generate/files += LICENSE.txt
generate/files += pkg/cliutil/LICENSE.pflag.txt

generate:
	$(MAKE) generate-clean
	$(MAKE) $(generate/files)
	$(MAKE) go-mod-tidy
.PHONY: generate

generate-clean:
	rm -rf $(generate/files)
.PHONY: generate-clean

LICENSE.txt:
	curl https://apache.org/licenses/LICENSE-2.0.txt >$@
pkg/cliutil/LICENSE.pflag.txt:
	curl https://raw.githubusercontent.com/spf13/pflag/ad68c28ee799163e627e77fcc6e8ecaa866e3535/LICENSE >$@

go-mod-tidy:
	go mod tidy
	go mod vendor
.PHONY: go-mod-tidy

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

ocibuild.cov: check
	test -e $@
	touch $@
check:
	go test -count=1 -coverprofile=ocibuild.cov -coverpkg=./... -race ./...
.PHONY: check

%.cov.html: %.cov
	go tool cover -html=$< -o=$@

lint: tools/bin/golangci-lint
	tools/bin/golangci-lint run ./...
.PHONY: lint

# Aux

tools/bin/%: tools/src/%/pin.go tools/src/%/go.mod
	cd $(<D) && GOOS= GOARCH= go build -o $(abspath $@) $$(sed -En 's,^import "(.*)".*,\1,p' pin.go)
tools/bin/crane: tools/bin/%: tools/src/%/pin.go go.mod
	cd $(<D) && GOOS= GOARCH= go build -o $(abspath $@) $$(sed -En 's,^import "(.*)".*,\1,p' pin.go)
tools/bin/%: tools/src/%.sh
	mkdir -p $(@D)
	install -m755 $< $@

.DELETE_ON_ERROR:
.PHONY: FORCE
FORCE:
