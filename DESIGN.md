

github.com/google/go-containerregistry

go-containerregistry uses
`github.com/google/go-containerregistry/pkg/logs` for logging

* Commands:
  - `crane`: 
    + `./cmd/crane`
    + `./pkg/crane`
  - `gcrane`
    + `./cmd/gcrane`
    + `./pkg/gcrane`
  - `registry`
    + `./cmd/registry`
    + `./pkg/registry`
* Core libraries
  - `./pkg/v1` deals with OCI v1; in particular, the "[image-spec][]"
    and "[distribution-spec][]"; the "[runtime-spec][]" is
    out-of-scope.
  - `./pkg/legacy` deals with [Docker image spec v1][].
* Supplementary libraries
  - `./pkg/logs` - `./pkg/v1` uses this for logging; there are
    `logs.Warn`, `logs.Progress`, and `logs.Debug` global variables
    which are `*log.Logger` objects.
  - `./pkg/authn`
  - `./pkg/name`

[image-spec]: https://github.com/opencontainers/image-spec
[distribution-spec]: https://github.com/opencontainers/distribution-spec
[runtime-spec]: https://github.com/opencontainers/runtime-spec
[Docker image spec v1]: https://github.com/moby/moby/blob/master/image/spec/v1.md
