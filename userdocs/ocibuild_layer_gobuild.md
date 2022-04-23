## ocibuild layer gobuild

Create a layer of Go binaries

### Synopsis

Works more or less like `go build`.  Passes through env-vars (except for GOOS and GOARCH; naturally those need to be set to reflect the target layer).  Use GOFLAGS to pass in extra flags.

When directing stdout to a file, the timestamps within the resulting layer file will be the current time (clamped by SOURCE_DATE_EPOCH).  If SOURCE_DATE_EPOCH is not set, this may result in unnecessary layer changes; to prevent this, use the --output=FILENAME flag, which avoids updating the layer file if the only changes are timestamps.

```
ocibuild layer gobuild [flags] PACKAGES... >OUT_LAYERFILE
```

### Options

```
  -h, --help              help for gobuild
  -o, --output FILENAME   Write the layer to FILENAME, rather than stdout.  Using this rather than directing stdout to a file may prevent unnescessary timestamp bumps.
```

### SEE ALSO

* [ocibuild layer](ocibuild_layer.md)	 - Manipulate individual layers for use in an image

