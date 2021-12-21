## ocibuild layer dir

Create a layer from a directory

```
ocibuild layer dir [flags] IN_DIRNAME >OUT_LAYERFILE
```

### Options

```
  -h, --help                  help for dir
      --prefix PREFIX         Add a PREFIX to the filenames in the directory, should be forward-slash separated and should be absolute but NOT starting with a slash.  For example, "usr/local/bin".
      --prefix-gid int        The numeric group ID of the --prefix directory
      --prefix-gname string   The symbolic group name of the --prefix directory (default "root")
      --prefix-uid int        The numeric user ID of the --prefix directory
      --prefix-uname string   The symbolic user name of the --prefix directory (default "root")
```

### SEE ALSO

* [ocibuild layer](ocibuild_layer.md)	 - Manipulate individual layers for use in an image

