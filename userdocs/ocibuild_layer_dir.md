## ocibuild layer dir

Create a layer from a directory

```
ocibuild layer dir [flags] IN_DIRNAME >OUT_LAYERFILE
```

### Options

```
      --chown-gid GID         Force the numeric group ID of read files to be GID; use a value <0 to use the actual GID (default -1)
      --chown-gname gname     Force symbolic group name of the read files to be gname; an empty value uses the actual group name (default "root")
      --chown-uid UID         Force the numeric user ID of read files to be UID; a value of <0 uses the actual UID (default -1)
      --chown-uname uname     Force symbolic user name of the read files to be uname; an empty value uses the user name
  -h, --help                  help for dir
      --prefix PREFIX         Add a PREFIX to the filenames in the directory, should be forward-slash separated and should be absolute but NOT starting with a slash.  For example, "usr/local/bin".
      --prefix-gid int        The numeric group ID of the --prefix directory
      --prefix-gname string   The symbolic group name of the --prefix directory (default "root")
      --prefix-uid int        The numeric user ID of the --prefix directory
      --prefix-uname string   The symbolic user name of the --prefix directory (default "root")
```

### SEE ALSO

* [ocibuild layer](ocibuild_layer.md)	 - Manipulate individual layers for use in an image

