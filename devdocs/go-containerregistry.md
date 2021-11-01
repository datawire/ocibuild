# Overview of go-containerregistry

`layertool` makes uses of the `github.com/google/go-containerregistry`
library, and pairs well with the `crane` CLI tool that is part of
go-containerregistry.

go-containerregistry is a little unapproachable, but if you're working
on layertool, you'll want to be familiar with go-containerregistry.
So this document attempts to be just enough of an introduction to
point you in the right direction.

## Common

go-containerregistry uses
`github.com/google/go-containerregistry/pkg/logs` for logging

## Packages

* Commands:

  - `crane`: A tool for interacting with remote images and registries.
    + `./cmd/crane`
    + `./pkg/crane`

  - `gcrane`: A superset of `crane`, with additional subcommands that
    are specific to gcr.io.
    + `./cmd/gcrane`
    + `./pkg/gcrane`

  - `registry`: A simple registry, for use in testing.
    + `./cmd/registry`
    + `./pkg/registry`

* Core libraries

  - `./pkg/v1` deals with OCI v1; in particular, the "[image-spec][]"
    and "[distribution-spec][]"; the "[runtime-spec][]" is
    out-of-scope.

    Ways that images are represented:

     + `./pkg/v1/daemon`: In a running Docker daemon
     + `./pkg/v1/remote`: In a remote Docker registry
        - `./pkg/v1/remote/transport`
     + `./pkg/v1/layout`: As a directory on a filesystem
     + `./pkg/v1/tarball`: As a tarball

    Plus some utilities for testing:

     + `./pkg/v1/fake`
     + `./pkg/v1/random`

    Utilities:

     + `./pkg/v1/empty`: `empty.Image` is the `FROM scratch` image
     + `./pkg/v1/cache`: Layer cache

    Other:

     + `./pkg/v1/google`
     + `./pkg/v1/match`
     + `./pkg/v1/mutate`
     + `./pkg/v1/partial`
     + `./pkg/v1/static`
     + `./pkg/v1/stream`
     + `./pkg/v1/types`
     + `./pkg/v1/validate`

  - `./pkg/legacy` deals with [Docker image spec v1][].

     + `./pkg/legacy/tarball`

* Supplementary libraries

  - `./pkg/logs` - `./pkg/v1` uses this for logging; there are
    `logs.Warn`, `logs.Progress`, and `logs.Debug` global variables
    which are `*log.Logger` objects.

  - `./pkg/authn`

     + `./pkg/authn/k8schain`

  - `./pkg/name`

[image-spec]: https://github.com/opencontainers/image-spec
[distribution-spec]: https://github.com/opencontainers/distribution-spec
[runtime-spec]: https://github.com/opencontainers/runtime-spec
[Docker image spec v1]: https://github.com/moby/moby/blob/master/image/spec/v1.md
