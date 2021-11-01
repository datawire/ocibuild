# layertool

`layertool` is a command-line tool for manipulating Docker image
layers as files.  It pairs well with the [`crane`][] tool for
interacting with remote Docker images and registries.

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
  <(layertool image \
      --base=<(crane pull docker.io/alpine:latest /dev/stdout) \
      <(layertool squash \
          <(layertool wheel <(curl https://files.pythonhosted.org/packages/af/f4/524415c0744552cce7d8bf3669af78e8a069514405ea4fcbd0cc44733744/urllib3-1.26.7-py2.py3-none-any.whl)) \
          <(layertool wheel <(curl https://files.pythonhosted.org/packages/69/bf/f0f194d3379d3f3347478bd267f754fc68c11cbf2fe302a6ab69447b1417/beautifulsoup4-4.10.0-py3-none-any.whl))) \
      <(layertool squash \
          <(layertool wheel ./python/mypackage.whl) \
          <(layertool go ./cmd/go-program-that-calls-python))) \
  docker.io/datawire/layertool-example:latest
```

Now, in actual use you probably wouldn't want to make as heavy use of
process substitution, and instead cache things to files, in order to
better support incremental builds.  Also, working with pipes instead
of regular files uses more memory, because the whole file must be kept
in memory after reading it, rather than being able to seek around on
disk.

### `ko`

The [`ko`][] tool "Dockerifies" a Go program.  `ko` can also do some
cool stuff with Kubernetes manifests making use of the resulting
image, but we're just going to look at the building-an-image part of
`ko`'s functionality.

```bash
docker tag "$(ko publish --local ./cmd/go-program)" docker.io/datawire/layertool-example:latest
docker push docker.io/datawire/layertool-example:latest
```

```bash
layertool image \
  --base=<(crane pull gcr.io/distroless/static:nonroot) \
  <(layertool go ./cmd/go-program) \
  | docker load
docker push docker.io/datawire/layertool-example:latest
```

[`crane`]: https://pkg.go.dev/github.com/google/go-containerregistry/cmd/crane
[`ko`]: https://github.com/google/ko
[Emissary]: https://github.com/emissary-ingress/emissary
