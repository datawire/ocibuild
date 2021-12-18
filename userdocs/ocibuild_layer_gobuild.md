## ocibuild layer gobuild

Create a layer from a directory

### Synopsis

Works more or less like `go build`.  Passes through env-vars (except for GOOS and GOARCH; naturally those need to be set to reflect the target layer).  Use GOFLAGS to pass in extra flags.

```
ocibuild layer gobuild [flags] PACKAGES... >OUT_LAYERFILE
```

### Options

```
  -h, --help   help for gobuild
```

### SEE ALSO

* [ocibuild layer](ocibuild_layer.md)	 - Manipulate individual layers for use in an image

