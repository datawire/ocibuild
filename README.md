# ocibuild

[![PkgGoDev](https://pkg.go.dev/badge/github.com/datawire/ocibuild)](https://pkg.go.dev/github.com/datawire/ocibuild/pkg)
[![Go Report Card](https://goreportcard.com/badge/github.com/datawire/ocibuild)](https://goreportcard.com/report/github.com/datawire/ocibuild)
[![Quality Assurance](https://github.com/datawire/ocibuild/actions/workflows/qa.yml/badge.svg)](https://github.com/datawire/ocibuild/actions)
[![Coverage Status](https://coveralls.io/repos/github/datawire/ocibuild/badge.svg)](https://coveralls.io/github/datawire/ocibuild)

`ocibuild` is a command-line tool for manipulating Docker image layers
as files.  It pairs well with the [`crane`][] tool for interacting
with remote Docker images and registries.

## Documentation

Read the command documentation right here in your web browser or text
editor at [`./userdocs/`][], or in your terminal with `man ocibuild`
or `ocibuild --help`.

## Installation

### Minimal install

If you want just the executable binary (for CI or as an
automatically-downloaded component in your larger build system), you
can grab it the usual `go get` way:

```shell
go get github.com/datawire/ocibuild
```

### Complete install

If you want not just the executable, but also shell completion and
man-pages and whatnot, clone the repo, and run `make install`.  It
understands `DESTDIR` and `prefix` `bindir` and everything else that
37 years of GNU has made you come to expect.  No `./configure`
nescessary!

```shell
git clone https://github.com/datawire/ocibuild
cd ocibuild
make build
sudo make install prefix=/usr
```

or

```shell
git clone https://github.com/datawire/ocibuild
cd ocibuild
make install prefix=$HOME/.local
```

## Examples

### Complex example

It is possible to use Bash process substitution (the `<(command)`
construct) to elegantly build "complex" images.

This example:
 - pulls down Alpine to use as the base image
 - adds a single layer containing several third-party Python packages
 - adds a single layer containing both a locally built Python package (that
   presumably makes use of the third-party packages), and a locally
   built Go binary
 - pushes the resulting image up to DockerHub

Why anything would want such a silly [architecture][Emissary], I'm not
sure :P

```bash
crane push \
  <(ocibuild image build \
      --config.Entrypoint=/usr/local/bin/go-program-that-calls-python \
      --base=<(crane pull docker.io/alpine:latest /dev/stdout) \
      <(ocibuild layer squash \
          <(ocibuild layer wheel --platform-file=python.yml <(curl https://files.pythonhosted.org/packages/af/f4/524415c0744552cce7d8bf3669af78e8a069514405ea4fcbd0cc44733744/urllib3-1.26.7-py2.py3-none-any.whl)) \
          <(ocibuild layer wheel --platform-file=python.yml <(curl https://files.pythonhosted.org/packages/69/bf/f0f194d3379d3f3347478bd267f754fc68c11cbf2fe302a6ab69447b1417/beautifulsoup4-4.10.0-py3-none-any.whl))) \
      <(ocibuild layer squash \
          <(ocibuild layer wheel --platform-file=python.yml ./python/mypackage.whl) \
          <(ocibuild layer gobuild ./cmd/go-program-that-calls-python))) \
  docker.io/datawire/ocibuild-example:latest
```

Now, in actual use you probably wouldn't want to make as heavy use of
process substitution, and instead cache things to files, in order to
better support incremental builds.  Also, working with pipes instead
of regular files uses more memory, because the whole file must be kept
in memory after reading it, rather than being able to seek around on
disk.

In ([Emissary's][Emissary] style of) Make, this example would look
more like

```Makefile
ocibuild-example.img.tar: $(tools/ocibuild) base.img.tar python-deps.layer.tar my-code.layer.tar
	{ $(tools/ocibuild) image build \
	  --config.Entrypoint=/usr/local/bin/go-program-that-calls-python \
	  --base=$(filter %.img.tar,$^) \
	  $(filter %.layer.tar,$^); } >$@

base.img.tar: $(tools/crane)
	$(tools/crane) pull docker.io/alpine:latest >$@

%.whl.layer.tar: %.whl python.yml $(tools/ocibuild)
	$(tools/ocibuild) layer wheel --platform-file=python.yml $< >$@

pypi.urllib-1.26,7-py2.py3-none-any     = af/f4/524415c0744552cce7d8bf3669af78e8a069514405ea4fcbd0cc44733744
pypi.beautifulsoup4-4.10.0-py3-none-any = 69/bf/f0f194d3379d3f3347478bd267f754fc68c11cbf2fe302a6ab69447b1417
pypi-downloads/%.whl:
	curl https://files.pythonhosted.org/packages/$(pypi.$*)/$*.whl >$@
my-pydeps.layer.tar: $(tools/ocibuild) $(patsubst pypi.%,pypi-downloads/%.whl.layer.tar,$(filter pypi.%,$(.VARIABLES)))
	$(tools/ocibuild) layer squash $(filter %.layer.tar,$^) >$@

my-go.layer.tar: $(tools/ocibuild) $(tools/write-ifchanged) FORCE
	$(tools/ocibuild) layer gobuild ./cmd/go-program-that-call-python | $(tools/write-ifchanged) $@

my-code.layer.tar: $(tools/ocibuild) my-go.layer.tar python/mypackage.whl.layer.tar
	$(tools/ocibuild) layer squash $(filter %.layer.tar,$^) >$@
```

### `ko`

The [`ko`][] tool "Dockerifies" a Go program.  `ko` can also do some
cool stuff with Kubernetes manifests making use of the resulting
image, but we're just going to look at the building-an-image part of
`ko`'s functionality.

The following `ko` recipe...

```bash
docker tag "$(ko publish --local ./cmd/go-program)" docker.io/datawire/ocibuild-example:latest
docker push docker.io/datawire/ocibuild-example:latest
```

...is more-or-less equivalent to the following `ocibuild` recipe:

```bash
ocibuild image build \
  --tag=docker.io/datawire/ocibuild-example:latest \
  --config.Cmd=/usr/local/bin/go-program \
  --base=<(crane pull gcr.io/distroless/static:nonroot) \
  <(ocibuild layer gobuild ./cmd/go-program) \
  | docker load
docker push docker.io/datawire/ocibuild-example:latest
```

[`crane`]: https://pkg.go.dev/github.com/google/go-containerregistry/cmd/crane
[`ko`]: https://github.com/google/ko
[Emissary]: https://github.com/emissary-ingress/emissary
[`./userdocs/`]: ./userdocs/
